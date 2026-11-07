package cardigann

import "testing"

func arg(s string) filterArgs     { return filterArgs{Items: []string{s}} }
func args(s ...string) filterArgs { return filterArgs{Items: s, IsList: true} }
func testEnv(t *testing.T) env {
	return env{Scope: fixtureScope(), Clock: testClock, charset: mustCharset(t, "UTF-8")}
}
func runFilter(t *testing.T, name string, a filterArgs, in value) (value, error) {
	t.Helper()
	f, err := compileFilter(name, a)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return filterBlocks{f}.Apply(in, testEnv(t))
}

func TestFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		args   filterArgs
		in     string
		want   string
	}{
		{"querystring", "querystring", arg("id"), "details.php?ID=42&x=1", "42"},
		{"querystring first", "querystring", arg("id"), "/d?id=1&id=2#id=3", "1"},
		{"querystring decodes", "querystring", arg("q"), "/s?q=a+b%26c", "a b&c"},
		{"querystring bare key", "querystring", arg("hd"), "/s?hd&id=1", ""},
		{"dateparse", "dateparse", arg("2006-01-02 15:04"), "2021-03-04 05:06", "Thu, 04 Mar 2021 05:06:00 +2"},
		{"dateparse .NET format", "dateparse", arg("dd.MM.yyyy HH:mm"), "04.03.2021 05:06", "Thu, 04 Mar 2021 05:06:00 +2"},
		{"dateparse keeps a mismatch", "dateparse", arg("2006-01-02"), "yesterday", "yesterday"},
		{"timeparse", "timeparse", arg("2006-01-02"), "2021-03-04", "Thu, 04 Mar 2021 00:00:00 +2"},
		{"regexp", "regexp", arg(`(\d+) GB`), "size 12 GB", "12"},
		{"regexp no match", "regexp", arg(`(\d+) GB`), "none", ""},
		{"re_replace", "re_replace", args(`\s+`, "."), "a  b c", "a.b.c"},
		{"re_replace template", "re_replace", args(`^`, "{{ .Config.sitelink }}"), "x", "https://somesite.com/x"},
		{"split", "split", args("/", "1"), "a/b/c", "b"},
		{"split negative", "split", args("/", "-1"), "a/b/c", "c"},
		{"split first rune", "split", args("/-", "0"), "a/b-c", "a"},
		{"replace", "replace", args("-", " "), "a-b-c", "a b c"},
		{"replace template", "replace", args("X", "{{ .Config.x }}"), "aXb", "a1b"},
		{"trim", "trim", filterArgs{}, "  a b \t", "a b"},
		{"trim char", "trim", arg("*"), "**a*b**", "a*b"},
		{"prepend", "prepend", arg("{{ .Config.sitelink }}"), "x", "https://somesite.com/x"},
		{"append", "append", arg(" GB"), "12", "12 GB"},
		{"tolower", "tolower", filterArgs{}, "AbC", "abc"},
		{"toupper", "toupper", filterArgs{}, "AbC", "ABC"},
		{"base64decode", "base64decode", filterArgs{}, "aGk_", "hi?"},
		{"base64decode unpadded", "base64decode", filterArgs{}, "aGk", "hi"},
		{"base64encode", "base64encode", filterArgs{}, "hi?", "aGk_"},
		{"urldecode", "urldecode", filterArgs{}, "a+b%2Fc", "a b/c"},
		{"urlencode", "urlencode", filterArgs{}, "a b/c", "a+b%2Fc"},
		{"htmldecode", "htmldecode", filterArgs{}, "a &amp; b &lt;&#39;&#x41;&eacute;&bogus; &amp", "a & b <'Aé&bogus; &amp"},
		{"htmlencode", "htmlencode", filterArgs{}, `<a href="x">'&é`, "&lt;a href=&quot;x&quot;&gt;&#39;&amp;&#233;"},
		{"timeago", "timeago", filterArgs{}, "2 hours ago", "Tue, 27 Oct 2026 15:04:05 +2"},
		{"reltime", "reltime", filterArgs{}, "1 day", "Mon, 26 Oct 2026 17:04:05 +2"},
		{"fuzzytime", "fuzzytime", filterArgs{}, "Mar 4th, 2021", "Thu, 04 Mar 2021 00:00:00 +2"},
		{"validfilename", "validfilename", filterArgs{}, `a:b/c?d"e`, "a_b_c_d_e"},
		{"validfilename empty", "validfilename", filterArgs{}, "", "_"},
		{"diacritics", "diacritics", arg("replace"), "ŠĐĆŽ - šđčćž", "SĐCZ - sđccz"},
		{"jsonjoinarray", "jsonjoinarray", args("$.genres", ", "), `{"genres": ["Action", "Drama", 3]}`, "Action, Drama, 3"},
		{"validate", "validate", arg("1080p, 720p, HDR, x265"), "Movie (2021) [x265 HDR 1080p]", "1080p, hdr, x265"},
		{"hexdump", "hexdump", filterArgs{}, "a", "a"},
		{"strdump", "strdump", filterArgs{}, "a", "a"},
		{"andmatch", "andmatch", filterArgs{}, "a", "a"},
		{"unknown filter", "nosuchfilter", arg("x"), "a", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runFilter(t, tt.filter, tt.args, textValue(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if got.IsNull() || got.String() != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.filter, tt.in, got.String(), tt.want)
			}
		})
	}
}

func TestFiltersOnNull(t *testing.T) {
	tests := []struct {
		filter  string
		args    filterArgs
		want    value
		wantErr bool
	}{
		{"tolower", filterArgs{}, nullValue, true},
		{"regexp", arg("(a)"), nullValue, true},
		{"timeago", filterArgs{}, nullValue, true},
		{"hexdump", filterArgs{}, nullValue, true},
		{"htmldecode", filterArgs{}, nullValue, false},
		{"htmlencode", filterArgs{}, nullValue, false},
		{"urlencode", filterArgs{}, textValue(""), false},
		{"prepend", arg("x"), textValue("x"), false},
		{"append", arg("x"), textValue("x"), false},
	}
	for _, tt := range tests {
		got, err := runFilter(t, tt.filter, tt.args, nullValue)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s(null): error %v, want an error %v", tt.filter, err, tt.wantErr)
			continue
		}
		if err == nil && (got.IsNull() != tt.want.IsNull() || got.String() != tt.want.String()) {
			t.Errorf("%s(null) = %#v, want %#v", tt.filter, got, tt.want)
		}
	}
}

func TestFilterRuntimeErrors(t *testing.T) {
	tests := []struct {
		filter string
		args   filterArgs
		in     string
	}{
		{"querystring", arg("id"), "no-query"},
		{"querystring", arg("id"), "/d?x=1"},
		{"split", args("/", "5"), "a/b"},
		{"base64decode", filterArgs{}, "a"},
		{"timeago", filterArgs{}, "3 fortnights"},
		{"fuzzytime", filterArgs{}, "garbage"},
		{"jsonjoinarray", args("$.x", ","), "not json"},
		{"replace", args("", "x"), "abc"},
		{"re_replace", args("a", "{{ .Config.missing }}"), "abc"},
	}
	for _, tt := range tests {
		if got, err := runFilter(t, tt.filter, tt.args, textValue(tt.in)); err == nil {
			t.Errorf("%s(%q) = %q, want an error", tt.filter, tt.in, got.String())
		}
	}
}

func TestCompileFilterErrors(t *testing.T) {
	tests := []struct {
		filter string
		args   filterArgs
	}{
		{"querystring", filterArgs{}},
		{"querystring", args("a", "b")},
		{"dateparse", filterArgs{}},
		{"regexp", arg("(")},
		{"re_replace", arg("a")},
		{"split", args("/", "x")},
		{"split", args("", "1")},
		{"replace", arg("a")},
		{"trim", arg("")},
		{"diacritics", arg("remove")},
		{"jsonjoinarray", args("$.[", ",")},
		{"prepend", arg("{{ if .X }}")},
	}
	for _, tt := range tests {
		if _, err := compileFilter(tt.filter, tt.args); err == nil {
			t.Errorf("compileFilter(%s, %v): want an error", tt.filter, tt.args)
		}
	}
}

func TestFilterChain(t *testing.T) {
	var fs filterBlocks
	for _, f := range []struct {
		name string
		args filterArgs
	}{
		{"regexp", arg(`(\d+(?:\.\d+)?) (\w+)`)},
		{"append", arg(" GB")},
		{"replace", args(".", ",")},
	} {
		b, err := compileFilter(f.name, f.args)
		if err != nil {
			t.Fatal(err)
		}
		fs = append(fs, b)
	}
	got, err := fs.Apply(textValue("Size: 1.5 GB"), testEnv(t))
	if err != nil || got.String() != "1,5 GB" {
		t.Errorf("chain = %q, %v", got.String(), err)
	}
}
