package subtitlesapi_test

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/subtitlesapi"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

type owner struct{ w *subtitlestest.World }

func (o owner) Subtitles() (*subtitles.Subtitles, bool) { return o.w.Subs, true }
func (o owner) Targets() (*targets.Targets, bool)       { return o.w.Targets, true }

func serve(t *testing.T) (*subtitlestest.World, *ownertest.Owner) {
	t.Helper()
	w := subtitlestest.Open(t, subtitles.DefaultTimings(), subtitling.DefaultTimings())
	return w, ownertest.Serve(t, w.Lib, func(r *api.Router) { subtitlesapi.Routes(r, owner{w}) })
}

// answers calls and checks the status and that the reply holds each of
// parts.
func answers(t *testing.T, o *ownertest.Owner, method, path, body string, status int, parts ...string) string {
	t.Helper()
	code, got := o.Call(t, method, path, body)
	if code != status {
		t.Fatalf("%s %s = %d %s, want %d", method, path, code, got, status)
	}
	for _, p := range parts {
		if !strings.Contains(got, p) {
			t.Fatalf("%s %s = %s, want it to hold %s", method, path, got, p)
		}
	}
	return got
}

func TestTheSettingsReadAndRefuseAGapUnderTheInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, o := serve(t)
		answers(t, o, http.MethodGet, "/api/v1/subtitles/settings", "", http.StatusOK,
			`"key":"subtitles.sync.match"`, `"kind":"no_languages"`, `"interval":28800`)
		answers(t, o, http.MethodPatch, "/api/v1/subtitles/settings", `{"values":{"subtitles.sync.match":55}}`, http.StatusOK)
		answers(t, o, http.MethodPatch, "/api/v1/subtitles/settings",
			`{"values":{"tasks.subtitle-search.cadence":172800,"subtitles.longestGap":86400}}`, http.StatusUnprocessableEntity,
			`gap_too_short`, `"location":"body.values.subtitles.longestGap"`, `Use 2 days or longer. Subtitle search runs every 2 days.`)
		answers(t, o, http.MethodGet, "/api/v1/subtitles/nodes/"+storage.NewUUID().String()+"/settings", "", http.StatusNotFound, `not_a_storage_node`)
	})
}

func TestAProviderKeyIsRefusedInTheProvidersOwnWords(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		answers(t, o, http.MethodGet, "/api/v1/subtitles/providers", "", http.StatusOK, `"id":"subdl"`, `"kind":"needs_key"`)
		w.Holder = subtitlestest.KeyHolder{Refused: "Not Authorized"}
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusUnprocessableEntity,
			`subtitle_key_rejected`, `SubDL rejected this key. It responded \"Not Authorized\". Check that you copied the whole key.`)
		w.Holders(false)
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusConflict, `no_subtitles_holder`)
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/order", `{"ids":["gestdown"]}`, http.StatusUnprocessableEntity,
			`provider_order_mismatch`, `List every Subtitle Provider once.`)
		answers(t, o, http.MethodPatch, "/api/v1/subtitles/providers/opensubtitles", `{"on":true}`, http.StatusNotFound, `subtitle_provider_not_found`)
	})
}

func TestAWorksSubtitlesReadAndATrackIsIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		m := media.Record{Video: media.Video{Width: 1920, Height: 1080}, Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr"}}}
		m.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "en"}}
		tree := w.Work("film", p)
		f := w.Place(tree, "Film/Film.mkv", m)
		work := "/api/v1/works/" + tree.ID.String() + "/subtitles"
		answers(t, o, http.MethodGet, work, "", http.StatusOK, `"path":"Film/Film.mkv"`, `"kind":"embedded"`)
		track := "/api/v1/files/" + f.String() + "/subtitles/tracks/"
		answers(t, o, http.MethodPut, track+"0/ignored", "", http.StatusNoContent)
		w.Settle()
		answers(t, o, http.MethodGet, work, "", http.StatusOK, `"kind":"missing"`, `"ignored":true`)
		answers(t, o, http.MethodDelete, track+"0/ignored", "", http.StatusNoContent)
		answers(t, o, http.MethodPut, track+"3/ignored", "", http.StatusNotFound, `track_not_found`)
	})
}

func TestTheManualActionsSayTheyAreNotBuiltAndCapAnUpload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, o := serve(t)
		want := "/api/v1/files/" + storage.NewUUID().String() + "/subtitles/en"
		answers(t, o, http.MethodPost, want+"/search", "", http.StatusNotImplemented,
			`subtitle_action_not_built`, `This build of SlipStream can't search for, download or upload subtitles yet.`)
		answers(t, o, http.MethodPost, "/api/v1/files/"+storage.NewUUID().String()+"/subtitles/eng/search", "", http.StatusBadRequest, `bad_want`)
		upload := func(size int) string {
			return `{"name":"Film.en.srt","bytes":"` + base64.StdEncoding.EncodeToString(make([]byte, size)) + `"}`
		}
		answers(t, o, http.MethodPost, want+"/upload", upload(5<<20+1), http.StatusUnprocessableEntity,
			`subtitle_too_large`, `Use a subtitle file of 5 MB or smaller.`)
		answers(t, o, http.MethodPost, want+"/upload", upload(6<<20), http.StatusRequestEntityTooLarge)
		answers(t, o, http.MethodPost, want+"/upload", upload(1<<10), http.StatusNotImplemented, `subtitle_action_not_built`)
	})
}
