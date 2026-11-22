package setting_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/setting"
)

var videoExtensions = setting.List{
	Key: "library.importing.videoExtensions", Label: "Video file extensions",
	Default: []string{"mkv", "mp4", "m4v", "avi", "ts", "m2ts", "mov", "wmv", "webm", "mpg", "mpeg"},
	Min:     1, Item: setting.Extension,
}

var smallestVideo = setting.Size{
	Key: "library.importing.smallestVideo", Label: "Smallest video file",
	Default: 50 * setting.MB, Min: 0, Max: 10 * setting.GB,
}

func TestListParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entries []string
		want    []string
		refusal string
	}{
		{"written lowercase with no leading dot, in order", []string{" .MKV", "Mp4", "m2ts"}, []string{"mkv", "mp4", "m2ts"}, ""},
		{"empty", []string{}, nil, "Keep at least 1 extension in the list."},
		{"a blank entry", []string{"mkv", " . "}, nil, "An extension can't be blank."},
		{"the same extension twice once written", []string{"mkv", ".MKV"}, nil, "mkv is in the list twice. Remove one."},
		{"a pattern", []string{"*.mkv"}, nil, `"*.mkv" isn't an extension. Use letters and digits only, such as mkv.`},
		{"a path", []string{`movies\a.mkv`}, nil, `"movies\a.mkv" isn't an extension. Use letters and digits only, such as mkv.`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := videoExtensions.Parse(&tc.entries)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || !slices.Equal(*got.Override(), tc.want) {
				t.Fatalf("Parse = %v, %v; want %v", got.Override(), err, tc.want)
			}
		})
	}
	if got, err := videoExtensions.Parse(nil); err != nil || got.Override() != nil {
		t.Fatalf("Parse(nil) = %v, %v; want a reset", got.Override(), err)
	}
}

func TestSizeParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		n       int64
		refusal string
	}{
		{"zero", 0, ""},
		{"whole MB", 700 * setting.MB, ""},
		{"the maximum", 10 * setting.GB, ""},
		{"above the maximum", 10*setting.GB + setting.MB, "Between 0 MB and 10 GB."},
		{"below zero", -setting.MB, "Between 0 MB and 10 GB."},
		{"off a whole MB", 50*setting.MB + 1, "In whole MB."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := smallestVideo.Parse(&tc.n)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || *got.Override() != tc.n {
				t.Fatalf("Parse(%d) = %v, %v", tc.n, got.Override(), err)
			}
		})
	}
}

func TestListAndSizeRoundTripTheirWireValue(t *testing.T) {
	for _, tc := range []struct {
		name    string
		parse   func(json.RawMessage) (string, error)
		raw     string
		want    string
		refusal string
	}{
		{"a list", stored(videoExtensions.ParseJSON), `[".MKV","webm"]`, "[mkv webm]", ""},
		{"a list as one string", stored(videoExtensions.ParseJSON), `"mkv"`, "", "A list of extensions, as strings."},
		{"an empty list", stored(videoExtensions.ParseJSON), `[]`, "", "Keep at least 1 extension in the list."},
		{"null resets a list", stored(videoExtensions.ParseJSON), `null`, "reset", ""},
		{"a size in bytes", stored(smallestVideo.ParseJSON), `104857600`, "104857600", ""},
		{"a size as words", stored(smallestVideo.ParseJSON), `"100 MB"`, "", "Bytes, as a whole number."},
		{"a size out of range", stored(smallestVideo.ParseJSON), `11811160064`, "", "Between 0 MB and 10 GB."},
		{"null resets a size", stored(smallestVideo.ParseJSON), `null`, "reset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(json.RawMessage(tc.raw))
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ParseJSON(%s) = %q, %v, want %q", tc.raw, got, err, tc.want)
			}
		})
	}

	written, err := videoExtensions.ParseJSON(json.RawMessage(`["MKV",".mp4"]`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(videoExtensions.Field(written.Override()))
	if err != nil {
		t.Fatal(err)
	}
	var field setting.ListField
	if err := json.Unmarshal(b, &field); err != nil {
		t.Fatal(err)
	}
	again, err := videoExtensions.ParseJSON(mustJSON(t, field.Value))
	if err != nil || !slices.Equal(*again.Override(), []string{"mkv", "mp4"}) {
		t.Fatalf("the field's value parsed again = %v, %v", again.Override(), err)
	}
}

func TestCheckRefusesAListOrSizeThatCantWork(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"an empty default", setting.List{Key: "l", Min: 1, Item: setting.Extension}.Check(), "setting l: default [] is not a list: l: Keep at least 1 extension in the list."},
		{"a default not in its written form", setting.List{Key: "l", Default: []string{".MKV"}, Item: setting.Extension}.Check(), "setting l: default [.MKV] is not written as [mkv]"},
		{"no item", setting.List{Key: "l", Default: []string{"mkv"}}.Check(), `setting l: item "" is not a list item`},
		{"a size off a whole MB", setting.Size{Key: "s", Default: 1000, Max: setting.GB}.Check(), "setting s: default and range are not whole steps of 1048576"},
		{"a size default above its maximum", setting.Size{Key: "s", Default: 2 * setting.GB, Max: setting.GB}.Check(), "setting s: default 2147483648 is outside 0 to 1073741824"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || tc.err.Error() != tc.want {
				t.Fatalf("Check = %v, want %s", tc.err, tc.want)
			}
		})
	}
	if err := videoExtensions.Check(); err != nil {
		t.Fatalf("Check of a list that can work = %v", err)
	}
	if err := smallestVideo.Check(); err != nil {
		t.Fatalf("Check of a size that can work = %v", err)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
