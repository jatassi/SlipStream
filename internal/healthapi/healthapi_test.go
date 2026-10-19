package healthapi_test

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/healthapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type watch struct{ h *health.Health }

func (w watch) Health() (*health.Health, bool) { return w.h, true }

type discard struct{}

func (discard) Publish(context.Context, health.Changed) {}

var media = role.NodeID{UUID: storage.NewUUID()}

func answer(a health.Answer) func(context.Context) (health.Answer, error) {
	return func(context.Context) (health.Answer, error) { return a, nil }
}

func serve(t *testing.T) *ownertest.Owner {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), health.Schema())
	hub := live.NewHub(nil)
	t.Cleanup(hub.Close)
	disk := health.Volume{Node: media, NodeName: "Media server", Mount: `D:\`, Name: "D: (Media)", Total: 1 << 40, Free: 10 << 30, SampledAt: time.Now(), Live: true}
	h, err := health.Open(t.Context(), health.Config{
		Library: lib,
		Hub:     hub,
		Changed: discard{},
		Storage: health.StorageParts{Volumes: []health.VolumeSource{{
			Volumes: func(context.Context) ([]health.Volume, error) { return []health.Volume{disk}, nil },
			Sample:  func(context.Context, role.NodeID) error { return health.ErrUnreachable },
		}}},
		Categories: []health.Category{
			{About: health.About{ID: health.NodesID, Label: "Nodes", Phrase: "Nodes"}, Source: health.Source{Answer: answer(health.Answer{})}},
			{About: health.About{ID: health.ScheduledTasksID, Label: "Scheduled tasks", Phrase: "scheduled tasks"}, Source: health.Source{Answer: answer(health.Answer{
				Items: []health.Item{{Key: "housekeeping", Level: health.Warn, Says: health.Says{Subject: "Housekeeping", Text: "Housekeeping failed.", Action: health.RunAction("housekeeping")}}},
			})}},
		},
		Names: func(context.Context, storage.Reader[storage.Library]) (map[role.NodeID]string, error) {
			return map[role.NodeID]string{media: "Media server"}, nil
		},
		Zone:    time.UTC,
		Timings: health.DefaultTimings(),
		Log:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return ownertest.Serve(t, lib, func(r *api.Router) { healthapi.Routes(r, watch{h}) })
}

func TestTheReportListsEachCategoryWithItsItems(t *testing.T) {
	s := serve(t)
	key := health.KeyOf("v", media.String(), health.Opaque(`D:\`))
	s.Expect(t, http.MethodGet, "/api/v1/system/health", "", http.StatusOK,
		`{"level":"fail","first":{"category":"storage","key":"v.id1.RDpc","text":"Free up space on D: (Media) on Media server, or add a library root on another volume. It has 10 GB left."},"categories":[`+
			`{"id":"storage","label":"Storage","phrase":"storage","level":"fail","canCheck":true,"items":[{"key":"v.id1.RDpc","level":"fail","subject":"D: (Media) on Media server",`+
			`"text":"Free up space on D: (Media) on Media server, or add a library root on another volume. It has 10 GB left.","detail":[{"label":"Free","bytes":10737418240},{"label":"Total","bytes":1099511627776},`+
			`{"label":"Warning level","text":"100 GB (global)"},{"label":"Error level","text":"20 GB (global)"},{"label":"Sampled","at":"T"}],"action":{"kind":"check"},"since":"T","changedAt":"T","unsure":false}]},`+
			`{"id":"nodes","label":"Nodes","phrase":"Nodes","level":null,"canCheck":false,"items":[]},`+
			`{"id":"scheduled-tasks","label":"Scheduled tasks","phrase":"scheduled tasks","level":"warn","canCheck":false,"items":[{"key":"housekeeping","level":"warn","subject":"Housekeeping",`+
			`"text":"Housekeeping failed.","detail":[],"action":{"kind":"run","taskId":"housekeeping"},"since":"T","changedAt":"T","unsure":false}]}]}`)

	_, checked := s.Call(t, http.MethodPost, "/api/v1/system/health/storage/items/"+string(key)+"/check", "")
	if want := `"unreached":[{"key":"` + string(key) + `","words":"Media server is offline, so its figures are as of `; !strings.Contains(checked, want) {
		t.Fatalf("checkHealthItem on an offline volume = %s, want it unreached", checked)
	}
	s.Expect(t, http.MethodPost, "/api/v1/system/health/nodes/check", "", http.StatusNotFound,
		`{"status":404,"code":"no_check","title":"There is no Check now for that"}`)
	s.Expect(t, http.MethodPost, "/api/v1/system/health/scheduled-tasks/items/housekeeping/check", "", http.StatusNotFound,
		`{"status":404,"code":"no_check","title":"There is no Check now for that"}`)
	s.Expect(t, http.MethodPost, "/api/v1/system/health/storage/items/nothing/check", "", http.StatusNotFound,
		`{"status":404,"code":"health_item_not_found","title":"The category has no item with that key"}`)
	s.Expect(t, http.MethodPost, "/api/v1/system/health/indexers/check", "", http.StatusNotFound,
		`{"status":404,"code":"health_category_not_found","title":"No Health category has that id"}`)
}

func TestTheLevelsAreSetGloballyAndPerVolume(t *testing.T) {
	s := serve(t)
	s.Expect(t, http.MethodPatch, "/api/v1/system/health/settings", `{"errorBelow":{"amount":214748364800,"unit":"bytes"}}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"No more than the warning level, 100 GB."}`)
	s.Expect(t, http.MethodPatch, "/api/v1/system/health/volumes", `{"nodeId":"`+storage.NewUUID().String()+`","mount":"E:\\","warnBelow":{"amount":5,"unit":"percent"}}`,
		http.StatusNotFound, `{"status":404,"code":"health_volume_not_found","title":"No volume reports at that Node and mount"}`)
	_, row := s.Call(t, http.MethodPatch, "/api/v1/system/health/volumes", `{"nodeId":"`+media.String()+`","mount":"D:\\","warnBelow":{"amount":0,"unit":"percent"},"errorBelow":{"amount":0,"unit":"percent"}}`)
	if want := `"warnBytes":0,"errorBytes":0,"skipsWarning":false,"level":null}`; !strings.Contains(row, want) {
		t.Fatalf("updateHealthVolume to 0%% = %s, want it below neither level", row)
	}
	if status, body := s.Call(t, http.MethodDelete, "/api/v1/system/health/volumes?nodeId="+media.String()+"&mount=D:%5C", ""); status != http.StatusNoContent {
		t.Fatalf("deleteHealthVolume = %d %s, want 204", status, body)
	}
	_, list := s.Call(t, http.MethodGet, "/api/v1/system/health/volumes", "")
	if want := `"warnBytes":107374182400,"errorBytes":21474836480,"skipsWarning":false,"level":"fail"}],"notReporting":[]}`; !strings.Contains(list, want) {
		t.Fatalf("listHealthVolumes after the reset = %s, want the global levels and a fail", list)
	}
}
