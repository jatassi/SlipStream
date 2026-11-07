package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// TestASavedIndexerStoppedAtACaptchaSignsInAgainWithTheCodeTypedOnItsTest
// signs Larkspur out of its site. The next search's sign-in meets the
// captcha and stops Larkspur, which raises its Fail. Its Test hands back a
// captcha, New captcha fetches another from seedbox, and a Test answering
// that one's code signs in and clears the stop.
func TestASavedIndexerStoppedAtACaptchaSignsInAgainWithTheCodeTypedOnItsTest(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	larkspur := w.indexers(t)["Larkspur"].ID
	args, err := json.Marshal(site.SiteArgs{Site: site.Larkspur})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.run.Cue(worldtest.Context(t), site.Expire.Name, args); err != nil {
		t.Fatal(err)
	}

	expectLine(t, "once Larkspur is signed out", w.searchAnother(t, "9000002"), "Larkspur", "stopped: Stopped because signing in again needs a captcha.")
	stopped := "stopped." + larkspur
	worldtest.Until(t, "the stop raises its item", func() (bool, error) {
		_, open := healthItems(t, w.a, "indexers")[stopped]
		return open, nil
	})
	if it := healthItems(t, w.a, "indexers")[stopped]; it.Level != "fail" || it.Text != "Larkspur needs a captcha to sign in again. Open the Indexer and type the code." ||
		it.Action == nil || it.Action.Href != "/settings/indexers/"+larkspur {
		t.Errorf("the stop's item is %+v, want a Fail to type the code, linking to Larkspur", it)
	}

	path := "/api/v1/indexers/" + larkspur
	refusedWithCaptcha(t, "testIndexer without a code", w.a.Admin("home", http.MethodPost, path+"/test", ""), "needs a captcha")
	reply := w.a.Admin("home", http.MethodPost, path+"/captcha", "")
	if reply.Status != http.StatusOK {
		t.Fatalf("getIndexerCaptcha = %d %s, want 200", reply.Status, reply.Body)
	}
	fresh := decode(t, "getIndexerCaptcha", reply, &captchaView{})
	answer, err := json.Marshal(map[string]any{"captcha": map[string]string{"attempt": fresh.Attempt, "answer": fresh.code(t)}})
	if err != nil {
		t.Fatal(err)
	}
	if r := w.a.Admin("home", http.MethodPost, path+"/test", string(answer)); r.Status != http.StatusOK {
		t.Fatalf("testIndexer with the code = %d %s, want 200", r.Status, r.Body)
	}
	if s := w.standing(t, larkspur).Status; s.State != "ok" {
		t.Errorf("after the Test Larkspur's status is %+v, want ok", s)
	}
	worldtest.Until(t, "the passing Test clears the stop", func() (bool, error) {
		_, open := healthItems(t, w.a, "indexers")[stopped]
		return !open, nil
	})
}
