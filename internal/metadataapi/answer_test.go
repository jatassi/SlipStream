package metadataapi

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

func TestARunningRefreshSaysSinceWhenInTheNodesZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, secret.Schema(), identity.Schema())
		since := time.Now().Add(-30 * time.Minute)
		o := ownertest.ServeIn(t, lib, time.FixedZone("UTC+9", 9*60*60), func(r *api.Router) {
			op := api.Op{ID: "refreshFixture", Method: http.MethodPost, Path: "/fixture", Errors: []*api.Code{taskRunning}}
			api.Register(r.Owner("fixture", api.Admin), api.Admin, op, func(context.Context, identity.Admin, *struct{}) (*struct{}, error) {
				return nil, answer(&tasks.ErrRunning{Since: since})
			})
		})
		o.Expect(t, http.MethodPost, "/api/v1/fixture", "", http.StatusConflict,
			`{"status":409,"code":"task_running","title":"The task has a run open","detail":"Metadata refresh has been running since 08:30. The next run refreshes every Work."}`)
	})
}
