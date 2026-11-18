package subtitlesapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/calls"
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

const (
	onSpeech      = "1\n00:00:01,500 --> 00:00:02,500\nHello there.\n"
	againstSpeech = "1\n00:00:00,000 --> 00:00:00,400\nOne.\n\n2\n00:00:03,500 --> 00:00:03,900\nTwo.\n"
)

func field(t *testing.T, reply, name string) string {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(reply), &v); err != nil {
		t.Fatal(err)
	}
	s, ok := v[name].(string)
	if !ok {
		t.Fatalf("%s has no %s", reply, name)
	}
	return s
}

func TestAPickAnUploadAndAKeptUploadRunOverTheAPI(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		en := subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
		p := w.Profile("Films", en)
		m := media.Record{Video: media.Video{Width: 1920, Height: 1080}, Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr"}}}
		tree := w.Work("film", p)
		f := w.Place(tree, "Film/Film.mkv", m)
		w.Sweep()
		w.Holder = subtitlestest.KeyHolder{Account: "film-fan", Plan: "Free"}
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusOK)
		w.Catalog.List(subtitlestest.Listing{
			Offer: subtitling.Offer{Provider: "subdl", Key: "late", Language: en.Language, Releases: []string{"Film.2024.1080p.WEB-DL-GRP"}},
			Name:  "Film.srt", Bytes: []byte(againstSpeech),
		})
		want := "/api/v1/files/" + f.String() + "/subtitles/en"
		search := field(t, answers(t, o, http.MethodPost, want+"/candidates", "", http.StatusAccepted), "searchId")
		w.Settle()
		answers(t, o, http.MethodGet, want+"/candidates/"+search, "", http.StatusOK, `"state":"done"`, `"key":"subdl:late"`, `"made_by_person"`)
		answers(t, o, http.MethodGet, want+"/candidates/"+storage.NewUUID().String(), "", http.StatusGone,
			`subtitle_search_expired`, `These results have expired. Search again.`)
		picked := field(t, answers(t, o, http.MethodPost, want+"/candidates/"+search+"/subdl:late", "", http.StatusAccepted), "attemptId")
		w.Settle()
		attempts := "/api/v1/files/" + f.String() + "/subtitles/attempts/"
		answers(t, o, http.MethodGet, attempts+picked, "", http.StatusOK, `"step":"finished"`, `"kind":"failed_timing"`, `"threshold":30`, `"uploadId":null`)
		if !slices.Contains(w.Attempts.All(), subtitles.Attempted{AttemptID: picked, FileID: f.String()}) {
			t.Fatalf("the pick published %v, want subtitles.attempt for %s", w.Attempts.All(), picked)
		}
		body := `{"name":"Film.en.srt","bytes":"` + base64.StdEncoding.EncodeToString([]byte(againstSpeech)) + `"}`
		uploaded := field(t, answers(t, o, http.MethodPost, want+"/upload", body, http.StatusAccepted), "attemptId")
		w.Settle()
		refused := field(t, answers(t, o, http.MethodGet, attempts+uploaded, "", http.StatusOK, `"kind":"failed_timing"`), "uploadId")
		answers(t, o, http.MethodPost, "/api/v1/files/"+f.String()+"/subtitles/uploads/"+storage.NewUUID().String()+"/keep", "", http.StatusGone,
			`subtitle_upload_expired`, `This upload has expired. Upload the file again.`)
		answers(t, o, http.MethodPost, "/api/v1/files/"+f.String()+"/subtitles/uploads/"+refused+"/keep", "", http.StatusNoContent)
		w.Settle()
		if got := w.On("Film/Film.en.srt"); got != againstSpeech {
			t.Fatalf("the kept upload wrote %q", got)
		}
		answers(t, o, http.MethodGet, attempts+storage.NewUUID().String(), "", http.StatusGone, `subtitle_attempt_expired`)
		answers(t, o, http.MethodPost, want+"/mark-bad", "", http.StatusAccepted)
		w.Settle()
		if got := w.On("Film/Film.en.srt"); got != "" {
			t.Fatalf("Mark bad left %q", got)
		}
		answers(t, o, http.MethodPost, want+"/mark-bad", "", http.StatusNotFound, `subtitle_not_found`)
		answers(t, o, http.MethodPost, want+"/search", "", http.StatusAccepted)
		answers(t, o, http.MethodPost, "/api/v1/files/"+f.String()+"/subtitles/fr/search", "", http.StatusNotFound, `subtitle_not_searched`)
	})
}

func TestAPickFromAProviderOutOfDownloadsSaysUntilWhen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		en := subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
		f := w.Place(w.Work("film", w.Profile("Films", en)), "Film/Film.mkv", media.Record{Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr"}}})
		w.Sweep()
		w.Holder = subtitlestest.KeyHolder{Account: "film-fan", Plan: "Free"}
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusOK)
		w.Catalog.List(subtitlestest.Listing{Offer: subtitling.Offer{Provider: "subdl", Key: "x", Language: en.Language}, Name: "Film.srt", Bytes: []byte(againstSpeech)})
		want := "/api/v1/files/" + f.String() + "/subtitles/en"
		search := field(t, answers(t, o, http.MethodPost, want+"/candidates", "", http.StatusAccepted), "searchId")
		w.Settle()
		reset := time.Now().Add(6 * time.Hour)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Capped, Words: "has no downloads left today", Until: reset})
		picked := field(t, answers(t, o, http.MethodPost, want+"/candidates/"+search+"/subdl:x", "", http.StatusAccepted), "attemptId")
		w.Settle()
		answers(t, o, http.MethodGet, "/api/v1/files/"+f.String()+"/subtitles/attempts/"+picked, "", http.StatusOK,
			`"kind":"unfetched"`, `"answer":{"provider":"subdl","kind":"capped","until":"`+reset.UTC().Format(time.RFC3339Nano)+`"}`)
	})
}

func TestAPickFromAProviderAlreadyPausedSaysUntilWhen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		en := subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
		f := w.Place(w.Work("film", w.Profile("Films", en)), "Film/Film.mkv", media.Record{Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr"}}})
		w.Sweep()
		w.Holder = subtitlestest.KeyHolder{Account: "film-fan", Plan: "Free"}
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusOK)
		w.Catalog.List(subtitlestest.Listing{Offer: subtitling.Offer{Provider: "subdl", Key: "x", Language: en.Language}, Name: "Film.srt", Bytes: []byte(againstSpeech)})
		want := "/api/v1/files/" + f.String() + "/subtitles/en"
		search := field(t, answers(t, o, http.MethodPost, want+"/candidates", "", http.StatusAccepted), "searchId")
		w.Settle()
		reset := time.Now().Add(6 * time.Hour)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Capped, Words: "has no downloads left today", Until: reset})
		answers(t, o, http.MethodPost, want+"/search", "", http.StatusAccepted)
		w.Settle()
		answers(t, o, http.MethodGet, "/api/v1/subtitles/providers", "", http.StatusOK, `"state":{"kind":"out"`)
		picked := field(t, answers(t, o, http.MethodPost, want+"/candidates/"+search+"/subdl:x", "", http.StatusAccepted), "attemptId")
		w.Settle()
		answers(t, o, http.MethodGet, "/api/v1/files/"+f.String()+"/subtitles/attempts/"+picked, "", http.StatusOK,
			`"kind":"unfetched"`, `"answer":{"provider":"subdl","kind":"capped","until":"`+reset.UTC().Format(time.RFC3339Nano)+`"}`)
	})
}

func TestAnUploadOverTheCapIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, o := serve(t)
		want := "/api/v1/files/" + storage.NewUUID().String() + "/subtitles/en"
		answers(t, o, http.MethodPost, "/api/v1/files/"+storage.NewUUID().String()+"/subtitles/eng/search", "", http.StatusBadRequest, `bad_want`)
		upload := func(size int) string {
			return `{"name":"Film.en.srt","bytes":"` + base64.StdEncoding.EncodeToString(make([]byte, size)) + `"}`
		}
		answers(t, o, http.MethodPost, want+"/upload", upload(5<<20+1), http.StatusUnprocessableEntity,
			`subtitle_too_large`, `Use a subtitle file of 5 MB or smaller.`)
		answers(t, o, http.MethodPost, want+"/upload", upload(6<<20), http.StatusRequestEntityTooLarge)
	})
}

func TestAKeyASearchFindsRejectedReadsWhenItWasRejected(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, o := serve(t)
		en := subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
		f := w.Place(w.Work("film", w.Profile("Films", en)), "Film/Film.mkv", media.Record{Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr"}}})
		w.Sweep()
		w.Holder = subtitlestest.KeyHolder{Account: "film-fan", Plan: "Free"}
		answers(t, o, http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"k1"}`, http.StatusOK)
		w.Settle()
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Refused, Words: "refused the key: Not Authorized"})
		rejected := time.Now().Add(w.Timings.WakeDelay)
		answers(t, o, http.MethodPost, "/api/v1/files/"+f.String()+"/subtitles/en/search", "", http.StatusAccepted)
		w.Settle()
		answers(t, o, http.MethodGet, "/api/v1/subtitles/providers", "", http.StatusOK,
			`"state":{"kind":"rejected","since":"`+rejected.UTC().Format(time.RFC3339Nano)+`","until":null,"words":"Not Authorized"}`)
	})
}
