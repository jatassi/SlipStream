package e2e

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type captchaView struct {
	Attempt string `json:"attempt"`
	Image   string `json:"image"`
}

// code is the code c's image shows, as the Admin reads it.
func (c captchaView) code(t *testing.T) string {
	t.Helper()
	data, ok := strings.CutPrefix(c.Image, "data:image/png;base64,")
	if !ok {
		t.Fatalf("the captcha's image is %.40s…, want a PNG data URL", c.Image)
	}
	png, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}
	text, err := site.CaptchaText(png)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// refusedWithCaptcha checks r is a failed Test whose detail holds want and
// returns the new captcha it carries.
func refusedWithCaptcha(t *testing.T, what string, r worldtest.Reply, want string) captchaView {
	t.Helper()
	var p struct {
		Code    string       `json:"code"`
		Detail  string       `json:"detail"`
		Captcha *captchaView `json:"captcha"`
	}
	if err := json.Unmarshal([]byte(r.Body), &p); err != nil || r.Status != http.StatusUnprocessableEntity || p.Code != "indexer_test_failed" ||
		!strings.Contains(p.Detail, want) || p.Captcha == nil {
		t.Fatalf("%s = %d %s, want a failed Test saying %q with a new captcha", what, r.Status, r.Body, want)
	}
	return *p.Captcha
}

// TestAnAdminAddsLarkspurByItsCaptchaAndGrabsFromIt adds Larkspur beside
// the Quarry that setup added, typing the code of its captcha, and
// searches. Larkspur's priority of 10 outranks Quarry's 25, so the Grab
// comes from Larkspur, which wants thanks before a download.
func TestAnAdminAddsLarkspurByItsCaptchaAndGrabsFromIt(t *testing.T) {
	t.Parallel()
	w := begin(t, "catalog-sites")
	body := func(answer *captchaView) string {
		draft := map[string]any{
			"kind": "catalog", "name": "Larkspur", "definition": site.Larkspur, "on": true, "automaticSearch": true, "priority": 10,
			"values": map[string]string{"username": site.Username, "password": site.Password(w.s, site.Larkspur)},
		}
		if answer != nil {
			draft["captcha"] = map[string]string{"attempt": answer.Attempt, "answer": answer.code(t)}
		}
		b, err := json.Marshal(draft)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	first := refusedWithCaptcha(t, "createIndexer without a code", w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body(nil)), "needs a captcha")
	wrong := strings.Replace(body(&first), `"answer":"`, `"answer":"x`, 1)
	second := refusedWithCaptcha(t, "createIndexer with a wrong code", w.a.Admin("home", http.MethodPost, "/api/v1/indexers", wrong), site.WrongCaptcha)
	worldtest.Expect(t, "createIndexer answering a used captcha", w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body(&first)), worldtest.Reply{
		Status: http.StatusGone, Body: `{"status":410,"code":"captcha_expired","title":"The captcha expired. Load a new one and type its code."}` + "\n",
	})
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body(&second))
	if reply.Status != http.StatusCreated {
		t.Fatalf("createIndexer with the code = %d %s, want 201", reply.Status, reply.Body)
	}
	var larkspur struct {
		ID         string `json:"id"`
		Kind       string `json:"kind"`
		Definition *struct {
			ID          string `json:"id"`
			BaseAddress string `json:"baseAddress"`
		} `json:"definition"`
		Session *struct {
			Pasted bool `json:"pasted"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &larkspur); err != nil || larkspur.Kind != "catalog" || larkspur.Definition == nil ||
		larkspur.Definition.ID != site.Larkspur || larkspur.Definition.BaseAddress != "https://"+site.Host(site.Larkspur)+"/" ||
		larkspur.Session == nil || larkspur.Session.Pasted {
		t.Fatalf("createIndexer = %s, want Larkspur's catalog Indexer at its first link, holding the session it signed in for", reply.Body)
	}

	stream := worldtest.NewMessages(t, w.m, world.AdminIdentity)
	search := w.search(t)
	event(t, stream, "grabs.search-changed", `{"searchId":"`+search+`","workId":"`+w.work+`"}`)
	w.ended(t, grabChanged(t, stream, w.work))
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if from := w.grabbedFrom(t, w.grab.ID); from != larkspur.ID {
		t.Fatalf("the Grab came from Indexer %s, want Larkspur's %s", from, larkspur.ID)
	}
	var list struct {
		Searches []struct {
			Indexers []struct {
				Name     string `json:"name"`
				Outcome  string `json:"outcome"`
				Releases int    `json:"releases"`
			} `json:"indexers"`
		} `json:"searches"`
	}
	decode(t, "listSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &list)
	if len(list.Searches) != 1 || len(list.Searches[0].Indexers) != 2 {
		t.Fatalf("listSearches = %+v, want one search of Larkspur and Quarry", list)
	}
	for _, ix := range list.Searches[0].Indexers {
		if ix.Outcome != "found" || ix.Releases != 2 {
			t.Errorf("%s's search ended %s with %d Releases, want both found", ix.Name, ix.Outcome, ix.Releases)
		}
	}
}
