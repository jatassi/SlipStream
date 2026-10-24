package artwork_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

var mine = artwork.NewMinter([]byte("mine"))

func TestATokenOpensToItsReference(t *testing.T) {
	for _, r := range []artwork.Ref{
		{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: "/kqjL17yufvn9OVLyXYpvtyrFfak.jpg"},
		{Source: "music-coverart", Shape: artwork.ShapeCover, Path: "1b022e01-4da6-387b-8658-8678046e4cef"},
		{Source: "s", Shape: artwork.ShapePhoto, Path: "a path/with ünicode?&#"},
	} {
		u := mine.URL(r)
		token, ok := strings.CutPrefix(u, artwork.Route)
		if !ok || strings.ContainsAny(token, "/?#&=+ ") {
			t.Errorf("URL(%+v) = %q, want the artwork route and a token safe in a path", r, u)
		}
		got, err := mine.Open(token)
		if err != nil || got != r {
			t.Errorf("Open(URL(%+v)) = %+v, %v", r, got, err)
		}
	}
}

func TestATokenItDidntIssueIsForged(t *testing.T) {
	r := artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: "/a.jpg"}
	theirs := artwork.NewMinter([]byte("theirs")).Token(r)
	other := mine.Token(artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: "/b.jpg"})
	token := mine.Token(r)
	payload, _, _ := strings.Cut(token, ".")
	_, otherMAC, _ := strings.Cut(other, ".")
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	loose := []byte(token)
	loose[len(loose)-1] = alphabet[strings.IndexByte(alphabet, loose[len(loose)-1])^1]
	for name, token := range map[string]string{
		"another install's":   theirs,
		"a spliced signature": payload + "." + otherMAC,
		"no signature":        payload,
		"empty":               "",
		"not base64":          "!!.!!",
		"a bare dot":          ".",
		"its own, respelled":  string(loose),
	} {
		if got, err := mine.Open(token); !errors.Is(err, artwork.ErrForged) {
			t.Errorf("Open(%s token %q) = %+v, %v; want ErrForged", name, token, got, err)
		}
	}
}
