package cardigann

import "testing"

func mustCharset(t *testing.T, name string) charset {
	t.Helper()
	c, err := lookupCharset(name)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLookupCharset(t *testing.T) {
	for _, name := range []string{"UTF-8", "utf-8", "windows-1251", "ISO-8859-1", "iso-8859-2", "windows-1252", "tis-620", "windows-874"} {
		if _, err := lookupCharset(name); err != nil {
			t.Errorf("lookupCharset(%q): %v", name, err)
		}
	}
	if _, err := lookupCharset("klingon"); err == nil {
		t.Error("lookupCharset(klingon): want an error")
	}
	latin1 := mustCharset(t, "iso-8859-1")
	if got := latin1.Decode([]byte{0x80}); got != "\u0080" {
		t.Errorf("iso-8859-1 reads 0x80 as %q, want U+0080 as Latin-1 does", got)
	}
}

func TestURLEncode(t *testing.T) {
	utf8 := mustCharset(t, "UTF-8")
	cp1251 := mustCharset(t, "windows-1251")
	tests := []struct {
		name     string
		c        charset
		in, want string
	}{
		{"UTF-8", utf8, "a b&c/é", "a+b%26c%2F%C3%A9"},
		{"UTF-8", utf8, "-_.!*()", "-_.!*()"},
		{"UTF-8", utf8, "~'", "%7E%27"},
		{"windows-1251", cp1251, "Привет мир", "%CF%F0%E8%E2%E5%F2+%EC%E8%F0"},
		{"windows-1251", cp1251, "é", "%3F"},
	}
	for _, tt := range tests {
		if got := tt.c.URLEncode(tt.in); got != tt.want {
			t.Errorf("%s URLEncode(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
	if got := urlEncodeUTF8("Привет"); got != "%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82" {
		t.Errorf("urlEncodeUTF8 = %q", got)
	}
}

func TestURLDecode(t *testing.T) {
	utf8 := mustCharset(t, "UTF-8")
	cp1251 := mustCharset(t, "windows-1251")
	tests := []struct {
		name     string
		c        charset
		in, want string
	}{
		{"UTF-8", utf8, "a+b%26c%2f%C3%A9", "a b&c/é"},
		{"UTF-8", utf8, "100%", "100%"},
		{"UTF-8", utf8, "%zz%4", "%zz%4"},
		{"windows-1251", cp1251, "%CF%F0", "Пр"},
	}
	for _, tt := range tests {
		if got := tt.c.URLDecode(tt.in); got != tt.want {
			t.Errorf("%s URLDecode(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestCharsetCodec(t *testing.T) {
	cp1251 := mustCharset(t, "windows-1251")
	if got := cp1251.Decode([]byte{0xCF, 0xF0}); got != "Пр" {
		t.Errorf("Decode = %q", got)
	}
	if got := string(cp1251.Encode("Пр?")); got != "\xCF\xF0?" {
		t.Errorf("Encode = %q", got)
	}
	if got := mustCharset(t, "UTF-8").Decode([]byte("a\xffb")); got != "a�b" {
		t.Errorf("UTF-8 Decode = %q", got)
	}
}

func TestQueryString(t *testing.T) {
	got := mustCharset(t, "UTF-8").QueryString([]pair{{"q", "a b"}, {"cat[]", "1"}, {"cat[]", "2"}}, "&")
	if want := "q=a+b&cat[]=1&cat[]=2"; got != want {
		t.Errorf("QueryString = %q, want %q", got, want)
	}
	got = mustCharset(t, "UTF-8").QueryString([]pair{{"a", "1"}, {"b", ""}}, ";")
	if want := "a=1;b="; got != want {
		t.Errorf("QueryString = %q, want %q", got, want)
	}
}
