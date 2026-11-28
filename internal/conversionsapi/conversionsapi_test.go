package conversionsapi_test

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/conversionsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// Each operation S44 (Conversion) declares ahead of its build answers 501
// not_built_yet exactly while its notyet_<op>.go file exists.
func TestNotYetLedger(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	s := ownertest.Serve(t, lib, func(r *api.Router) { conversionsapi.Routes(r) })
	id, node := storage.NewUUID().String(), storage.NewUUID().String()
	for _, c := range []struct{ op, method, path, body string }{
		{"listConversions", http.MethodGet, "/api/v1/conversions", ""},
		{"getConversionGroup", http.MethodGet, "/api/v1/conversions/groups/backfill-1", ""},
		{"moveConversionUp", http.MethodPost, "/api/v1/conversions/" + id + "/move-up", ""},
		{"moveConversion", http.MethodPost, "/api/v1/conversions/" + id + "/move", `{"after":null}`},
		{"retryConversion", http.MethodPost, "/api/v1/conversions/" + id + "/retry", ""},
		{"cancelConversion", http.MethodPost, "/api/v1/conversions/" + id + "/cancel", ""},
		{"cancelConversionGroup", http.MethodPost, "/api/v1/conversions/groups/backfill-1/cancel", ""},
		{"getNodeConverting", http.MethodGet, "/api/v1/nodes/" + node + "/converting", ""},
		{"updateNodeConverting", http.MethodPut, "/api/v1/nodes/" + node + "/converting", `{"atOnce":null,"hours":{"on":false,"window":{"start":"01:00","end":"07:00"}},"processor":true}`},
		{"rerunNodeTests", http.MethodPost, "/api/v1/nodes/" + node + "/converting/rerun", ""},
	} {
		_, err := os.Stat("notyet_" + c.op + ".go")
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		stub := err == nil
		status, reply := s.Call(t, c.method, c.path, c.body)
		if (status == http.StatusNotImplemented) != stub {
			t.Errorf("%s answered %d %s; its notyet file exists: %v", c.op, status, reply, stub)
		}
	}
}
