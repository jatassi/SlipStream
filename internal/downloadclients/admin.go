package downloadclients

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// Draft is an external client as an Admin enters it, parsed at the API
// boundary.
type Draft struct {
	Node role.NodeID
	Kind clients.Kind
	// Name is the kind's name when empty.
	Name     string
	Address  clients.Address
	Username string
	Secrets  clients.Secrets
	Label    clients.Label
	// Folder is as the client sees it, "" for the client's default.
	Folder  string
	Folders clients.Folders
	Daemon  *clients.Daemon
	// CheckEvery and ImportDelay are at their defaults when zero.
	CheckEvery, ImportDelay setting.Stored[int64]
	On                      bool
}

func (d Draft) external(id ClientID) External {
	e := External{
		ID: id, Node: d.Node, Kind: d.Kind, Name: d.Name, Address: d.Address, Username: d.Username, Label: d.Label,
		Folder: d.Folder, Folders: d.Folders, Daemon: d.Daemon, On: d.On, Version: 1,
	}
	if e.Name == "" {
		e.Name = d.Kind.Facts().Name
	}
	return e.withCheckEvery(d.CheckEvery).withImportDelay(d.ImportDelay)
}

func (e External) withCheckEvery(s setting.Stored[int64]) External {
	e.checkEvery = s.Override()
	e.CheckEvery = CheckEvery.Value(e.checkEvery)
	return e
}

func (e External) withImportDelay(s setting.Stored[int64]) External {
	e.importDelay = s.Override()
	e.ImportDelay = ImportDelay.Value(e.importDelay)
	return e
}

// Edit is a change to a saved client: nil leaves a field.
type Edit struct {
	Node *role.NodeID
	// Kind changes how the saved address and label read when the edit
	// leaves them.
	Kind *clients.Kind
	// Name set to "" is the kind's name.
	Name    *string
	Address *clients.Address
	// Username set to "" signs in with none.
	Username *string
	// Password and APIKey replace the saved secret. The API sends nil for
	// one sent empty, which keeps the saved value.
	Password, APIKey *string
	Label            *clients.Label
	Folder           *string
	Folders          *clients.Folders
	// Daemon set to nil is Deluge's only daemon.
	Daemon                  **clients.Daemon
	CheckEvery, ImportDelay *setting.Stored[int64]
	On                      *bool
}

// applied is was, holding saved, with e on top at the next settings
// version. A saved address or label that doesn't fit a new kind is a
// *RefusedError.
func (e Edit) applied(was External, saved clients.Secrets) (External, clients.Secrets, error) {
	now, s := was, saved
	if e.Kind != nil && *e.Kind != was.Kind {
		now.Kind = *e.Kind
		var err error
		if now.Address, err = clients.ParseAddress(was.Address.String(), now.Kind); err != nil {
			return External{}, clients.Secrets{}, &RefusedError{Field: clients.FieldAddress, Err: err}
		}
		if now.Label, err = clients.ParseLabel(was.Label.String(), now.Kind); err != nil {
			return External{}, clients.Secrets{}, &RefusedError{Field: clients.FieldLabel, Err: err}
		}
	}
	set(&now.Node, e.Node)
	set(&now.Name, e.Name)
	if now.Name == "" {
		now.Name = now.Kind.Facts().Name
	}
	set(&now.Address, e.Address)
	set(&now.Username, e.Username)
	set(&s.Password, e.Password)
	set(&s.APIKey, e.APIKey)
	set(&now.Label, e.Label)
	set(&now.Folder, e.Folder)
	set(&now.Folders, e.Folders)
	set(&now.Daemon, e.Daemon)
	set(&now.On, e.On)
	if e.CheckEvery != nil {
		now = now.withCheckEvery(*e.CheckEvery)
	}
	if e.ImportDelay != nil {
		now = now.withImportDelay(*e.ImportDelay)
	}
	now.Version++
	now.Saved = SavedSecrets{Password: s.Password != "", APIKey: s.APIKey != ""}
	return now, s, fits(now, s)
}

func set[T any](to, v *T) {
	if v != nil {
		*to = *v
	}
}

// tested is whether going from was to now changes what the Test reads:
// the Node, kind, address, username, a secret, the daemon, the label, the
// folder or the Client folders.
func tested(was, now External, saved, s clients.Secrets) bool {
	return was.Node != now.Node || was.Kind != now.Kind || was.Address.String() != now.Address.String() || was.Username != now.Username ||
		saved != s || !sameDaemon(was.Daemon, now.Daemon) || was.Label != now.Label || was.Folder != now.Folder ||
		!slices.Equal(was.Folders.Pairs(), now.Folders.Pairs())
}

func sameDaemon(a, b *clients.Daemon) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

var (
	// ErrNotTaken is a sign-in field or a daemon the kind doesn't take.
	ErrNotTaken = errors.New("the kind doesn't take this field")
	// ErrNeeded is a sign-in field the kind needs.
	ErrNeeded = errors.New("the kind needs this field")
)

// RefusedError is a field SlipStream can't use.
type RefusedError struct {
	Field clients.Field
	Err   error
}

func (e *RefusedError) Error() string { return string(e.Field) + ": " + e.Err.Error() }
func (e *RefusedError) Unwrap() error { return e.Err }

// fits refuses a sign-in field or a daemon e's kind doesn't take, and a
// sign-in field it needs that s leaves out.
func fits(e External, s clients.Secrets) error {
	f := e.Kind.Facts()
	for _, c := range []struct {
		field clients.Field
		need  clients.Need
		set   bool
	}{
		{clients.FieldUsername, f.SignIn.Username, e.Username != ""},
		{clients.FieldPassword, f.SignIn.Password, s.Password != ""},
	} {
		switch {
		case c.set && c.need == clients.NotTaken:
			return &RefusedError{Field: c.field, Err: ErrNotTaken}
		case !c.set && c.need == clients.Required:
			return &RefusedError{Field: c.field, Err: ErrNeeded}
		}
	}
	switch {
	case s.APIKey != "" && !f.SignIn.APIKey:
		return &RefusedError{Field: clients.FieldAPIKey, Err: ErrNotTaken}
	case e.Daemon != nil && !f.Daemon:
		return &RefusedError{Field: clients.FieldDaemon, Err: ErrNotTaken}
	}
	return nil
}

var ErrNotDownloading = errors.New("the Node doesn't hold Downloading")

// TestReport is what a passing Test of Subject learned.
type TestReport struct {
	Subject Subject
	// Product and Version are the product as it answered: "Transmission",
	// "4.1.1".
	Product, Version string
	// NodeFolder is the download folder as the Node sees it.
	NodeFolder string
	Warnings   []clients.Warning
	// Daemons are the daemons Deluge's web UI lists.
	Daemons []clients.Daemon
}

// TestFailedError is a Test the client or its Node failed on Subject.
type TestFailedError struct {
	Problem clients.Problem
	Subject Subject
	Node    role.NodeID
	// Daemons are the daemons Deluge's web UI lists, for a Test that
	// failed choosing one.
	Daemons []clients.Daemon
}

func (e *TestFailedError) Error() string { return "the Test failed: " + e.Problem.Error() }

// UnavailableError is a Test that couldn't run: Node is out of step, or
// the call didn't reach it.
type UnavailableError struct {
	Node role.NodeID
	Err  error
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("node %s can't run the Test: %v", e.Node, e.Err)
}
func (e *UnavailableError) Unwrap() error { return e.Err }

var errOutOfStep = errors.New("the Node is out of step")

// InUseError refuses turning off, moving or deleting the client Node's
// new Grabs go to.
type InUseError struct{ Node role.NodeID }

func (e *InUseError) Error() string {
	return fmt.Sprintf("node %s's new Grabs go to the client", e.Node)
}

// HasTorrentsError refuses moving or deleting client Name while it holds
// Count Torrents SlipStream tracks.
type HasTorrentsError struct {
	Name   string
	Count  int
	Moving bool
}

func (e *HasTorrentsError) Error() string {
	return fmt.Sprintf("%s holds %d torrents SlipStream tracks", e.Name, e.Count)
}

// External is client id as an Admin reads it. A deleted client is
// ErrNoClient.
func (c *Clients) External(ctx context.Context, id ClientID) (External, error) {
	e, _, err := c.saved(ctx, id)
	return e, err
}

// ClientCard is client id as its card shows it. A deleted client is
// ErrNoClient.
func (c *Clients) ClientCard(ctx context.Context, id ClientID) (ExternalCard, error) {
	var out ExternalCard
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := c.externals(ctx, r)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(all, func(e ExternalCard) bool { return e.ID == id })
		if i < 0 {
			return ErrNoClient
		}
		out = all[i]
		return nil
	})
	return out, err
}

func (c *Clients) saved(ctx context.Context, id ClientID) (External, clients.Secrets, error) {
	var (
		e External
		s clients.Secrets
	)
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err := db.New(r).GetClient(ctx, id)
		switch {
		case errors.Is(err, storage.ErrNoRows):
			return ErrNoClient
		case err != nil:
			return err
		}
		if e, err = externalOf(row, c.cfg.Sealer); err != nil {
			return err
		}
		s, err = unseal(c.cfg.Sealer, id, row.SecretsSealed)
		return err
	})
	return e, s, err
}

// CreateClient tests d on its Node, then saves it in one Write at settings
// version 1 with the Test as its first poll, and touches the Node. It
// fails with *RefusedError, ErrNotDownloading, ErrNameTaken,
// *AddressTakenError, *TestFailedError or *UnavailableError.
func (c *Clients) CreateClient(ctx context.Context, d Draft) (ExternalCard, error) {
	e := d.external(ClientID{UUID: storage.NewUUID()})
	if err := c.drafted(e, d.Secrets); err != nil {
		return ExternalCard{}, err
	}
	if err := free(ctx, c.cfg.Library, e); err != nil {
		return ExternalCard{}, err
	}
	report, err := c.test(ctx, e, d.Secrets)
	if err != nil {
		return ExternalCard{}, err
	}
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := free(ctx, tx, e); err != nil {
			return err
		}
		p, err := rowOf(c.cfg.Sealer, e, d.Secrets)
		if err != nil {
			return err
		}
		now := storage.Now()
		err = db.New(tx).InsertClient(ctx, db.InsertClientParams{
			ID: p.ID, NodeID: p.NodeID, Kind: p.Kind, Name: p.Name, Address: p.Address, AddressKey: p.AddressKey,
			Username: p.Username, SecretsSealed: p.SecretsSealed, Label: p.Label, Folder: p.Folder, ClientFolders: p.ClientFolders,
			DaemonHost: p.DaemonHost, DaemonPort: p.DaemonPort, CheckEverySeconds: p.CheckEverySeconds,
			ImportDelaySeconds: p.ImportDelaySeconds, IsOn: p.IsOn, CreatedAt: now,
		})
		if err != nil {
			return err
		}
		return passed(ctx, tx, e, report, now)
	})
	if err != nil {
		return ExternalCard{}, err
	}
	c.saveDone(e.ID, e.Node)
	return c.ClientCard(ctx, e.ID)
}

// TestDraft tests d on its Node and saves nothing.
func (c *Clients) TestDraft(ctx context.Context, d Draft) (TestReport, error) {
	e := d.external(ClientID{UUID: storage.NewUUID()})
	if err := c.drafted(e, d.Secrets); err != nil {
		return TestReport{}, err
	}
	return c.test(ctx, e, d.Secrets)
}

// drafted refuses e unless its fields fit its kind and its Node holds
// Downloading.
func (c *Clients) drafted(e External, s clients.Secrets) error {
	if err := fits(e, s); err != nil {
		return err
	}
	if !c.cfg.Roster.Holds(role.Downloading, e.Node) {
		return ErrNotDownloading
	}
	return nil
}

// EditClient saves ed on client id in one Write that bumps its settings
// version, testing first, on the Node it will be on, when ed changes what
// the Test reads. The Test uses the saved secrets wherever ed leaves them,
// and a passing Test is the client's poll at the new version. Turning off
// or moving the client its Node's new Grabs go to is *InUseError, and
// moving one that holds Torrents is *HasTorrentsError. It fails as
// CreateClient does, and with ErrNoClient.
func (c *Clients) EditClient(ctx context.Context, id ClientID, ed Edit) (ExternalCard, error) {
	was, saved, err := c.saved(ctx, id)
	if err != nil {
		return ExternalCard{}, err
	}
	now, s, err := ed.applied(was, saved)
	if err != nil {
		return ExternalCard{}, err
	}
	if now.Node != was.Node && !c.cfg.Roster.Holds(role.Downloading, now.Node) {
		return ExternalCard{}, ErrNotDownloading
	}
	err = c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error { return edits(ctx, r, was, now) })
	if err != nil {
		return ExternalCard{}, err
	}
	var report *TestReport
	if tested(was, now, saved, s) {
		r, err := c.test(ctx, now, s)
		if err != nil {
			return ExternalCard{}, err
		}
		report = &r
	}
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := edits(ctx, tx, was, now); err != nil {
			return err
		}
		p, err := rowOf(c.cfg.Sealer, now, s)
		if err != nil {
			return err
		}
		at := storage.Now()
		p.UpdatedAt = at
		n, err := db.New(tx).UpdateClient(ctx, p)
		switch {
		case err != nil:
			return err
		case n == 0:
			return ErrNoClient
		case report == nil:
			return nil
		}
		return passed(ctx, tx, now, *report, at)
	})
	if err != nil {
		return ExternalCard{}, err
	}
	c.saveDone(id, was.Node, now.Node)
	return c.ClientCard(ctx, id)
}

// edits refuses going from was to now when now's name or address is
// another client's, when it turns off or moves the client its Node's new
// Grabs go to, or when it moves a client that holds Torrents.
func edits(ctx context.Context, r storage.Reader[file], was, now External) error {
	if err := free(ctx, r, now); err != nil {
		return err
	}
	moved := now.Node != was.Node
	if !moved && (now.On || !was.On) {
		return nil
	}
	if err := notGrabsTo(ctx, r, was.ID); err != nil || !moved {
		return err
	}
	return untracked(ctx, r, was, true)
}

// notGrabsTo is *InUseError when a Node's new Grabs go to client id.
func notGrabsTo(ctx context.Context, r storage.Reader[file], id ClientID) error {
	node, err := db.New(r).GrabsGoTo(ctx, &id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	}
	return &InUseError{Node: node}
}

// untracked is *HasTorrentsError while e holds Torrents SlipStream tracks.
func untracked(ctx context.Context, r storage.Reader[file], e External, moving bool) error {
	n, err := db.New(r).CountTracked(ctx, &e.ID)
	if err != nil || n == 0 {
		return err
	}
	return &HasTorrentsError{Name: e.Name, Count: int(n), Moving: moving}
}

// TestSaved tests client id with ed's unsaved changes, on the Node it
// would be on, using its saved secrets wherever ed leaves them. It saves
// nothing.
func (c *Clients) TestSaved(ctx context.Context, id ClientID, ed Edit) (TestReport, error) {
	was, saved, err := c.saved(ctx, id)
	if err != nil {
		return TestReport{}, err
	}
	now, s, err := ed.applied(was, saved)
	if err != nil {
		return TestReport{}, err
	}
	if !c.cfg.Roster.Holds(role.Downloading, now.Node) {
		return TestReport{}, ErrNotDownloading
	}
	return c.test(ctx, now, s)
}

// DeleteClient soft-deletes client id with its secrets nulled in one
// Write, and touches its Node. The client its Node's new Grabs go to is
// *InUseError, and one a Grab is importing from is *ImportingError. One
// that holds Torrents SlipStream tracks is *HasTorrentsError unless
// stopTracking, which parts each of them client_deleted in the same Write
// and leaves them in the client.
func (c *Clients) DeleteClient(ctx context.Context, id ClientID, stopTracking bool) error {
	var (
		node   role.NodeID
		parted []ID
	)
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		e, err := ExternalOf(ctx, tx, id)
		if err != nil {
			return err
		}
		node = e.Node
		if err := notGrabsTo(ctx, tx, id); err != nil {
			return err
		}
		q := db.New(tx)
		if parted, err = q.ListTracked(ctx, &id); err != nil {
			return err
		}
		if err := c.notImporting(ctx, tx, e, parted); err != nil {
			return err
		}
		now := storage.Now()
		if !stopTracking {
			if err := untracked(ctx, tx, e, false); err != nil {
				return err
			}
		} else if err := q.StopTracking(ctx, db.StopTrackingParams{At: now, ClientID: &id}); err != nil {
			return err
		}
		_, err = q.DeleteClient(ctx, db.DeleteClientParams{DeletedAt: now, ID: id})
		return err
	})
	if err != nil {
		return err
	}
	c.saveDone(id, node)
	c.committed(nil, parted)
	return nil
}

// ImportingError refuses deleting client Name while a Grab imports from
// one of its Torrents.
type ImportingError struct{ Name string }

func (e *ImportingError) Error() string { return "a Grab is importing from " + e.Name }

// notImporting is *ImportingError while a Grab imports from any of ids,
// e's Torrents.
func (c *Clients) notImporting(ctx context.Context, r storage.Reader[file], e External, ids []ID) error {
	uses, err := c.cfg.Uses.Of(ctx, r, ids)
	if err != nil {
		return err
	}
	for _, us := range uses {
		if slices.ContainsFunc(us, func(u Use) bool { return u.Stage == UseImporting }) {
			return &ImportingError{Name: e.Name}
		}
	}
	return nil
}

func (c *Clients) saveDone(id ClientID, nodes ...role.NodeID) {
	nodes = slices.Compact(nodes)
	c.cfg.Touch(nodes...)
	c.clientsChanged([]ClientID{id}, nodes...)
}

// passed records a passing Test of e as its poll at e's settings version.
func passed(ctx context.Context, tx *storage.Tx[file], e External, r TestReport, at storage.Time) error {
	_, err := db.New(tx).SetPoll(ctx, db.SetPollParams{
		PollState: new(string(clients.Responding)), PollSince: at, PollSettingsVersion: new(e.Version),
		ClientVersion: text(r.Version), ID: e.ID, NodeID: e.Node,
	})
	return err
}

// test is the one Test: e's Node's TestClient with e and its secrets s in
// the call.
func (c *Clients) test(ctx context.Context, e External, s clients.Secrets) (TestReport, error) {
	if !c.cfg.Roster.InStep(e.Node) {
		return TestReport{}, &UnavailableError{Node: e.Node, Err: errOutOfStep}
	}
	res, err := c.cfg.Remote(e.Node).TestClient(ctx, &dlv1.TestClientRequest{
		Client: clientOnWire(e), Secrets: &dlv1.Secrets{Password: s.Password, ApiKey: s.APIKey},
	})
	if err != nil {
		if _, notSent := rpc.NotSent(err); notSent || rpc.Lost(err) {
			return TestReport{}, &UnavailableError{Node: e.Node, Err: err}
		}
		return TestReport{}, fmt.Errorf("test client %s on node %s: %w", e.ID, e.Node, err)
	}
	switch a := res.GetAnswer().(type) {
	case *dlv1.TestClientResponse_Failed:
		p, err := problemFromWire(a.Failed)
		if err != nil {
			return TestReport{}, fmt.Errorf("test client %s: %w", e.ID, err)
		}
		return TestReport{}, &TestFailedError{Problem: p, Subject: e.Subject(), Node: e.Node, Daemons: daemonsFromWire(a.Failed.GetDaemons())}
	case *dlv1.TestClientResponse_Passed:
		return reportFromWire(a.Passed, e.Subject())
	}
	return TestReport{}, fmt.Errorf("test client %s: an answer of kind %T", e.ID, res.GetAnswer())
}

func reportFromWire(p *dlv1.Passed, s Subject) (TestReport, error) {
	r := TestReport{Subject: s, Product: p.GetProduct(), Version: p.GetVersion(), NodeFolder: p.GetNodeFolder()}
	for _, w := range p.GetWarnings() {
		got, ok := warningFromWire[w]
		if !ok {
			return TestReport{}, fmt.Errorf("a Test's warning %s", w)
		}
		r.Warnings = append(r.Warnings, got)
	}
	r.Daemons = daemonsFromWire(p.GetDaemons())
	return r, nil
}

func daemonsFromWire(daemons []*dlv1.Daemon) []clients.Daemon {
	var out []clients.Daemon
	for _, d := range daemons {
		out = append(out, clients.Daemon{Host: d.GetHost(), Port: int(d.GetPort())})
	}
	return out
}
