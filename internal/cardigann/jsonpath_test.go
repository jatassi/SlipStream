package cardigann

import (
	"testing"
	"time"
)

const testJSON = `{
  "data": [
    {"id": 1, "name": "A <b>", "size": 1.5, "big": 1e15, "small": 0.00001, "free": true,
     "added": "2021-03-04T05:06:07", "addedZ": "2021-03-04T05:06:07+05:30",
     "tags": ["x", "y"], "files": {"n": 2}, "nothing": null, "imdb": "tt01"},
    {"id": 2, "name": "B", "size": 2, "free": false, "tags": [], "files": {}},
    {"name": "no id"}
  ],
  "total": 3
}`

func mustJSON(t *testing.T, body string) *jsonToken {
	t.Helper()
	j, err := parseJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestJSONText(t *testing.T) {
	row := mustJSON(t, testJSON).items[0].items[0]
	tests := []struct {
		path, want string
	}{
		{"id", "1"},
		{"name", "A <b>"},
		{"size", "1.5"},
		{"big", "1E+15"},
		{"small", "1E-05"},
		{"free", "True"},
		{"added", "03/04/2021 05:06:07"},
		{"addedZ", "03/04/2021 01:36:07"},
		{"tags[1]", "y"},
		{"files.n", "2"},
		{"['files']['n']", "2"},
	}
	for _, tt := range tests {
		tok, err := row.SelectToken(tt.path)
		if err != nil || tok == nil {
			t.Fatalf("SelectToken(%q) = %v, %v", tt.path, tok, err)
		}
		got, err := tok.Text(testZone)
		if err != nil || got.String() != tt.want {
			t.Errorf("%s = %q, %v; want %q", tt.path, got.String(), err, tt.want)
		}
	}
	if tok, _ := row.SelectToken("nothing"); tok == nil {
		t.Error("nothing: a JSON null wasn't found")
	} else if v, _ := tok.Text(testZone); !v.IsNull() {
		t.Errorf("a JSON null reads as %q", v.String())
	}
	if tok, _ := row.SelectToken("files"); tok != nil {
		if _, err := tok.Text(testZone); err == nil {
			t.Error("an object read as a value: want an error")
		}
	}
	if tok, _ := row.SelectToken("missing.x"); tok != nil {
		t.Error("a missing path found a token")
	}
}

func TestJSONString(t *testing.T) {
	row := mustJSON(t, `{"a": 1, "b": [true, null, 2.0, "q\"< "], "c": {}}`)
	want := "{\n  \"a\": 1,\n  \"b\": [\n    true,\n    null,\n    2.0,\n    \"q\\\"<\\u2028\"\n  ],\n  \"c\": {}\n}"
	if got := row.String(); got != want {
		t.Errorf("String() =\n%s\nwant\n%s", got, want)
	}
}

func TestFormatDotNetDouble(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{1.5, "1.5"}, {2, "2"}, {0.1, "0.1"}, {1e14, "100000000000000"}, {1e15, "1E+15"},
		{1.25e20, "1.25E+20"}, {0.0001, "0.0001"}, {0.00001, "1E-05"}, {-3.5, "-3.5"},
	}
	for _, tt := range tests {
		if got := formatDotNetDouble(tt.in); got != tt.want {
			t.Errorf("formatDotNetDouble(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestJSONRows(t *testing.T) {
	doc := mustJSON(t, testJSON)
	tests := []struct {
		selector string
		want     int
	}{
		{"data", 3},
		{"$.data", 3},
		{"data:has(id)", 2},
		{"data:not(id)", 1},
		{"data:has(files.n)", 1},
		{"data:contains(no id)", 1},
		{"data:has(id):not(imdb)", 1},
	}
	for _, tt := range tests {
		rows, ok := doc.Rows(tt.selector)
		if !ok || len(rows) != tt.want {
			t.Errorf("Rows(%q) = %d rows, %v; want %d", tt.selector, len(rows), ok, tt.want)
		}
	}
	for _, sel := range []string{"total", "missing", "data[0]"} {
		if _, ok := doc.Rows(sel); ok {
			t.Errorf("Rows(%q): want no array", sel)
		}
	}
}

func TestSelectJSON(t *testing.T) {
	e := testEnv(t)
	doc := mustJSON(t, testJSON)
	rows, _ := doc.Rows("data")
	tests := []struct {
		name, sel string
		row       *jsonToken
		want      string
	}{
		{"value", "name", rows[0], "A <b>"},
		{"array joined", "tags", rows[0], "x,y"},
		{"empty array", "tags", rows[1], ""},
		{"filter on the selected token", "tags:has([1])", rows[0], "x,y"},
		{"parent row", "..total", doc, "3"},
		{"no path reads the row's filters", ":has(imdb)", rows[0], ""},
	}
	for _, tt := range tests {
		b := block(t, tt.sel, "")
		got, err := b.SelectJSON(tt.row, e, true)
		if tt.name == "no path reads the row's filters" {
			if err == nil {
				t.Errorf("%s: an object read as a value: want an error", tt.name)
			}
			continue
		}
		if err != nil || got.String() != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got.String(), err, tt.want)
		}
	}
	if v, err := block(t, "tags:not([1])", "").SelectJSON(rows[0], e, false); err != nil || !v.IsNull() {
		t.Errorf("an optional filtered field = %#v, %v", v, err)
	}
	if _, err := block(t, "imdb", "").SelectJSON(rows[1], e, true); err == nil {
		t.Error("a required missing field: want an error")
	}

	yes, _ := parseTemplate("yes")
	no, _ := parseTemplate("no")
	c := block(t, "free", "")
	c.Case = keyedList[*template]{{Key: "True", Value: yes}, {Key: "*", Value: no}}
	for i, want := range []string{"yes", "no"} {
		if v, err := c.SelectJSON(rows[i], e, true); err != nil || v.String() != want {
			t.Errorf("case on row %d = %q, %v; want %q", i, v.String(), err, want)
		}
	}
	nameOnly := block(t, "name", "")
	nameOnly.Case = keyedList[*template]{{Key: "B", Value: yes}}
	if v, _ := nameOnly.SelectJSON(rows[0], e, true); v.String() != "A <b>" {
		t.Errorf("a value no case matches = %q, want it kept", v.String())
	}
}

func TestParseJSONErrors(t *testing.T) {
	for _, body := range []string{"", "{", `{"a":1} x`, "[1,]"} {
		if _, err := parseJSON(body); err == nil {
			t.Errorf("parseJSON(%q): want an error", body)
		}
	}
	for _, p := range []string{"a[", "a[x]", "a.*", "a[?(@.x)]"} {
		if _, err := compileJSONPath(p); err == nil {
			t.Errorf("compileJSONPath(%q): want an error", p)
		}
	}
}

func TestParseISODate(t *testing.T) {
	tests := []struct {
		in     string
		want   time.Time
		offset bool
		isDate bool
	}{
		{"2021-03-04T05:06:07", time.Date(2021, 3, 4, 5, 6, 7, 0, time.UTC), false, true},
		{"2021-03-04T05:06:07.5Z", time.Date(2021, 3, 4, 5, 6, 7, 5e8, time.UTC), true, true},
		{"2021-03-04T05:06:07-02:00", time.Date(2021, 3, 4, 7, 6, 7, 0, time.UTC), true, true},
		{"2021-02-30T05:06:07", time.Time{}, false, false},
		{"2021-03-04 05:06:07", time.Time{}, false, false},
		{"not a date", time.Time{}, false, false},
	}
	for _, tt := range tests {
		got, offset, isDate := parseISODate(tt.in)
		if isDate != tt.isDate || (isDate && (!got.Equal(tt.want) || offset != tt.offset)) {
			t.Errorf("parseISODate(%q) = %v, %v, %v", tt.in, got, isDate, offset)
		}
	}
}
