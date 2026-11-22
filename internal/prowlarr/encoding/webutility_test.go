package encoding

import "testing"

func TestHTMLDecode(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Tom &amp; Jerry", "Tom & Jerry"},
		{"&lt;b&gt;&quot;x&quot;&#39;", `<b>"x"'`},
		{"caf&eacute; &#233; &#xE9; &#XE9;", "café é é é"},
		{"&nbsp;", " "},
		{"AT&T", "AT&T"},
		{"&amp", "&amp"},
		{"&bogus;", "&bogus;"},
		{"&#xD800;", "&#xD800;"},
		{"&#x110000;", "&#x110000;"},
		{"a && b", "a && b"},
		{"Привет &mdash; мир", "Привет — мир"},
	}
	for _, tt := range tests {
		if got := HTMLDecode(tt.in); got != tt.want {
			t.Errorf("HTMLDecode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHTMLEncode(t *testing.T) {
	tests := []struct{ in, want string }{
		{`<a href="x">Tom & Jerry's</a>`, "&lt;a href=&quot;x&quot;&gt;Tom &amp; Jerry&#39;s&lt;/a&gt;"},
		{"café", "caf&#233;"},
		{"Привет", "Привет"},
		{"😀", "&#128512;"},
	}
	for _, tt := range tests {
		if got := HTMLEncode(tt.in); got != tt.want {
			t.Errorf("HTMLEncode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
