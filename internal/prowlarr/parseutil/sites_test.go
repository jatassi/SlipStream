package parseutil

import (
	"maps"
	"testing"
)

func TestCoerceIntAsTheSitesCallIt(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"1,234", 1234, false},
		{" 17 ", 17, false},
		{"Go to files 3", 3, false},
		{"-", 0, false},
		{"", 0, false},
		{"2.147.483.647", 2147483647, false},
		{"2147483648", 0, true},
		{"1.5", 15, false},
		{"1,234.5", 0, true},
	}
	for _, tt := range tests {
		got, err := CoerceInt(tt.in)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("CoerceInt(%q) = %d, %v; want %d, error %v", tt.in, got, err, tt.want, tt.wantErr)
		}
		n, ok := TryCoerceInt(tt.in)
		if ok == tt.wantErr || n != tt.want {
			t.Errorf("TryCoerceInt(%q) = %d, %v", tt.in, n, ok)
		}
	}
}

func TestTryCoerce(t *testing.T) {
	if n, ok := TryCoerceLong("1762353000"); !ok || n != 1762353000 {
		t.Errorf("TryCoerceLong(1762353000) = %d, %v", n, ok)
	}
	if _, ok := TryCoerceLong("1,234.5"); ok {
		t.Error("TryCoerceLong(1,234.5): want false")
	}
	if f, ok := TryCoerceDouble("0,5"); !ok || f != 0.5 {
		t.Errorf("TryCoerceDouble(0,5) = %v, %v", f, ok)
	}
	if _, ok := TryCoerceDouble("."); ok {
		t.Error("TryCoerceDouble(.): want false")
	}
	if f, ok := TryCoerceFloat("2.5x"); !ok || f != 2.5 {
		t.Errorf("TryCoerceFloat(2.5x) = %v, %v", f, ok)
	}
}

func TestGetImdbID(t *testing.T) {
	tests := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"tt0111161", 111161, true},
		{"0111161", 111161, true},
		{"tt12345678", 12345678, true},
		{"tt123456789", 0, false},
		{"tt0111161\n", 111161, true},
		{"https://www.imdb.com/title/tt0111161", 0, false},
		{"TT0111161", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := GetImdbID(tt.in)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("GetImdbID(%q) = %d, %v; want %d, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestGetArgumentFromQueryString(t *testing.T) {
	tests := []struct {
		url, arg, want string
	}{
		{"details.php?id=42&hit=1", "id", "42"},
		{"/download.php?ID=7", "id", "7"},
		{"browse.php?cat=1&cat=2", "cat", "1"},
		{"t?name=a+b%20c#top", "name", "a b c"},
		{"t?flag&x=1", "flag", ""},
		{"t?q=%E9", "q", "%E9"},
	}
	for _, tt := range tests {
		got, err := GetArgumentFromQueryString(tt.url, tt.arg)
		if err != nil || got != tt.want {
			t.Errorf("GetArgumentFromQueryString(%q, %q) = %q, %v; want %q", tt.url, tt.arg, got, err, tt.want)
		}
	}
	for _, tt := range []struct{ url, arg string }{{"details.php", "id"}, {"details.php?x=1", "id"}} {
		if got, err := GetArgumentFromQueryString(tt.url, tt.arg); err == nil {
			t.Errorf("GetArgumentFromQueryString(%q, %q) = %q; want an error", tt.url, tt.arg, got)
		}
	}
}

func TestCookieHeaderToDictionary(t *testing.T) {
	tests := []struct {
		in   string
		want map[string]string
	}{
		{"uid=1; pass=abc", map[string]string{"uid": "1", "pass": "abc"}},
		{"session=x; Path=/; Domain=.site.org; Expires=Wed, 21 Oct 2026 07:28:00 GMT; HttpOnly; SameSite=Lax", map[string]string{"session": "x"}},
		{"a=1; a=2", map[string]string{"a": "2"}},
		{"cf_clearance=x.y-z; __cfduid=abc", map[string]string{"cf_clearance": "x.y-z", "__cfduid": "abc"}},
		{"a=\"quoted\"; b=ok", map[string]string{"b": "ok"}},
		{"a=1 b=2", map[string]string{"a": "1", "b": "2"}},
		{"", map[string]string{}},
	}
	for _, tt := range tests {
		if got := CookieHeaderToDictionary(tt.in); !maps.Equal(got, tt.want) {
			t.Errorf("CookieHeaderToDictionary(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestGetBytesAsTheSitesWriteIt(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"1.37 GB", 1471026298},
		{"700.5 MB", 734527488},
		{"4,5 GiB", 4831838208},
		{"1,234.56 MB", 1294529986},
		{"12 KB", 12288},
		{"", 0},
	}
	for _, tt := range tests {
		got, err := GetBytes(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("GetBytes(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
}
