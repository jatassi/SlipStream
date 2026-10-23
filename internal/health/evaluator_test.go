package health_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type (
	nodesPoked   struct{}
	tasksPoked   struct{}
	volumesPoked struct{}
)

var (
	nodesEvent   = live.AdminEvent[nodesPoked]("test.nodes")
	tasksEvent   = live.AdminEvent[tasksPoked]("test.tasks")
	volumesEvent = live.AdminEvent[volumesPoked]("test.volumes")

	media = role.NodeID{UUID: storage.NewUUID()}
)

const mount = `D:\`

type source struct {
	mu     sync.Mutex
	answer health.Answer
	err    error
	calls  int
	check  func(item health.Key) []health.Unreached
}

func (s *source) Answer(context.Context) (health.Answer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.answer, s.err
}

func (s *source) Check(_ context.Context, item health.Key) ([]health.Unreached, error) {
	return s.check(item), nil
}

func (s *source) set(a health.Answer, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.answer, s.err = a, err
}

func (s *source) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type volumes struct {
	mu      sync.Mutex
	list    []health.Volume
	calls   int
	away    bool
	sampled []role.NodeID
}

func (v *volumes) Volumes(context.Context) ([]health.Volume, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls++
	return slices.Clone(v.list), nil
}

func (v *volumes) Sample(_ context.Context, node role.NodeID) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.away {
		return errors.Join(errors.New("dial Media server"), health.ErrUnreachable)
	}
	v.sampled = append(v.sampled, node)
	return nil
}

func (v *volumes) set(free int64, inStep bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.list = []health.Volume{{
		Node: media, NodeName: "Media server", Mount: mount, Name: "D: (Media)",
		Total: setting.TB, Free: free, SampledAt: time.Now(), Live: inStep,
	}}
}

func (v *volumes) count() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls
}

type sink struct {
	mu sync.Mutex
	ts []health.Transition
}

func (s *sink) take(_ context.Context, tx *storage.Tx[storage.Library], ts []health.Transition) error {
	if tx == nil {
		return errors.New("transitions outside a Write")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ts = append(s.ts, ts...)
	return nil
}

func (s *sink) kinds() []health.Kind {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]health.Kind, len(s.ts))
	for i, t := range s.ts {
		out[i] = t.Kind
	}
	return out
}

type publisher struct {
	mu  sync.Mutex
	got [][]string
}

func (p *publisher) Publish(ctx context.Context, v health.Changed) {
	if storage.InWrite(ctx) {
		panic("health.changed published inside a Write")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.got = append(p.got, v.Categories)
}

func (p *publisher) take() [][]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	got := p.got
	p.got = nil
	return got
}

type world struct {
	lib     *storage.DB[storage.Library]
	hub     *live.Hub
	nodes   *source
	tasks   *source
	vols    *volumes
	sink    *sink
	changed *publisher

	mu     sync.Mutex
	names  map[role.NodeID]string
	roster chan struct{}
}

func (w *world) Changed() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.roster
}

// remove takes node off the roster.
func (w *world) remove(node role.NodeID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.names, node)
	close(w.roster)
	w.roster = make(chan struct{})
}

func newWorld(t *testing.T) *world {
	hub := live.NewHub([]live.Event{nodesEvent, tasksEvent, volumesEvent})
	t.Cleanup(hub.Close)
	w := &world{
		lib:     storagetest.Library(t, health.Schema()),
		hub:     hub,
		nodes:   &source{},
		tasks:   &source{},
		vols:    &volumes{},
		sink:    &sink{},
		changed: &publisher{},
		names:   map[role.NodeID]string{media: "Media server"},
		roster:  make(chan struct{}),
	}
	w.nodes.check = func(health.Key) []health.Unreached { return nil }
	return w
}

func (w *world) open(t *testing.T) *health.Health {
	t.Helper()
	h, err := health.Open(t.Context(), health.Config{
		Library: w.lib,
		Hub:     w.hub,
		Changed: w.changed,
		Storage: health.StorageParts{Volumes: []health.VolumeSource{{Volumes: w.vols.Volumes, Sample: w.vols.Sample, Events: []live.Event{volumesEvent}}}},
		Categories: []health.Category{
			{About: health.About{ID: health.NodesID, Label: "Nodes", Phrase: "Nodes", Events: []live.Event{nodesEvent}}, Source: health.Source{Answer: w.nodes.Answer, Check: w.nodes.Check}},
			{About: health.About{ID: health.DownloadClientsID, Label: "Download Clients", Phrase: "download clients"}, Source: health.Source{Answer: func(context.Context) (health.Answer, error) { return health.Answer{}, nil }}},
			{About: health.About{ID: health.ScheduledTasksID, Label: "Scheduled tasks", Phrase: "scheduled tasks", Events: []live.Event{tasksEvent}}, Source: health.Source{Answer: w.tasks.Answer}},
		},
		Names: func(context.Context, storage.Reader[storage.Library]) (map[role.NodeID]string, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			return maps(w.names), nil
		},
		Roster:      w,
		Transitions: w.sink.take,
		Zone:        time.UTC,
		Timings:     health.DefaultTimings(),
		Log:         slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func maps(m map[role.NodeID]string) map[role.NodeID]string {
	out := map[role.NodeID]string{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func run(t *testing.T, h *health.Health) (stop func()) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	stop = func() {
		cancel()
		<-done
	}
	t.Cleanup(stop)
	return stop
}

func node(level health.Level, text string) health.Item {
	return health.Item{Key: health.KeyOf(media.String()), Level: level, Says: health.Says{Subject: "Media server", Text: text}}
}

func unsure() health.Unsure {
	return health.Unsure{Key: health.KeyOf(media.String()), Says: health.Says{Subject: "Media server", Text: "Media server is reconnecting."}}
}

func items(t *testing.T, r health.Report, id health.ID) []health.Shown {
	t.Helper()
	for _, c := range r.Categories {
		if c.ID == id {
			return c.Items
		}
	}
	t.Fatalf("the report has no category %s", id)
	return nil
}

func evaluate(t *testing.T, h *health.Health, ids ...health.ID) health.Report {
	t.Helper()
	r, err := h.Evaluate(t.Context(), ids...)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestEachChangeIsOneTransition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		h := w.open(t)
		run(t, h)
		raisedAt := time.Now()
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Warn, "Media server is slow to answer.")}}, nil)
		evaluate(t, h, health.NodesID)
		evaluate(t, h, health.NodesID)
		time.Sleep(10 * time.Second)
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Warn, "Media server is still slow.")}}, nil)
		evaluate(t, h, health.NodesID)
		time.Sleep(10 * time.Second)
		worseAt := time.Now()
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		r := evaluate(t, h, health.NodesID)
		got := items(t, r, health.NodesID)
		if len(got) != 1 || !got[0].Since.Equal(raisedAt) || !got[0].ChangedAt.Equal(worseAt) || got[0].Level != health.Fail {
			t.Fatalf("after worse the item is %+v, want fail since %v changed %v", got, raisedAt, worseAt)
		}
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Warn, "Media server is slow to answer.")}}, nil)
		evaluate(t, h, health.NodesID)
		w.nodes.set(health.Answer{}, nil)
		r = evaluate(t, h, health.NodesID)
		want := []health.Kind{health.Raised, health.Worse, health.Better, health.Cleared}
		if got := w.sink.kinds(); !slices.Equal(got, want) {
			t.Fatalf("transitions %v, want %v", got, want)
		}
		cleared := w.sink.ts[3]
		if cleared.Subject != "Media server" || cleared.Text != "Media server is slow to answer." || cleared.Level() != health.Warn {
			t.Fatalf("cleared transition %+v, want the last words at warn", cleared)
		}
		if len(items(t, r, health.NodesID)) != 0 || r.Level != health.None || r.First != nil {
			t.Fatalf("after the clear the report is %+v, want nothing wrong", r)
		}
	})
}

func TestAFailingSourceLeavesItsRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		h := w.open(t)
		run(t, h)
		since := time.Now()
		time.Sleep(time.Second)
		w.nodes.set(health.Answer{}, errors.New("the board is closed"))
		r := evaluate(t, h, health.NodesID)
		got := items(t, r, health.NodesID)
		if len(got) != 1 || !got[0].Since.Equal(since) || got[0].Text != "Media server is offline." || r.Categories[1].Unread == "" {
			t.Fatalf("after a failed answer nodes shows %+v, want the open item in its last words and why it is unread", r.Categories[1])
		}
		w.nodes.set(health.Answer{}, nil)
		evaluate(t, h, health.NodesID)
		if got := w.sink.kinds(); !slices.Equal(got, []health.Kind{health.Raised, health.Cleared}) {
			t.Fatalf("transitions %v, want raised then cleared once the source answers", got)
		}
	})
}

func TestAFailedAnswerIsAskedAgainAfterTheRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.nodes.set(health.Answer{}, errors.New("the board is closed"))
		h := w.open(t)
		run(t, h)
		synctest.Wait()
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		time.Sleep(health.DefaultTimings().Retry)
		synctest.Wait()
		r, err := h.Report(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if c := r.Categories[1]; c.Unread != "" || len(c.Items) != 1 {
			t.Fatalf("a retry after a failed answer shows %+v, want the source read and its item raised", c)
		}
	})
}

func TestAnUnsureKeyKeepsItsRowAndSince(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		h := w.open(t)
		run(t, h)
		since := time.Now()
		time.Sleep(time.Minute)
		w.nodes.set(health.Answer{Unsure: []health.Unsure{unsure()}}, nil)
		got := items(t, evaluate(t, h, health.NodesID), health.NodesID)
		if len(got) != 1 || !got[0].Unsure || got[0].Level != health.Fail || !got[0].Since.Equal(since) || got[0].Text != "Media server is reconnecting." {
			t.Fatalf("unsure item %+v, want it kept at fail since %v in the unsure words", got, since)
		}
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		got = items(t, evaluate(t, h, health.NodesID), health.NodesID)
		if len(got) != 1 || got[0].Unsure || !got[0].Since.Equal(since) {
			t.Fatalf("item %+v, want it sure again since %v", got, since)
		}
		if got := w.sink.kinds(); !slices.Equal(got, []health.Kind{health.Raised}) {
			t.Fatalf("transitions %v, want only the first raise", got)
		}
	})
}

func TestAnEventEvaluatesOnlyItsCategory(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		h := w.open(t)
		run(t, h)
		synctest.Wait()
		nodes, vols := w.nodes.count(), w.vols.count()
		w.tasks.set(health.Answer{Items: []health.Item{{Key: "tidy", Level: health.Warn, Says: health.Says{Subject: "Tidy", Text: "Tidy failed. Run it again.", Action: health.RunAction("tidy")}}}}, nil)
		tasksEvent.Bind(w.hub).Publish(t.Context(), tasksPoked{})
		synctest.Wait()
		if w.nodes.count() != nodes || w.vols.count() != vols {
			t.Fatalf("a tasks event asked nodes %d→%d and storage %d→%d times, want neither", nodes, w.nodes.count(), vols, w.vols.count())
		}
		r, err := h.Report(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got := items(t, r, health.ScheduledTasksID); len(got) != 1 || h.Worst(health.ScheduledTasksID) != health.Warn || h.Worst(health.NodesID) != health.None {
			t.Fatalf("after the event tasks shows %+v, want the warning", got)
		}
		if got := w.changed.take(); !slices.EqualFunc(got, [][]string{{"scheduled-tasks"}}, slices.Equal) {
			t.Fatalf("health.changed %v, want one for scheduled-tasks", got)
		}
	})
}

func TestCheckNowAnswersTheReportItLeft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		item := node(health.Fail, "Media server's downloads folder is read-only.")
		item.Action = health.CheckAction()
		w.nodes.set(health.Answer{Items: []health.Item{item, {Key: "other", Level: health.Warn, Says: health.Says{Subject: "Kitchen", Text: "Kitchen is slow."}}}}, nil)
		w.nodes.check = func(health.Key) []health.Unreached {
			w.nodes.set(health.Answer{}, nil)
			return []health.Unreached{{Key: item.Key, Words: "Kitchen is offline."}}
		}
		h := w.open(t)
		run(t, h)
		for _, c := range []struct {
			id   health.ID
			key  health.Key
			want error
		}{
			{"backups", "", health.ErrNoCategory},
			{health.NodesID, "gone", health.ErrNoItem},
			{health.NodesID, "other", health.ErrNoCheck},
			{health.ScheduledTasksID, "", health.ErrNoCheck},
		} {
			if _, err := h.Check(t.Context(), c.id, c.key); !errors.Is(err, c.want) {
				t.Errorf("Check(%s, %q) = %v, want %v", c.id, c.key, err, c.want)
			}
		}
		checked, err := h.Check(t.Context(), health.NodesID, item.Key)
		if err != nil {
			t.Fatal(err)
		}
		if len(items(t, checked.Report, health.NodesID)) != 0 || len(checked.Unreached) != 1 {
			t.Fatalf("Check now answered %+v, want the fresh report with nothing open and what it couldn't reach", checked)
		}
	})
}

func TestARestartKeepsSince(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		since := time.Now()
		stop := run(t, w.open(t))
		synctest.Wait()
		stop()
		time.Sleep(time.Hour)
		w.nodes.set(health.Answer{Unsure: []health.Unsure{unsure()}}, nil)
		h := w.open(t)
		run(t, h)
		r, err := h.Report(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got := items(t, r, health.NodesID); len(got) != 1 || !got[0].Since.Equal(since) || !got[0].Unsure {
			t.Fatalf("after a restart nodes shows %+v, want the item unsure since %v", got, since)
		}
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		got := items(t, evaluate(t, h, health.NodesID), health.NodesID)
		if len(got) != 1 || !got[0].Since.Equal(since) {
			t.Fatalf("after a restart the item is %+v, want since %v", got, since)
		}
		if got := w.sink.kinds(); !slices.Equal(got, []health.Kind{health.Raised}) {
			t.Fatalf("transitions %v, want the one raise before the restart", got)
		}
	})
}

func amount(t *testing.T, s setting.Space, a setting.Amount) *setting.Stored[setting.Amount] {
	t.Helper()
	v, err := s.Parse(&a)
	if err != nil {
		t.Fatal(err)
	}
	return &v
}

func TestALevelsWriteEvaluatesAndPublishesStorage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(150*setting.GB, true)
		h := w.open(t)
		run(t, h)
		if h.Worst() != health.None {
			t.Fatalf("150 GB free on the defaults is %v, want nothing wrong", h.Worst())
		}
		err := h.SetGlobalLevels(t.Context(), health.PairPatch{Error: amount(t, health.StorageErrorBelow, setting.Amount{N: 10 * setting.GB, Unit: setting.BytesUnit})})
		if err != nil {
			t.Fatal(err)
		}
		if got := w.changed.take(); !slices.EqualFunc(got, [][]string{{"storage"}}, slices.Equal) || h.Worst() != health.None {
			t.Fatalf("a levels write that moves no item published %v, want one for storage", got)
		}
		err = h.SetGlobalLevels(t.Context(), health.PairPatch{Warn: amount(t, health.StorageWarnBelow, setting.Amount{N: 200 * setting.GB, Unit: setting.BytesUnit})})
		if err != nil {
			t.Fatal(err)
		}
		if got := w.changed.take(); !slices.EqualFunc(got, [][]string{{"storage"}}, slices.Equal) {
			t.Fatalf("health.changed %v, want one for storage", got)
		}
		r, _ := h.Report(t.Context())
		got := items(t, r, health.StorageID)
		want := "Free up space on D: (Media) on Media server, or add a library root on another volume. It has 150 GB left."
		if len(got) != 1 || got[0].Level != health.Warn || got[0].Subject != "D: (Media) on Media server" || got[0].Text != want || got[0].Action.Kind() != health.ActionCheck {
			t.Fatalf("storage shows %+v, want a warning saying %q", got, want)
		}

	})
}

func onlyVolume(t *testing.T, h *health.Health) health.ListedVolume {
	t.Helper()
	list, err := h.VolumeList(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Volumes) != 1 || len(list.NotReporting) != 0 {
		t.Fatalf("volumes %+v, want D: alone", list)
	}
	return list.Volumes[0]
}

func storagePublished(t *testing.T, w *world) {
	t.Helper()
	if got := w.changed.take(); !slices.EqualFunc(got, [][]string{{"storage"}}, slices.Equal) {
		t.Fatalf("health.changed %v, want one for storage", got)
	}
}

func TestAVolumesOwnLevelsWinUntilReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(150*setting.GB, true)
		h := w.open(t)
		run(t, h)
		err := h.SetGlobalLevels(t.Context(), health.PairPatch{Warn: amount(t, health.StorageWarnBelow, setting.Amount{N: 200 * setting.GB, Unit: setting.BytesUnit})})
		if err != nil {
			t.Fatal(err)
		}
		w.changed.take()
		err = h.SetVolumeLevels(t.Context(), media, mount, health.PairPatch{Error: amount(t, health.StorageErrorBelow, setting.Amount{N: 300 * setting.GB, Unit: setting.BytesUnit})})
		if refused, ok := errors.AsType[*setting.RangeError](err); !ok || refused.Key != health.StorageErrorBelow.Key {
			t.Fatalf("an error level above the warning = %v, want a RangeError", err)
		}
		if got := w.changed.take(); len(got) != 0 {
			t.Fatalf("a refused write published %v", got)
		}
		err = h.SetVolumeLevels(t.Context(), media, mount, health.PairPatch{Error: amount(t, health.StorageErrorBelow, setting.Amount{N: 16, Unit: setting.PercentUnit})})
		if err != nil {
			t.Fatal(err)
		}
		v := onlyVolume(t, h)
		if v.Level != health.Fail || v.ErrorBelow.IsDefault || !v.WarnBelow.IsDefault ||
			v.WarnBelow.Default != (setting.Amount{N: 200 * setting.GB, Unit: setting.BytesUnit}) || !v.Levels.Error.Own {
			t.Fatalf("volume %+v, want D: in error on its own error level and the global warning", v)
		}
		storagePublished(t, w)

		if err := h.SetVolumeLevels(t.Context(), media, mount, health.PairPatch{Error: &setting.Stored[setting.Amount]{}}); err != nil {
			t.Fatal(err)
		}
		if v := onlyVolume(t, h); !v.ErrorBelow.IsDefault || v.Level != health.Warn {
			t.Fatalf("after Reset the volume is %+v, want it back on the global levels", v)
		}
		if got := w.sink.kinds(); !slices.Equal(got, []health.Kind{health.Raised, health.Worse, health.Better}) {
			t.Fatalf("transitions %v, want raised, worse, better", got)
		}
	})
}

func TestAnOverrideOutlivesItsVolumeUntilItsNodeGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(500*setting.GB, true)
		h := w.open(t)
		run(t, h)
		err := h.SetVolumeLevels(t.Context(), media, mount, health.PairPatch{Warn: amount(t, health.StorageWarnBelow, setting.Amount{N: 10, Unit: setting.PercentUnit})})
		if err != nil {
			t.Fatal(err)
		}
		w.vols.mu.Lock()
		w.vols.list = nil
		w.vols.mu.Unlock()
		list, err := h.VolumeList(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Volumes) != 0 || len(list.NotReporting) != 1 || list.NotReporting[0].NodeName != "Media server" || list.NotReporting[0].Mount != mount {
			t.Fatalf("volumes %+v, want D: listed as not reporting", list)
		}
		if err := h.SetVolumeLevels(t.Context(), media, mount, health.PairPatch{}); !errors.Is(err, health.ErrNoVolume) {
			t.Fatalf("levels for a volume that stopped reporting = %v, want ErrNoVolume", err)
		}
		if err := h.SetVolumeLevels(t.Context(), media, `E:\`, health.PairPatch{}); !errors.Is(err, health.ErrNoVolume) {
			t.Fatalf("levels for a volume nobody reports = %v, want ErrNoVolume", err)
		}
		w.changed.take()
		w.remove(media)
		synctest.Wait()
		evaluate(t, h, health.StorageID)
		if list, _ := h.VolumeList(t.Context()); len(list.NotReporting) != 0 {
			t.Fatalf("after its Node went the override is %+v, want it gone", list.NotReporting)
		}
		if got := w.changed.take(); !slices.EqualFunc(got, [][]string{{"storage"}}, slices.Equal) {
			t.Fatalf("health.changed %v, want one for storage", got)
		}
	})
}

func TestStorageCheckNowSamplesTheVolumesNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(10*setting.GB, false)
		h := w.open(t)
		run(t, h)
		r, _ := h.Report(t.Context())
		got := items(t, r, health.StorageID)
		if len(got) != 1 || got[0].Level != health.Fail || got[0].Text != "Free up space on D: (Media) on Media server, or add a library root on another volume. It had 10 GB left as of 00:00." {
			t.Fatalf("storage shows %+v, want an error in the words of an offline Node", got)
		}
		key := got[0].Key
		w.vols.mu.Lock()
		w.vols.away = true
		w.vols.mu.Unlock()
		checked, err := h.Check(t.Context(), health.StorageID, key)
		if err != nil {
			t.Fatal(err)
		}
		want := []health.Unreached{{Key: key, Words: "Media server is offline, so its figures are as of 00:00."}}
		if !slices.Equal(checked.Unreached, want) || len(items(t, checked.Report, health.StorageID)) != 1 {
			t.Fatalf("Check now on an offline Node = %+v, want %v and the item as it was", checked, want)
		}
		w.vols.mu.Lock()
		w.vols.away = false
		w.vols.mu.Unlock()
		w.vols.set(400*setting.GB, true)
		checked, err = h.Check(t.Context(), health.StorageID, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(checked.Unreached) != 0 || len(items(t, checked.Report, health.StorageID)) != 0 || !slices.Equal(w.vols.sampled, []role.NodeID{media}) {
			t.Fatalf("Check now on storage = %+v after sampling %v, want Media server sampled and nothing wrong", checked, w.vols.sampled)
		}
	})
}

func TestOpenPanicsOnACategoryIDsDoesNotDeclare(t *testing.T) {
	w := newWorld(t)
	defer func() {
		if recover() == nil {
			t.Fatal("Open did not panic")
		}
	}()
	_, _ = health.Open(t.Context(), health.Config{
		Library: w.lib, Hub: w.hub, Changed: w.changed, Zone: time.UTC, Timings: health.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
		Categories: []health.Category{{About: health.About{ID: "backups", Label: "Backups", Phrase: "Backups"}, Source: health.Source{Answer: w.nodes.Answer}}},
	})
}

func TestAVolumeWithNoTotalIsNotJudged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(0, true)
		w.vols.list[0].Total = 0
		h := w.open(t)
		run(t, h)
		if r, _ := h.Report(t.Context()); len(items(t, r, health.StorageID)) != 0 {
			t.Fatalf("a volume reporting no total shows %+v, want nothing", items(t, r, health.StorageID))
		}
	})
}

func TestAnEvaluationThatChangesNothingReadsNoStorage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		w.vols.set(10*setting.GB, true)
		w.nodes.set(health.Answer{Items: []health.Item{node(health.Fail, "Media server is offline.")}}, nil)
		h := w.open(t)
		run(t, h)
		synctest.Wait()
		if err := w.lib.Close(); err != nil {
			t.Fatal(err)
		}
		for _, c := range evaluate(t, h, health.StorageID, health.NodesID, health.ScheduledTasksID).Categories {
			if c.Unread != "" {
				t.Fatalf("an evaluation that changes nothing left %s unread with the library closed: %s", c.ID, c.Unread)
			}
		}
		w.nodes.set(health.Answer{}, nil)
		for _, c := range evaluate(t, h, health.NodesID).Categories {
			if c.ID == health.NodesID && c.Unread == "" {
				t.Fatal("clearing an item with the library closed left nodes read, want why it couldn't be written")
			}
		}
	})
}
