package myanonamouse

import (
	"encoding/json"
	"testing"
)

func TestATorrentTitleReadsFromAStringOrANumberAsStringConverterDoes(t *testing.T) {
	for body, want := range map[string]string{
		`{"title":"The Quiet Orchard"}`: "The Quiet Orchard",
		`{"title":1984}`:                "1984",
		`{"title":-2.50}`:               "-2.50",
		`{"title":null}`:                "",
	} {
		var got myAnonamouseTorrent
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Errorf("read %s: %v", body, err)
			continue
		}
		if string(got.Title) != want {
			t.Errorf("read %s: title = %q, want %q", body, got.Title, want)
		}
	}
	for _, body := range []string{`{"title":true}`, `{"title":["x"]}`, `{"title":{}}`} {
		var got myAnonamouseTorrent
		if err := json.Unmarshal([]byte(body), &got); err == nil {
			t.Errorf("read %s: title = %q, want an error as StringConverter throws", body, got.Title)
		}
	}
}
