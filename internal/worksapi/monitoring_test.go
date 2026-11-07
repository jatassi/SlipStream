package worksapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// stacked is a record of boxes things each, keyed b<n> and b<n>t<m>, then
// a spare box of spares things keyed s and st<m>.
func stacked(boxes, things, spares int) contenttypetest.Record {
	r := contenttypetest.Record{Tree: works.Fetched{Label: "The Stack"}}
	for b := 1; b <= boxes; b++ {
		box := works.Entry{Key: works.Key(fmt.Sprint("b", b)), Label: fmt.Sprint("Box ", b)}
		for m := 1; m <= things; m++ {
			box.Children = append(box.Children, works.Entry{Key: works.Key(fmt.Sprintf("b%dt%d", b, m)), Label: "Thing"})
		}
		r.Tree.Children = append(r.Tree.Children, box)
	}
	if spares > 0 {
		spare := works.Entry{Key: "s", Label: "Spare", Aside: true}
		for m := 1; m <= spares; m++ {
			spare.Children = append(spare.Children, works.Entry{Key: works.Key(fmt.Sprint("st", m)), Label: "Spare thing"})
		}
		r.Tree.Children = append(r.Tree.Children, spare)
	}
	return r
}

type watched struct {
	ID            string `json:"id"`
	MonitorNew    bool   `json:"monitorNew"`
	MonitorAsides bool   `json:"monitorAsides"`
	Levels        []struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	} `json:"levels"`
	Items []struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		Monitored bool   `json:"monitored"`
	} `json:"items"`
}

func (s *server) watched(t *testing.T, id string) watched {
	t.Helper()
	code, body := s.call(t, http.MethodGet, "/api/v1/works/"+id, "")
	var w watched
	if err := json.Unmarshal([]byte(body), &w); code != http.StatusOK || err != nil {
		t.Fatalf("getWork %s = %d %s, %v", id, code, body, err)
	}
	return w
}

// on is the keys of w's monitored Items, then its rule.
func (w watched) on() string {
	var keys []string
	for _, it := range w.Items {
		if it.Monitored {
			keys = append(keys, it.Key)
		}
	}
	slices.Sort(keys)
	return fmt.Sprintf("%s new=%t asides=%t", strings.Join(keys, " "), w.MonitorNew, w.MonitorAsides)
}

func (w watched) level(key string) string {
	for _, l := range w.Levels {
		if l.Key == key {
			return l.ID
		}
	}
	panic("no level " + key)
}

func (w watched) item(key string) string {
	for _, it := range w.Items {
		if it.Key == key {
			return it.ID
		}
	}
	panic("no Item " + key)
}

func (s *server) put(t *testing.T, key string, r contenttypetest.Record, body string) string {
	t.Helper()
	s.fx.Put(works.Key(key), r)
	return s.add(t, `{"contentType":"fixture","key":"`+key+`"`+body+`}`).Work.ID
}

type monitoringReply struct {
	Outcomes []struct {
		WorkID  string `json:"workId"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
		Start   int    `json:"start"`
		Stop    int    `json:"stop"`
		Counts  *struct {
			Asides  int `json:"asides"`
			Choices []struct {
				Preset   string `json:"preset"`
				Monitors int    `json:"monitors"`
			} `json:"choices"`
		} `json:"counts"`
		Problem *struct {
			Code string `json:"code"`
		} `json:"problem"`
	} `json:"outcomes"`
}

// lines is each outcome as "<outcome> +start -stop", in order, with its
// counts as preset=n when the reply carries them.
func (m monitoringReply) lines() string {
	var out []string
	for _, o := range m.Outcomes {
		line := fmt.Sprintf("%s +%d -%d", o.Outcome, o.Start, o.Stop)
		switch {
		case o.Reason != "":
			line = o.Outcome + " " + o.Reason
		case o.Problem != nil:
			line = o.Outcome + " " + o.Problem.Code
		}
		if o.Counts != nil {
			var b strings.Builder
			for _, c := range o.Counts.Choices {
				fmt.Fprintf(&b, " %s=%d", c.Preset, c.Monitors)
			}
			line += b.String() + fmt.Sprintf(" asides=%d", o.Counts.Asides)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func (s *server) setMonitoring(t *testing.T, body string) monitoringReply {
	t.Helper()
	code, reply := s.call(t, http.MethodPut, "/api/v1/works/monitoring", body)
	var m monitoringReply
	if err := json.Unmarshal([]byte(reply), &m); code != http.StatusOK || err != nil {
		t.Fatalf("setWorksMonitoring %s = %d %s, %v", body, code, reply, err)
	}
	return m
}

func choose(preset string, ids ...string) string {
	return `{"workIds":["` + strings.Join(ids, `","`) + `"],"preset":"` + preset + `","includeAsides":false`
}

func TestAddAppliesThePreset(t *testing.T) {
	for _, c := range []struct {
		today func() time.Time
		body  string
		want  string
	}{
		{time.Now, `,"preset":"all"`, "b1t1 b1t2 b2t1 b2t2 new=true asides=false"},
		{time.Now, `,"preset":"all","includeAsides":true`, "b1t1 b1t2 b2t1 b2t2 st1 new=true asides=true"},
		{time.Now, `,"preset":"future"`, " new=true asides=false"},
		{time.Now, `,"preset":"missing"`, "b1t1 b1t2 b2t1 b2t2 new=true asides=false"},
		{time.Now, `,"preset":"existing"`, " new=false asides=false"},
		{time.Now, `,"preset":"first"`, "b1t1 b1t2 new=false asides=false"},
		{time.Now, `,"preset":"latest"`, "b2t1 b2t2 new=true asides=false"},
		{time.Now, `,"preset":"none","includeAsides":true`, "st1 new=false asides=true"},
		{time.Now, ``, "b1t1 b1t2 b2t1 b2t2 new=true asides=false"},
		{func() time.Time { return time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC) }, `,"preset":"future"`, "b1t1 b1t2 b2t1 b2t2 new=true asides=false"},
		{func() time.Time { return time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC) }, `,"preset":"latest"`, "b1t1 b1t2 new=true asides=false"},
	} {
		s := serveAt(t, c.today)
		id := s.put(t, "w1", stacked(2, 2, 1), c.body)
		if got := s.watched(t, id).on(); got != c.want {
			t.Errorf("an add with%s on %s monitors %q, want %q", c.body, c.today().Format(time.DateOnly), got, c.want)
		}
	}
}

func TestPreviewCountsEveryPresetBeforeTheAdd(t *testing.T) {
	s := serveAt(t, func() time.Time { return time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC) })
	s.fx.Put("w1", stacked(3, 4, 2))
	code, body := s.call(t, http.MethodGet, "/api/v1/content-types/fixture/catalog/w1", "")
	var p struct {
		Presets json.RawMessage `json:"presets"`
	}
	if err := json.Unmarshal([]byte(body), &p); code != http.StatusOK || err != nil {
		t.Fatalf("previewCatalogRecord = %d %s", code, body)
	}
	want := `{"asides":2,"choices":[{"preset":"all","monitors":12,"monitorsNew":true},{"preset":"future","monitors":12,"monitorsNew":true},` +
		`{"preset":"missing","monitors":12,"monitorsNew":true},{"preset":"existing","monitors":0,"monitorsNew":false},{"preset":"first","monitors":4,"monitorsNew":false},` +
		`{"preset":"latest","monitors":4,"monitorsNew":true},{"preset":"none","monitors":0,"monitorsNew":false}]}`
	if string(p.Presets) != want {
		t.Fatalf("preview presets = %s, want %s", p.Presets, want)
	}
}

func TestSetMonitoringPreviewsPerWork(t *testing.T) {
	s := stock(t, true)
	one := s.addRecord(t, "w1", stacked(2, 3, 1), `,"preset":"all"`)
	two := s.addRecord(t, "w2", stacked(1, 2, 0), `,"preset":"first"`)
	s.placeAt(t, one.Items[0], one, "/a.mkv", release.Bluray1080p, 1, s.hd.ID)
	s.placeAt(t, one.Items[4], one, "/b.mkv", release.WEBDL1080p, 1, s.hd.ID)
	ids := []string{one.ID.String(), two.ID.String(), storage.NewUUID().String(), one.ID.String()}
	before := s.watched(t, one.ID.String()).on() + " | " + s.watched(t, two.ID.String()).on()

	got := s.setMonitoring(t, choose("existing", ids...)+`,"preview":true}`).lines()
	want := "done +0 -4 all=6 future=0 missing=4 existing=2 first=3 latest=3 none=0 asides=1\n" +
		"done +0 -2 all=2 future=0 missing=2 existing=0 first=2 latest=2 none=0 asides=0\n" +
		"failed work_not_found"
	if got != want {
		t.Fatalf("a preview of Existing answers\n%s\nwant\n%s", got, want)
	}
	if after := s.watched(t, one.ID.String()).on() + " | " + s.watched(t, two.ID.String()).on(); after != before {
		t.Fatalf("a preview moved monitoring: %s, want %s", after, before)
	}

	got = s.setMonitoring(t, choose("existing", ids...)+`,"preview":false}`).lines()
	if want := "done +0 -4\ndone +0 -2\nfailed work_not_found"; got != want {
		t.Fatalf("applying Existing answers\n%s\nwant\n%s", got, want)
	}
	if got := s.watched(t, one.ID.String()).on(); got != "b1t1 b2t2 new=false asides=false" {
		t.Fatalf("after Existing, w1 monitors %q, want the two Things with a file, Met or Upgradable", got)
	}
	got = s.setMonitoring(t, choose("existing", ids...)+`,"preview":false}`).lines()
	if want := "skipped already\nskipped already\nfailed work_not_found"; got != want {
		t.Fatalf("applying Existing again answers\n%s\nwant\n%s", got, want)
	}

	got = s.setMonitoring(t, `{"workIds":["`+one.ID.String()+`"],"preset":"missing","includeAsides":true,"monitorNew":false,"preview":false}`).lines()
	if want := "done +5 -2"; got != want {
		t.Fatalf("applying Missing with Include and no new Things answers %s, want %s", got, want)
	}
	if got := s.watched(t, one.ID.String()).on(); got != "b1t2 b1t3 b2t1 b2t3 st1 new=false asides=true" {
		t.Fatalf("after Missing with Include, w1 monitors %q", got)
	}
}

func TestBulkMonitorKeepsEachWorksAsidesAndUnmonitorTurnsThemOff(t *testing.T) {
	s := serve(t)
	with := s.put(t, "w1", stacked(1, 2, 1), `,"preset":"all","includeAsides":true`)
	without := s.put(t, "w2", stacked(1, 2, 1), `,"preset":"none"`)
	bulk := func(on bool) {
		t.Helper()
		code, body := s.call(t, http.MethodPut, "/api/v1/works/monitored", fmt.Sprintf(`{"workIds":[%q,%q],"on":%t}`, with, without, on))
		if code != http.StatusOK {
			t.Fatalf("setWorksMonitored on=%t = %d %s", on, code, body)
		}
	}
	bulk(true)
	if a, b := s.watched(t, with).on(), s.watched(t, without).on(); a != "b1t1 b1t2 st1 new=true asides=true" || b != "b1t1 b1t2 new=true asides=false" {
		t.Fatalf("after bulk Monitor: %q and %q, want each Work's own asides kept", a, b)
	}
	bulk(false)
	if a := s.watched(t, with).on(); a != " new=false asides=false" {
		t.Fatalf("after bulk Unmonitor: %q, want nothing monitored and asides off", a)
	}
	bulk(true)
	if a := s.watched(t, with).on(); a != "b1t1 b1t2 new=true asides=false" {
		t.Fatalf("bulk Monitor after Unmonitor: %q, want the asides left off", a)
	}
}

func TestSetWorkMonitoredTurnsLevelsAndItems(t *testing.T) {
	s := serve(t)
	id := s.put(t, "w1", stacked(2, 2, 1), `,"preset":"none"`)
	other := s.put(t, "w2", stacked(1, 1, 0), `,"preset":"none"`)
	w, o := s.watched(t, id), s.watched(t, other)
	path := "/api/v1/works/" + id + "/monitored"
	set := func(body string) {
		t.Helper()
		if code, reply := s.call(t, http.MethodPut, path, body); code != http.StatusNoContent {
			t.Fatalf("setWorkMonitored %s = %d %s", body, code, reply)
		}
	}
	set(`{"levelIds":["` + w.level("b2") + `"],"itemIds":["` + w.item("st1") + `"],"on":true}`)
	if got := s.watched(t, id).on(); got != "b2t1 b2t2 st1 new=false asides=false" {
		t.Fatalf("after monitoring b2 and st1: %q, want b2's Things and st1 with the rule unchanged", got)
	}
	set(`{"levelIds":[],"itemIds":["` + w.item("b2t1") + `"],"on":false}`)
	if got := s.watched(t, id).on(); got != "b2t2 st1 new=false asides=false" {
		t.Fatalf("after unmonitoring b2t1: %q", got)
	}
	expect(t, s, http.MethodPut, path, `{"levelIds":[],"itemIds":["`+w.item("b1t1")+`","`+o.item("b1t1")+`"],"on":true}`, 422,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"The Work has no Item with that id.","errors":[{"location":"body.itemIds[1]","message":"The Work has no Item with that id."}]}`)
	expect(t, s, http.MethodPut, path, `{"levelIds":["`+o.level("b1")+`"],"itemIds":[],"on":true}`, 422,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"The Work has no level with that id.","errors":[{"location":"body.levelIds[0]","message":"The Work has no level with that id."}]}`)
	if got := s.watched(t, id).on(); got != "b2t2 st1 new=false asides=false" {
		t.Fatalf("a refused change moved monitoring: %q", got)
	}
	expect(t, s, http.MethodPut, "/api/v1/works/"+storage.NewUUID().String()+"/monitored", `{"levelIds":[],"itemIds":["`+w.item("b1t1")+`"],"on":true}`, 404,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
}

func TestPresetOnA10000ItemWork(t *testing.T) {
	s := serve(t)
	id := s.put(t, "w1", stacked(10, 1000, 0), `,"preset":"first"`)
	got := s.setMonitoring(t, choose("latest", id)+`,"preview":true}`).lines()
	if want := "done +1000 -1000 all=10000 future=0 missing=10000 existing=0 first=1000 latest=1000 none=0 asides=0"; got != want {
		t.Fatalf("a preview of Latest answers %s, want %s", got, want)
	}
	for _, c := range []struct{ preset, want string }{
		{"all", "done +9000 -0"}, {"first", "done +0 -9000"}, {"none", "done +0 -1000"}, {"missing", "done +10000 -0"},
	} {
		if got := s.setMonitoring(t, choose(c.preset, id)+`,"preview":false}`).lines(); got != c.want {
			t.Fatalf("applying %s answers %s, want %s", c.preset, got, c.want)
		}
	}
}

// TestBulkPresetOver32766Items applies presets to Works holding more
// Items between them than SQLite binds in one statement.
func TestBulkPresetOver32766Items(t *testing.T) {
	s := serve(t)
	var ids []string
	for i := range 17 {
		ids = append(ids, s.put(t, fmt.Sprint("w", i), stacked(2, 1000, 0), `,"preset":"none"`))
	}
	got := s.setMonitoring(t, choose("missing", ids...)+`,"preview":false}`).lines()
	if want := strings.Repeat("done +2000 -0\n", len(ids)-1) + "done +2000 -0"; got != want {
		t.Fatalf("applying Missing to every Work answers\n%s", got)
	}
	all := `{"workIds":["` + strings.Join(ids, `","`) + `"]`
	code, body := s.call(t, http.MethodPut, "/api/v1/works/monitored", all+`,"on":false}`)
	if code != http.StatusOK || strings.Count(body, `"done"`) != len(ids) {
		t.Fatalf("bulk Unmonitor = %d %s, want every Work done", code, body)
	}
	got = s.setMonitoring(t, choose("first", ids...)+`,"preview":true}`).lines()
	if !strings.HasPrefix(got, "done +1000 -0 all=2000") || strings.Count(got, "done +1000 -0") != len(ids) {
		t.Fatalf("a preview of First for every Work answers\n%s", got)
	}
}
