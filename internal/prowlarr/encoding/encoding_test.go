package encoding

import "testing"

func mustCharset(t *testing.T, name string) Charset {
	t.Helper()
	c, err := GetEncoding(name)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGetEncoding(t *testing.T) {
	for _, name := range []string{"UTF-8", "utf-8", "windows-1251", "ISO-8859-1", "iso-8859-2", "windows-1252", "tis-620", "windows-874"} {
		if _, err := GetEncoding(name); err != nil {
			t.Errorf("GetEncoding(%q): %v", name, err)
		}
	}
	if _, err := GetEncoding("klingon"); err == nil {
		t.Error("GetEncoding(klingon): want an error")
	}
	latin1 := mustCharset(t, "iso-8859-1")
	if got := latin1.GetString([]byte{0x80}); got != "\u0080" {
		t.Errorf("iso-8859-1 reads 0x80 as %q, want U+0080 as Latin-1 does", got)
	}
}

func TestURLEncode(t *testing.T) {
	utf8 := mustCharset(t, "UTF-8")
	cp1251 := mustCharset(t, "windows-1251")
	tests := []struct {
		name     string
		c        Charset
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
	if got := URLEncode("Привет"); got != "%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82" {
		t.Errorf("URLEncode = %q", got)
	}
}

func TestURLDecode(t *testing.T) {
	utf8 := mustCharset(t, "UTF-8")
	cp1251 := mustCharset(t, "windows-1251")
	tests := []struct {
		name     string
		c        Charset
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
	if got := cp1251.GetString([]byte{0xCF, 0xF0}); got != "Пр" {
		t.Errorf("GetString = %q", got)
	}
	if got := string(cp1251.GetBytes("Пр?")); got != "\xCF\xF0?" {
		t.Errorf("GetBytes = %q", got)
	}
	if got := mustCharset(t, "UTF-8").GetString([]byte("a\xffb")); got != "a�b" {
		t.Errorf("UTF-8 GetString = %q", got)
	}
}
