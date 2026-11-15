package grabsapi_test

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestGrabNowAnswersHowManyReleasesItGrabs(t *testing.T) {
	f := searchable(t, site{})
	for _, c := range []struct {
		name, work, body string
		status           int
		want             string
	}{
		{"no such Work", storage.NewUUID().String(), "{}", http.StatusNotFound, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`},
		{"nothing waits", f.tree.ID.String(), "{}", http.StatusAccepted, `{"grabbing":0}`},
		{"a cell that isn't waiting", f.tree.ID.String(),
			`{"cells":[{"itemId":"` + f.tree.Items[0].ID.String() + `","edition":"any","targetId":"` + f.hd.ID.String() + `"}]}`,
			http.StatusAccepted, `{"grabbing":0}`},
	} {
		if code, body := f.s.Call(t, http.MethodPost, "/api/v1/works/"+c.work+"/waits/grab", c.body); code != c.status || body != c.want+"\n" {
			t.Errorf("%s: %d %s, want %d %s", c.name, code, body, c.status, c.want)
		}
	}
}
