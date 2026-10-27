package worksapi_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// listLibrary compresses its rows for a client that accepts gzip, and
// answers plain JSON to one that doesn't.
func TestTheLibraryIsGzippedForAClientThatAcceptsIt(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	if code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1","monitor":false}`); code != http.StatusOK {
		t.Fatalf("addWork = %d %s", code, body)
	}
	_, plain := s.call(t, http.MethodGet, "/api/v1/library", "")
	get := func(accept string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/library", nil)
		req.Header.Set("X-Api-Key", s.apiKey)
		req.Header.Set("Accept-Encoding", accept)
		w := httptest.NewRecorder()
		s.handler.ServeHTTP(w, req)
		return w
	}
	zipped := get("br, gzip;q=0.8")
	r, err := gzip.NewReader(zipped.Body)
	if err != nil {
		t.Fatalf("listLibrary with gzip accepted answered %q, Content-Encoding %q: %v", zipped.Body.String(), zipped.Header().Get("Content-Encoding"), err)
	}
	body, err := io.ReadAll(r)
	if err != nil || string(body) != plain || zipped.Code != http.StatusOK || zipped.Header().Get("Content-Encoding") != "gzip" || zipped.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("gzipped listLibrary = %d %v, body %q, %v; want %q compressed and varying by Accept-Encoding", zipped.Code, zipped.Header(), body, err, plain)
	}
	for _, accept := range []string{"", "gzip;q=0", "br"} {
		w := get(accept)
		if w.Body.String() != plain || w.Header().Get("Content-Encoding") != "" || w.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatalf("listLibrary with Accept-Encoding %q = %v %q, want plain JSON varying by Accept-Encoding", accept, w.Header(), w.Body.String())
		}
	}
}
