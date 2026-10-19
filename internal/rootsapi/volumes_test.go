package rootsapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestVolumesListEachVolumeThatHoldsARoot(t *testing.T) {
	s := serve(t)
	s.Expect(t, http.MethodGet, "/api/v1/volumes", "", http.StatusOK, `{"nodes":[]}`)
	s.create(t, body("fixture", t.TempDir()))
	s.create(t, body("fixture", t.TempDir()))
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO roots_seen (root_id, place, volume_id) SELECT id, place, 'media' FROM roots_root"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO roots_volume (node_id, id, name, mount, kind, total, free, sampled_at) VALUES
			(?1, 'media', '/mnt/media', '/mnt/media', 'local', 8000, 1200, 1760000000000), (?1, 'spare', '/mnt/spare', '/mnt/spare', 'removable', 100, 90, 1760000000000)`, self)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Expect(t, http.MethodGet, "/api/v1/volumes", "", http.StatusOK, `{"nodes":[{"nodeId":"id1","live":true,"volumes":[`+
		`{"id":"media","name":"/mnt/media","mount":"/mnt/media","kind":"local","total":8000,"free":1200,"sampledAt":"T","rootIds":["id2","id3"]}]}]}`)
}

func TestCheckStorageEveryIsASetting(t *testing.T) {
	s := serve(t)
	field := func(every int64) string {
		return fmt.Sprintf(`{"sampleEvery":{"key":"storage.sampleEvery","label":"Check storage every","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,"kind":"duration","value":%d,"running":%d,"default":300,"min":60,"max":86400,"step":60}}`,
			every == 300, every, every)
	}
	const path = "/api/v1/roots/settings"
	s.Expect(t, http.MethodGet, path, "", http.StatusOK, field(300))
	s.Expect(t, http.MethodPatch, path, `{"sampleEvery":30}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 min and 1 day."}`)
	s.Expect(t, http.MethodPatch, path, `{"sampleEvery":60}`, http.StatusOK, field(60))
	s.Expect(t, http.MethodGet, path, "", http.StatusOK, field(60))
	s.Expect(t, http.MethodPatch, path, `{"sampleEvery":null}`, http.StatusOK, field(300))
}
