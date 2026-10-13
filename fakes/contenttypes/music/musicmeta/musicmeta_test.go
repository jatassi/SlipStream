package musicmeta_test

import (
	"io"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
)

type exchange struct {
	path, want string
}

func get(t *testing.T, f faketest.Served, path string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.Status + " Retry-After=" + res.Header.Get("Retry-After") + " " + string(body)
}

func check(t *testing.T, f faketest.Served, exchanges []exchange) {
	t.Helper()
	for _, e := range exchanges {
		if got := get(t, f, e.path); got != e.want {
			t.Errorf("GET %s = %q, want %q", e.path, got, e.want)
		}
	}
}

func inBothTiers(t *testing.T, s scenario.Scenario, exchanges []exchange) {
	t.Run("loopback", func(t *testing.T) { check(t, faketest.Loopback(t, musicmeta.Fake, s), exchanges) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { check(t, faketest.InMemory(t, musicmeta.Fake, s), exchanges) })
	})
}

func TestTheFakeAnswersAsMusicBrainzDoes(t *testing.T) {
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	inBothTiers(t, s, []exchange{
		{"/ws/2/artist/a1000000-0000-4000-8000-000000000001", "400 Bad Request Retry-After= {\"error\":\"Bad Request\"}\n"},
		{"/ws/2/elsewhere?fmt=json", "404 Not Found Retry-After= {\"error\":\"Not Found\"}\n"},
		{"/ws/2/artist/a1000000-0000-4000-8000-000000000002?fmt=json", "503 Service Unavailable Retry-After=1 {\"error\":\"Service Unavailable\"}\n"},
		{"/ws/2/artist/a1000000-0000-4000-8000-000000000002?fmt=json", `200 OK Retry-After= {"id":"a1000000-0000-4000-8000-000000000002","name":"Marrow Bay","sort-name":"Marrow Bay","country":"NZ","disambiguation":""}` + "\n"},
		{"/ws/2/artist?query=bay&fmt=json", `200 OK Retry-After= {"count":1,"offset":0,"artists":[{"id":"a1000000-0000-4000-8000-000000000002","name":"Marrow Bay","sort-name":"Marrow Bay","country":"NZ","disambiguation":"","score":80}]}` + "\n"},
		{"/ws/2/release-group?artist=a1000000-0000-4000-8000-000000000002&type=album|ep&fmt=json", `200 OK Retry-After= {"release-group-count":1,"release-group-offset":0,"release-groups":[{"id":"b2000000-0000-4000-8000-000000000001","title":"Saltgrass","primary-type":"Album","secondary-types":[],"first-release-date":"2019-11"}]}` + "\n"},
		{"/ws/2/release?release-group=b2000000-0000-4000-8000-000000000001&status=official&inc=recordings+media&fmt=json", `200 OK Retry-After= {"release-count":1,"release-offset":0,"releases":[{"id":"c2000000-0000-4000-8000-000000000001","title":"Saltgrass","status":"Official","date":"2019-11","disambiguation":"","media":[{"position":1,"track-count":1,"track-offset":0,"tracks":[{"position":1,"number":"1","title":"Saltgrass","length":null,"recording":{"id":"d2000000-0000-4000-8000-000000000001","title":"Saltgrass","length":null}}]}]}]}` + "\n"},
	})
}

func TestAScenarioWithoutMusicServesAnEmptyCatalog(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	inBothTiers(t, s, []exchange{
		{"/ws/2/artist?query=a&fmt=json", "200 OK Retry-After= {\"count\":0,\"offset\":0,\"artists\":[]}\n"},
		{"/ws/2/artist/a1000000-0000-4000-8000-000000000001?fmt=json", "404 Not Found Retry-After= {\"error\":\"Not Found\"}\n"},
	})
}
