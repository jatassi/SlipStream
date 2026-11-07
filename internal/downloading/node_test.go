package downloading_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/transmission"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

const (
	clientNode     = "Media server"
	clientPassword = "orchard-key"
)

// nodeControl is the Controller as the Node's built-in client and its
// actors fetch from it.
type nodeControl struct {
	*control
	password string
}

func (c nodeControl) ClientSecrets(context.Context, *dlv1.ClientSecretsRequest) (*dlv1.ClientSecretsResponse, error) {
	return &dlv1.ClientSecretsResponse{Secrets: &dlv1.Secrets{Password: c.password}}, nil
}

// external is a release the fake Transmission downloads at 100 bytes a
// second.
type external struct {
	name string
	hash bencode.InfoHash
	id   storage.UUID
}

// composite is a Node's Downloading Role over its built-in client and one
// fake Transmission, with the folders the Node's Storage would refuse.
type composite struct {
	*downloading.Node
	n       *node
	builtin *downloading.Client
	h       *holder
	engine  *checks
	served  faketest.Served
	daemon  faketransmission.Daemon
	rels    map[string]external
	client  *dlv1.Client
	// roots are the library roots the Node holds, and judged each
	// download folder the Node judged with whether it asked for write.
	mu     sync.Mutex
	roots  []string
	judged map[string]bool
}

func (n *node) startComposite(version string, releaseSize int) *composite {
	n.t.Helper()
	t := n.t
	d := faketransmission.Daemon{
		Node: clientNode, Version: version, Dir: t.TempDir(), Folder: "/downloads", Rate: 100,
		Username: "admin", Password: clientPassword,
	}
	catalog := faketest.Catalog(t, releaseSize, "orchard")
	c := &composite{n: n, daemon: d, rels: map[string]external{}, judged: map[string]bool{}}
	for _, rel := range catalog.Releases {
		x := external{name: rel.Name, hash: rel.InfoHash, id: storage.NewUUID()}
		c.rels[rel.Name] = x
		n.ctl.answer(x.id, &dlv1.TorrentResponse{Metainfo: rel.Metainfo, ContentType: "Movies"})
	}
	c.served = faketest.InMemory(t, faketransmission.Playing(d), scenario.Scenario{Seed: "transmission"},
		faketest.WithReleases(catalog), faketest.As(clientNode))
	c.client = &dlv1.Client{
		Id: storage.NewUUID().Bytes(), Kind: dlv1.Kind_KIND_TRANSMISSION, Name: "Transmission",
		Address: c.served.Address.URL("/transmission/rpc"), Username: d.Username, Label: clients.DefaultLabel,
		CheckEverySeconds: 10, On: true, SettingsVersion: 1,
		ClientFolders: []*dlv1.ClientFolder{{Client: "/downloads", Node: d.Dir}},
	}

	e, err := n.world.Open(engine.Config{ListenInterface: listen})
	if err != nil {
		t.Fatal(err)
	}
	c.engine = &checks{Engine: e}
	log := slog.New(slog.DiscardHandler)
	c.builtin = downloading.NewClient(downloading.Config{
		Engine: c.engine, Dir: n.dir, Listen: listen, Timings: timings, RetryMin: time.Second, RetryMax: time.Minute,
		Rand: rand.New(rand.NewPCG(1, 2)), ProvesPort: downloading.PublicPeer(false), Log: log, //nolint:gosec // seeded on purpose so tests replay
		Volume: n.volume,
	})
	c.Node = downloading.NewNode(c.builtin, downloading.NodeConfig{
		HTTP: c.served.Client, Timings: timings, Folder: c.judge,
		Rand: rand.New(rand.NewPCG(3, 4)), Log: log, //nolint:gosec // seeded on purpose so tests replay
	})
	c.h = &holder{intents: make(chan role.Snapshot[*dlv1.Intent])}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = c.builtin.Loop(ctx) })
	wg.Go(func() {
		if err := c.Run(n.local, nodeControl{control: n.ctl, password: clientPassword})(ctx, c.h); err != nil {
			t.Errorf("run: %v", err)
		}
	})
	t.Cleanup(func() {
		if err := c.builtin.Shutdown(n.local)(context.Background()); err != nil {
			t.Errorf("shutdown: %v", err)
		}
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	synctest.Wait()
	return c
}

// judge is the Node's Storage judging a client's download folder: one
// inside a root it holds is refused, as is one that isn't there.
func (c *composite) judge(_ context.Context, path string, write bool) (storing.CheckResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.judged[path] = write
	for _, root := range c.roots {
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
			return storing.CheckResult{Path: path, Refusal: &storing.Refusal{Reason: storing.ReasonInsideRoot}}, nil
		}
	}
	if _, err := os.Stat(path); err != nil {
		return storing.CheckResult{Path: path, Refusal: &storing.Refusal{Reason: storing.ReasonNotFound}}, nil
	}
	return storing.CheckResult{Path: path}, nil
}

func (c *composite) holdRoots(roots ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roots = roots
}

// judgedWith is whether the Node judged path, and asked for write.
func (c *composite) judgedWith(path string) (write, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	write, ok = c.judged[path]
	return write, ok
}

// orchard is orchard's Torrent for the fake Transmission, started.
func (c *composite) orchard(order int64) *dlv1.Torrent {
	rel := c.rels["orchard"]
	return &dlv1.Torrent{Id: rel.id.Bytes(), InfoHash: rel.hash[:], Wanted: dlv1.Wanted_WANTED_STARTED, Order: order, ClientId: c.client.GetId()}
}

// send hands the Node an intent with ts, and the fake Transmission among
// its clients unless without.
func (c *composite) send(withClient bool, ts ...*dlv1.Torrent) {
	s := live(ts...)
	if withClient {
		s.Intent.Clients = []*dlv1.Client{c.client}
	}
	c.h.intents <- s
	synctest.Wait()
}

// inClient is whether the fake Transmission lists the release, as a
// session of the test's own sees it.
func (c *composite) inClient(t *testing.T, name string) bool {
	t.Helper()
	a, err := clients.ParseAddress(c.client.GetAddress(), clients.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	s := clients.Session{
		HTTP: c.served.Client, Address: a, Username: c.daemon.Username, Secrets: clients.Secrets{Password: clientPassword},
		Call: 30 * time.Second, Log: slog.New(slog.DiscardHandler),
	}
	h := c.rels[name].hash
	got, err := transmission.Open(s).Poll(t.Context(), []bencode.InfoHash{h})
	if err != nil {
		t.Fatal(err)
	}
	_, ok := got[h]
	return ok
}

func (c *composite) polls() []*dlv1.ClientPoll {
	var out []*dlv1.ClientPoll
	for _, r := range c.h.reported() {
		if p := r.GetClient(); p != nil {
			out = append(out, p)
		}
	}
	return out
}

// rows is how many torrents the Node keeps of its external clients in
// local.db.
func (n *node) rows(t *testing.T) int {
	t.Helper()
	var count int
	if err := n.local.QueryRowContext(t.Context(), "SELECT count(*) FROM downloading_external").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func inEachTransmission(t *testing.T, test func(t *testing.T, version string)) {
	for _, v := range faketransmission.Versions {
		t.Run(v, func(t *testing.T) { synctest.Test(t, func(t *testing.T) { test(t, v) }) })
	}
}

// TestTheIntentSplitsByClient sends one intent with a torrent for the
// built-in client and one for the fake Transmission. Each lands in its own
// client, and the built-in engine never sees the external one.
func TestTheIntentSplitsByClient(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Meadow.mkv")
		n.world.Payload(f.hash, f.payload)
		mine := n.hold(f)
		c := n.startComposite(version, 1000)
		c.send(true, f.started(mine, 1), c.orchard(2))
		time.Sleep(30 * time.Second)
		synctest.Wait()

		if got := len(c.engine.discovers()); got != 1 {
			t.Errorf("the built-in engine was handed %d torrents, want its own one", got)
		}
		if !c.inClient(t, "orchard") {
			t.Fatal("the fake Transmission doesn't list orchard")
		}
		orchard := c.rels["orchard"].hash
		if _, err := c.builtin.Path(orchard, 0); !errors.Is(err, downloading.ErrNotPresent) {
			t.Errorf("the built-in client's Path for orchard = %v, want it not present", err)
		}
		want := filepath.Join(c.daemon.Dir, "Movies", "orchard.mkv")
		if got, err := c.Path(orchard, 0); err != nil || got != want {
			t.Errorf("Path for orchard = %q, %v; want %q", got, err, want)
		}
		if got, err := c.Path(f.hash, 0); err != nil || got != filepath.Join(n.dir, "Movies", f.hash.String(), f.name) {
			t.Errorf("Path for the built-in torrent = %q, %v; want it in the downloads folder", got, err)
		}
		if got := c.Folders(); !slices.Contains(got, c.daemon.Dir) || got[0] != n.dir {
			t.Errorf("Folders = %q, want the downloads folder first and the client's folder among them", got)
		}
	})
}

// TestInspectActFreesAndTheSeamsRouteByTheClientThatHoldsEach asks about
// a built-in torrent, an external one and one no client holds in one call
// each, and every answer comes back in the order asked.
func TestInspectActFreesAndTheSeamsRouteByTheClientThatHoldsEach(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Meadow.mkv")
		n.world.Payload(f.hash, f.payload)
		mine := n.hold(f)
		c := n.startComposite(version, 1000)
		c.send(true, f.started(mine, 1), c.orchard(2))
		time.Sleep(30 * time.Second)
		synctest.Wait()
		theirs, nobody := c.rels["orchard"].id, storage.NewUUID()

		got, err := c.Inspect(t.Context(), &dlv1.InspectRequest{Id: theirs.Bytes()})
		if err != nil || len(got.GetFiles()) != 1 || got.GetFiles()[0].GetDone() != 1000 {
			t.Errorf("Inspect orchard = %v, %v; want its one file done", got, err)
		}
		if got, err := c.Inspect(t.Context(), &dlv1.InspectRequest{Id: mine.Bytes()}); err != nil || len(got.GetFiles()) != 1 || got.GetFiles()[0].GetDone() != size {
			t.Errorf("Inspect the built-in torrent = %v, %v; want its one file done", got, err)
		}

		ids := [][]byte{theirs.Bytes(), nobody.Bytes(), mine.Bytes()}
		c.actsInOrder(t, ids, []dlv1.Refusal{dlv1.Refusal_REFUSAL_UNSPECIFIED, dlv1.Refusal_REFUSAL_NOT_HELD, dlv1.Refusal_REFUSAL_UNSPECIFIED})
		c.freesInOrder(t, ids, []int64{1000, 0, size})

		path, err := c.Open(source.Torrent{Hash: c.rels["orchard"].hash, File: 0})
		if err != nil || path != filepath.Join(c.daemon.Dir, "Movies", "orchard.mkv") {
			t.Errorf("Open orchard = %q, %v; want its file in the client's folder", path, err)
		}
		if _, err := c.Open(source.Torrent{Hash: f.hash, File: 0}); err != nil {
			t.Errorf("Open the built-in torrent: %v", err)
		}
	})
}

// actsInOrder rechecks ids in one call and wants each answered in turn,
// refused as refused says.
func (c *composite) actsInOrder(t *testing.T, ids [][]byte, refused []dlv1.Refusal) {
	t.Helper()
	acted, err := c.Act(t.Context(), &dlv1.ActRequest{Act: dlv1.Act_ACT_RECHECK, Ids: ids})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range acted.GetResults() {
		if !bytes.Equal(r.GetId(), ids[i]) || r.GetRefused() != refused[i] {
			t.Errorf("Act answer %d = %v, want %x refused %v", i, r, ids[i], refused[i])
		}
	}
}

// freesInOrder asks what deleting ids would free in one call and wants
// each answered in turn, freeing freed.
func (c *composite) freesInOrder(t *testing.T, ids [][]byte, freed []int64) {
	t.Helper()
	res, err := c.Frees(t.Context(), &dlv1.FreesRequest{Ids: ids})
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range res.GetFreed() {
		if !bytes.Equal(f.GetId(), ids[i]) || f.GetBytes() != freed[i] {
			t.Errorf("Frees answer %d = %v, want %x freeing %d", i, f, ids[i], freed[i])
		}
	}
}

// TestAnActorStartsAndStopsWithItsClient adds the fake Transmission to the
// intent, then drops it. The Node polls it while it is there, and forgets
// its torrents once it isn't, leaving them in the client.
func TestAnActorStartsAndStopsWithItsClient(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.send(true, c.orchard(1))
		time.Sleep(30 * time.Second)
		synctest.Wait()
		if polls := c.polls(); len(polls) == 0 || polls[len(polls)-1].GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Fatalf("polls = %v, want the client responding", polls)
		}
		if got := n.rows(t); got != 1 {
			t.Fatalf("the Node keeps %d torrents of the client, want 1", got)
		}

		c.send(false)
		polled := len(c.polls())
		time.Sleep(time.Minute)
		synctest.Wait()

		if got := len(c.polls()); got != polled {
			t.Errorf("the Node polled %d more times after the client left its intent, want none", got-polled)
		}
		if got := n.rows(t); got != 0 {
			t.Errorf("the Node keeps %d torrents of a client it no longer runs, want none", got)
		}
		if !c.inClient(t, "orchard") {
			t.Error("orchard left the fake Transmission, want it kept there")
		}
		if _, err := c.Path(c.rels["orchard"].hash, 0); !errors.Is(err, downloading.ErrNotPresent) {
			t.Errorf("Path for orchard = %v, want it not present", err)
		}
		if got := c.Folders(); slices.Contains(got, c.daemon.Dir) {
			t.Errorf("Folders = %q, want the client's folder gone", got)
		}
	})
}

func (c *composite) test(t *testing.T, draft *dlv1.Client, password string) *dlv1.TestClientResponse {
	t.Helper()
	res, err := c.TestClient(t.Context(), &dlv1.TestClientRequest{Client: draft, Secrets: &dlv1.Secrets{Password: password}})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestTestClientSignsInAndJudgesTheFolderAsTheNodeSeesIt tests drafts of
// the fake Transmission no intent names: one that passes, one with the
// wrong password, one whose folder lies inside a library root, and one
// whose folder no Client folder pairs.
func TestTestClientSignsInAndJudgesTheFolderAsTheNodeSeesIt(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)

		passed := c.test(t, c.client, clientPassword).GetPassed()
		if passed.GetProduct() != "Transmission" || !strings.HasPrefix(passed.GetVersion(), version+".") || passed.GetNodeFolder() != c.daemon.Dir || len(passed.GetWarnings()) != 0 {
			t.Errorf("Test = %v, want Transmission %s at %s with no warnings", passed, version, c.daemon.Dir)
		}
		if write, ok := c.judgedWith(c.daemon.Dir); !ok || write {
			t.Errorf("the Node judged its folder: %v, asking for write %v; want it judged read-only, as Transmission deletes its own files", ok, write)
		}

		if got := c.test(t, c.client, "pear").GetFailed(); got.GetCause() != dlv1.Cause_CAUSE_WRONG_CREDENTIALS {
			t.Errorf("Test with the wrong password failed with %v, want wrong credentials", got)
		}

		c.holdRoots(filepath.Dir(c.daemon.Dir))
		if got := c.test(t, c.client, clientPassword).GetFailed(); got.GetCause() != dlv1.Cause_CAUSE_FOLDER_IN_ROOT || got.GetFolder() != c.daemon.Dir {
			t.Errorf("Test with the folder in a root failed with %v, want the folder in a root", got)
		}
		c.holdRoots()

		unpaired := proto.CloneOf(c.client)
		unpaired.ClientFolders = []*dlv1.ClientFolder{{Client: "/elsewhere", Node: c.daemon.Dir}}
		if got := c.test(t, unpaired, clientPassword).GetFailed(); got.GetCause() != dlv1.Cause_CAUSE_FOLDER_UNREADABLE || got.GetFolder() != "/downloads" {
			t.Errorf("Test with no Client folder for /downloads failed with %v, want the folder unreadable", got)
		}
	})
}

// TestCheckClientsAnswersOnceThePollWithItsTokenIsInTheOutbox checks the
// fake Transmission now, long before its next poll is due.
func TestCheckClientsAnswersOnceThePollWithItsTokenIsInTheOutbox(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.client.CheckEverySeconds = 3600
		c.send(true)
		token := []byte("check-1")
		if _, err := c.CheckClients(t.Context(), &dlv1.CheckClientsRequest{Ids: [][]byte{c.client.GetId()}, Check: token}); err != nil {
			t.Fatal(err)
		}
		polls := c.polls()
		if len(polls) != 2 || !bytes.Equal(polls[1].GetCheck(), token) || polls[1].GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Errorf("polls = %v, want the first and then one carrying %q", polls, token)
		}
	})
}
