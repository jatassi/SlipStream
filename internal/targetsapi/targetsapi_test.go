package targetsapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/works"
)

type published struct{ n int }

func (p *published) Publish(context.Context, targets.Changed) { p.n++ }

type goals struct{ t *targets.Targets }

func (g goals) Targets() (*targets.Targets, bool) { return g.t, true }

func serve(t *testing.T) (*ownertest.Owner, *published) {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), targets.Schema())
	changed := &published{}
	svc := targets.New(targets.Config{Library: lib, Types: []works.TypeID{contenttypetest.ID}, Changed: changed})
	return ownertest.Serve(t, lib, func(r *api.Router) { targetsapi.Routes(r, goals{svc}) }), changed
}

func TestTargetsAndProfilesAreCreatedAndListed(t *testing.T) {
	s, changed := serve(t)
	s.Expect(t, http.MethodPost, "/api/v1/targets", `{"name":"1080p","resolutions":["1080p","720p"]}`, http.StatusCreated,
		`{"id":"id1","name":"1080p","resolutions":["1080p","720p"],"createdAt":"T"}`)
	_, uhd := s.Call(t, http.MethodPost, "/api/v1/targets", `{"name":"2160p","resolutions":["2160p"]}`)
	_, hd := s.Call(t, http.MethodGet, "/api/v1/targets", "")
	s.Expect(t, http.MethodGet, "/api/v1/targets", "", http.StatusOK,
		`{"targets":[{"id":"id1","name":"1080p","resolutions":["1080p","720p"],"createdAt":"T"},{"id":"id2","name":"2160p","resolutions":["2160p"],"createdAt":"T"}]}`)
	hdID, uhdID := ownertest.FirstID(hd), ownertest.FirstID(uhd)

	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["`+uhdID+`","`+hdID+`"]}`, http.StatusCreated,
		`{"id":"id1","name":"Standard","contentType":"fixture","default":true,"targetIds":["id2","id3"]}`)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Small","contentType":"fixture","targetIds":["`+hdID+`"]}`, http.StatusCreated,
		`{"id":"id1","name":"Small","contentType":"fixture","default":false,"targetIds":["id2"]}`)
	s.Expect(t, http.MethodGet, "/api/v1/profiles", "", http.StatusOK,
		`{"profiles":[{"id":"id1","name":"Small","contentType":"fixture","default":false,"targetIds":["id2"]},{"id":"id3","name":"Standard","contentType":"fixture","default":true,"targetIds":["id4","id2"]}]}`)
	if changed.n != 4 {
		t.Fatalf("targets.changed published %d times, want 4", changed.n)
	}
}

func TestEachTargetsErrorAnswersItsProblem(t *testing.T) {
	s, changed := serve(t)
	_, body := s.Call(t, http.MethodPost, "/api/v1/targets", `{"name":"1080p","resolutions":["1080p"]}`)
	id := ownertest.FirstID(body)
	s.Expect(t, http.MethodPost, "/api/v1/targets", `{"name":"1080p","resolutions":["720p"]}`, http.StatusConflict,
		`{"status":409,"code":"target_name_taken","title":"A Target already has that name"}`)
	s.Expect(t, http.MethodPost, "/api/v1/targets", `{"name":"Odd","resolutions":["1440p"]}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.resolutions[0]","message":"expected value to be one of \"2160p, 1080p, 720p, 576p, 480p\""}]}`)
	s.Expect(t, http.MethodPost, "/api/v1/targets", `{"name":"Twice","resolutions":["720p","720p"]}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.resolutions","message":"expected array items to be unique"}]}`)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"ghost","targetIds":["`+id+`"]}`, http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["00000000-0000-0000-0000-000000000000"]}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"target_not_found","title":"No Target has that id"}`)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["`+id+`"]}`, http.StatusCreated,
		`{"id":"id1","name":"Standard","contentType":"fixture","default":true,"targetIds":["id2"]}`)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["`+id+`"]}`, http.StatusConflict,
		`{"status":409,"code":"profile_name_taken","title":"A Profile already has that name"}`)
	if changed.n != 2 {
		t.Fatalf("targets.changed published %d times, want 2", changed.n)
	}
}
