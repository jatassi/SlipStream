package regex

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

var opts = Options{IgnoreCase: true, Timeout: time.Second}

func TestCompileErrorNamesPatternAndOffset(t *testing.T) {
	tests := []struct {
		pattern string
		offset  int
	}{
		{`a\qb`, 3},
		{`ab**`, 4},
		{`*a`, 1},
		{`a)b`, 2},
		{`[z-a]x`, 4},
		{`(a)\2x`, 5},
		{`a{3,2}x`, 6},
		{`(?<title>.+)(?Q)`, 15},
		{`日本\q`, 4},
		{`x\p{IsKlingon}y`, 14},
		{`[\P{IsGreek}]`, 12},
		{`(a`, 0},
		{`(?<title>.+)[a`, 0},
		{`a\`, 0},
	}
	for _, tt := range tests {
		_, err := Compile(tt.pattern, opts)
		var ce *CompileError
		if !errors.As(err, &ce) {
			t.Fatalf("Compile(%q) = %v; want a *CompileError", tt.pattern, err)
		}
		if ce.Pattern != tt.pattern || ce.Offset != tt.offset || ce.Reason == "" {
			t.Errorf("Compile(%q) = %+v; want offset %d", tt.pattern, *ce, tt.offset)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tt.pattern)) {
			t.Errorf("Compile(%q): %q doesn't name the pattern", tt.pattern, err)
		}
	}
}

func TestCatastrophicPatternTimesOut(t *testing.T) {
	const limit = 100 * time.Millisecond
	re := MustCompile(`z|(x+x+)+y`, Options{Timeout: limit})
	text := TextOf("z" + strings.Repeat("x", 40))
	calls := map[string]func() error{
		"Match":   func() error { _, err := re.Match(TextOf(strings.Repeat("x", 40))); return err },
		"Find":    func() error { _, _, err := re.Find(TextOf(strings.Repeat("x", 40))); return err },
		"FindAll": func() error { _, err := re.FindAll(text); return err },
		"Replace": func() error { _, err := re.Replace(text, "-"); return err },
	}
	for name, call := range calls {
		start := time.Now()
		err := call()
		elapsed := time.Since(start)
		var te *TimeoutError
		if !errors.As(err, &te) || te.Pattern != re.String() || te.Limit != limit {
			t.Errorf("%s = %v; want a *TimeoutError for %q at %v", name, err, re, limit)
		}
		if elapsed > 2*limit {
			t.Errorf("%s took %v; want under %v", name, elapsed, 2*limit)
		}
	}
}

func TestRepeatedGroupCaptures(t *testing.T) {
	re := MustCompile(`S(?<season>\d{2})(?:E(?<episode>\d{2}))+`, opts)
	m, ok, err := re.Find(TextOf("Show.s01e01E02e03.1080p"))
	if err != nil || !ok {
		t.Fatalf("Find = %v, %v; want a match", ok, err)
	}
	if got := m.Captures("episode"); !slices.Equal(got, []string{"01", "02", "03"}) {
		t.Errorf(`Captures("episode") = %q; want 01, 02, 03`, got)
	}
	if got, ok := m.Group("episode"); got != "03" || !ok {
		t.Errorf(`Group("episode") = %q, %v; want the last capture, 03`, got, ok)
	}
	if got, ok := m.Group("1"); got != "01" || !ok {
		t.Errorf(`Group("1") = %q, %v; want 01`, got, ok)
	}
	if m.Index() != 5 || m.Length() != 12 {
		t.Errorf("match at %d for %d; want 5 for 12", m.Index(), m.Length())
	}
}

func TestGroupThatCapturedNothing(t *testing.T) {
	m, _, _ := MustCompile(`a(?<b>b)?`, opts).Find(TextOf("a"))
	if got, ok := m.Group("b"); got != "" || ok {
		t.Errorf(`Group("b") = %q, %v; want "", false`, got, ok)
	}
	if got, ok := m.Group("nope"); got != "" || ok {
		t.Errorf(`Group("nope") = %q, %v; want "", false`, got, ok)
	}
}

func TestFindAll(t *testing.T) {
	ms, err := MustCompile(`\d+`, opts).FindAll(TextOf("1 22 333"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range ms {
		got = append(got, m.Value())
	}
	if !slices.Equal(got, []string{"1", "22", "333"}) {
		t.Errorf("FindAll = %q; want 1, 22, 333", got)
	}
}

func TestReplace(t *testing.T) {
	tests := []struct{ pattern, in, with, want string }{
		{`(\w+) (\w+)`, "abc def", "$2 $1", "def abc"},
		{`(?<a>\w+)`, "x y", "[${a}]", "[x] [y]"},
		{`a`, "aa", "$$", "$$"},
		{`b`, "abc", "$&$0", "abbc"},
		{`b`, "abc", "[$`|$']", "a[a|c]c"},
		{`(a)(b)?`, "ac", "$+", "c"},
		{`b`, "abc", "$_", "aabcc"},
		{`é`, "café é", "e", "cafe e"},
		{`x`, "abc", "y", "abc"},
	}
	for _, tt := range tests {
		got, err := MustCompile(tt.pattern, opts).Replace(TextOf(tt.in), tt.with)
		if err != nil || got != tt.want {
			t.Errorf("Replace(%q, %q, %q) = %q, %v; want %q", tt.pattern, tt.in, tt.with, got, err, tt.want)
		}
	}
}

func TestOptions(t *testing.T) {
	for _, tt := range []struct {
		pattern string
		o       Options
		want    bool
	}{
		{`remux`, Options{IgnoreCase: true, Timeout: time.Second}, true},
		{`remux`, Options{Timeout: time.Second}, false},
		{`re mux  # a comment`, Options{IgnoreCase: true, IgnorePatternWhitespace: true}, true},
		{`re mux`, Options{IgnoreCase: true}, false},
	} {
		if got, _ := MustCompile(tt.pattern, tt.o).Match(TextOf("REMUX")); got != tt.want {
			t.Errorf("%q with %+v matches REMUX: %v; want %v", tt.pattern, tt.o, got, tt.want)
		}
	}
}

func TestBlocks(t *testing.T) {
	tests := []struct{ pattern, in, want string }{
		{`[\p{IsCyrillic}]+`, "abc Привет", "Привет"},
		{`[^\p{IsBasicLatin}]+`, "abc日本", "日本"},
		{`\p{IsCJKUnifiedIdeographs}+`, "x日本y", "日本"},
		{`\P{IsBasicLatin}+`, "abcé", "é"},
	}
	for _, tt := range tests {
		re := MustCompile(tt.pattern, opts)
		m, _, err := re.Find(TextOf(tt.in))
		if err != nil || m.Value() != tt.want {
			t.Errorf("Find(%q, %q) = %q, %v; want %q", tt.pattern, tt.in, m.Value(), err, tt.want)
		}
		if re.String() != tt.pattern {
			t.Errorf("String() = %q; want the pattern as written, %q", re.String(), tt.pattern)
		}
	}
}

func TestCachedIsOnePerPatternAndOptions(t *testing.T) {
	a, err := Cached(`\bHDR10\+`, opts)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Cached(`\bHDR10\+`, opts)
	if a != b {
		t.Error("Cached twice with one pattern and Options gave two *Regex")
	}
	c, _ := Cached(`\bHDR10\+`, Options{Timeout: time.Second})
	if c == a {
		t.Error("Cached with other Options gave the same *Regex")
	}
	d, _ := Cached(`\bHDR10\b`, opts)
	if d == a {
		t.Error("Cached with another pattern gave the same *Regex")
	}
	var ce *CompileError
	if _, err := Cached(`(`, opts); !errors.As(err, &ce) {
		t.Errorf("Cached(%q) = %v; want a *CompileError", `(`, err)
	}
}

func TestCachedDropsTheLeastRecentlyUsed(t *testing.T) {
	first, _ := Cached(`first`, opts)
	kept, _ := Cached(`kept`, opts)
	for i := range cacheSize - 1 {
		if i%100 == 0 {
			_, _ = Cached(`kept`, opts)
		}
		_, _ = Cached(`p`+strconv.Itoa(i), opts)
	}
	if again, _ := Cached(`kept`, opts); again != kept {
		t.Error("a recently used pattern was dropped")
	}
	if again, _ := Cached(`first`, opts); again == first {
		t.Error("the least recently used pattern was kept past the bound")
	}
}
