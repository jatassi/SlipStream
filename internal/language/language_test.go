package language

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func TestParse(t *testing.T) {
	cases := []struct{ in, want string }{
		{"en", "en"}, {"eng", "en"}, {"EN", "en"}, {"fre", "fr"}, {"fra", "fr"}, {"ger", "de"},
		{"chi", "zh"}, {"zho", "zh"}, {"iw", "he"}, {"fil", "fil"}, {"yue", "yue"}, {"spa", "es"}, {"jpn", "ja"}, {"haw", "haw"},
		{"pt", "pt"}, {"pt-BR", "pt-BR"}, {"pt-br", "pt-BR"}, {"pt_BR", "pt-BR"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil || got.String() != c.want {
			t.Errorf("Parse(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
		if back, ok := ParseTag(got.Tag()); got.Tag() != c.want || !ok || back != got {
			t.Errorf("Parse(%q) has the tag %q, which reads back as %q, %v; want %q both ways", c.in, got.Tag(), back, ok, c.want)
		}
	}
	for _, in := range []string{"", "und", "mis", "mul", "zxx", "original", "en-US", "pt-PT", "zh-Hant", "sh", "sdh", "cc", "xx", "the", "english"} {
		if got, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %q; want a refusal", in, got)
		}
	}
}

func TestNames(t *testing.T) {
	cases := []struct{ in, name, native string }{
		{"en", "English", "English"},
		{"de", "German", "Deutsch"},
		{"fr", "French", "français"},
		{"ja", "Japanese", "日本語"},
		{"el", "Greek", "Ελληνικά"},
		{"pt", "Portuguese", "português"},
		{"pt-BR", "Brazilian Portuguese", "português"},
	}
	for _, c := range cases {
		code := must(t, c.in)
		if code.Name() != c.name || code.Native() != c.native {
			t.Errorf("%q names = %q, %q; want %q, %q", c.in, code.Name(), code.Native(), c.name, c.native)
		}
	}
}

func TestParseTag(t *testing.T) {
	cases := []struct{ in, want string }{
		{"en", "en"}, {"eng", "en"}, {"ger", "de"}, {"deu", "de"}, {"hi", "hi"},
		{"english", "en"}, {"French", "fr"}, {"pt-br", "pt-BR"}, {"pt-pt", "pt"},
		{"zh-cn", "zh"}, {"zh-tw", "zh"}, {"zh-hant", "zh"}, {"en-us", "en"},
	}
	for _, c := range cases {
		got, ok := ParseTag(c.in)
		if !ok || got.String() != c.want {
			t.Errorf("ParseTag(%q) = %q, %v; want %q", c.in, got, ok, c.want)
		}
	}
	for _, in := range []string{"sdh", "cc", "forced", "default", "foreign", "1080p", "x264", ""} {
		if got, ok := ParseTag(in); ok {
			t.Errorf("ParseTag(%q) = %q; want false", in, got)
		}
	}
}

func TestSpoken(t *testing.T) {
	audio := []release.Language{"", "pt"}
	for _, in := range []string{"pt", "pt-BR"} {
		if !must(t, in).Spoken(audio) {
			t.Errorf("%s isn't spoken by Portuguese audio", in)
		}
	}
	if must(t, "en").Spoken(audio) {
		t.Error("English is spoken by Portuguese and untagged audio")
	}
	if (Code{}).Spoken([]release.Language{""}) {
		t.Error("an untagged track speaks the zero Code")
	}
	if got := must(t, "pt-BR").Base(); got != "pt" {
		t.Errorf("pt-BR Base = %q; want pt", got)
	}
}

func must(t *testing.T, s string) Code {
	t.Helper()
	c, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
