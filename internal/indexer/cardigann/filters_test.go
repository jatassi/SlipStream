package cardigann

import (
	"testing"
	"time"
)

func TestApplyFilters(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		filters []Filter
		want    string
		wantErr bool
	}{
		{
			name:    "no filters",
			value:   "hello",
			filters: nil,
			want:    "hello",
		},
		{
			name:  "single replace filter",
			value: "hello world",
			filters: []Filter{
				{Name: "replace", Args: []string{"world", "there"}},
			},
			want: "hello there",
		},
		{
			name:  "chained filters",
			value: "  HELLO WORLD  ",
			filters: []Filter{
				{Name: "trim", Args: nil},
				{Name: "tolower", Args: nil},
			},
			want: "hello world",
		},
		{
			name:  "prepend and append",
			value: "world",
			filters: []Filter{
				{Name: "prepend", Args: []string{"hello "}},
				{Name: "append", Args: []string{"!"}},
			},
			want: "hello world!",
		},
		{
			name:  "unknown filter is skipped",
			value: "test",
			filters: []Filter{
				{Name: "unknownfilter", Args: nil},
			},
			want: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyFilters(tt.value, tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ApplyFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterReplace(t *testing.T) {
	tests := []struct {
		name  string
		value string
		args  []string
		want  string
	}{
		{
			name:  "basic replace",
			value: "hello world",
			args:  []string{"world", "there"},
			want:  "hello there",
		},
		{
			name:  "no match",
			value: "hello world",
			args:  []string{"xyz", "abc"},
			want:  "hello world",
		},
		{
			name:  "multiple occurrences",
			value: "hello hello hello",
			args:  []string{"hello", "hi"},
			want:  "hi hi hi",
		},
		{
			name:  "missing args",
			value: "hello",
			args:  []string{"one"},
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterReplace(tt.value, tt.args)
			if err != nil {
				t.Errorf("filterReplace() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("filterReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterReReplace(t *testing.T) {
	tests := []struct {
		name  string
		value string
		args  []string
		want  string
	}{
		{
			name:  "replace digits",
			value: "test123abc456",
			args:  []string{"[0-9]+", "X"},
			want:  "testXabcX",
		},
		{
			name:  "replace whitespace",
			value: "hello   world",
			args:  []string{`\s+`, " "},
			want:  "hello world",
		},
		{
			name:  "invalid regex",
			value: "test",
			args:  []string{"[invalid", "X"},
			want:  "test", // Invalid regex is skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterReReplace(tt.value, tt.args)
			if err != nil {
				t.Errorf("filterReReplace() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("filterReReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterSplit(t *testing.T) {
	tests := []struct {
		name  string
		value string
		args  []string
		want  string
	}{
		{
			name:  "first element",
			value: "a,b,c,d",
			args:  []string{",", "0"},
			want:  "a",
		},
		{
			name:  "second element",
			value: "a,b,c,d",
			args:  []string{",", "1"},
			want:  "b",
		},
		{
			name:  "last element with negative index",
			value: "a,b,c,d",
			args:  []string{",", "-1"},
			want:  "d",
		},
		{
			name:  "out of bounds index",
			value: "a,b,c",
			args:  []string{",", "10"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterSplit(tt.value, tt.args)
			if err != nil {
				t.Errorf("filterSplit() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("filterSplit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterTrim(t *testing.T) {
	tests := []struct {
		name  string
		value string
		args  []string
		want  string
	}{
		{
			name:  "default whitespace trim",
			value: "  hello  ",
			args:  nil,
			want:  "hello",
		},
		{
			name:  "custom character trim",
			value: "---hello---",
			args:  []string{"-"},
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterTrim(tt.value, tt.args)
			if err != nil {
				t.Errorf("filterTrim() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("filterTrim() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterURLEncodeDecode(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		encode string
	}{
		{
			name:   "basic encoding",
			value:  "hello world",
			encode: "hello+world",
		},
		{
			name:   "special characters",
			value:  "a=b&c=d",
			encode: "a%3Db%26c%3Dd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := filterURLEncode(tt.value, nil)
			if err != nil {
				t.Errorf("filterURLEncode() error = %v", err)
				return
			}
			if encoded != tt.encode {
				t.Errorf("filterURLEncode() = %v, want %v", encoded, tt.encode)
			}

			decoded, err := filterURLDecode(encoded, nil)
			if err != nil {
				t.Errorf("filterURLDecode() error = %v", err)
				return
			}
			if decoded != tt.value {
				t.Errorf("filterURLDecode() = %v, want %v", decoded, tt.value)
			}
		})
	}
}

func TestFilterSize(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "bytes",
			value: "100",
		},
		{
			name:  "KB",
			value: "1 KB",
		},
		{
			name:  "MB",
			value: "5 MB",
		},
		{
			name:  "GB",
			value: "2 GB",
		},
		{
			name:  "decimal value",
			value: "1.5 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterSize(tt.value, nil)
			if err != nil {
				t.Errorf("filterSize() error = %v", err)
				return
			}
			// Just verify we get a non-empty result
			if got == "" {
				t.Errorf("filterSize() returned empty string")
			}
		})
	}
}

func TestFilterTimeAgo(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{
			name:  "minutes ago",
			value: "5 minutes ago",
			valid: true,
		},
		{
			name:  "hours ago",
			value: "2 hours ago",
			valid: true,
		},
		{
			name:  "days ago",
			value: "3 days ago",
			valid: true,
		},
		{
			name:  "today",
			value: "Today",
			valid: true,
		},
		{
			name:  "yesterday",
			value: "Yesterday",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterTimeAgo(tt.value, nil)
			if err != nil && tt.valid {
				t.Errorf("filterTimeAgo() error = %v", err)
				return
			}
			if tt.valid {
				// Verify it's a valid time string
				_, err := time.Parse(time.RFC3339, got)
				if err != nil {
					t.Errorf("filterTimeAgo() returned invalid time format: %v", got)
				}
			}
		})
	}
}

func TestFilterHTMLEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		encoded string
	}{
		{
			name:    "basic HTML entities",
			value:   "<div>Hello & World</div>",
			encoded: "&lt;div&gt;Hello &amp; World&lt;/div&gt;",
		},
		{
			name:    "quotes",
			value:   `"hello"`,
			encoded: "&#34;hello&#34;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := filterHTMLEncode(tt.value, nil)
			if err != nil {
				t.Errorf("filterHTMLEncode() error = %v", err)
				return
			}
			if encoded != tt.encoded {
				t.Errorf("filterHTMLEncode() = %v, want %v", encoded, tt.encoded)
			}

			decoded, err := filterHTMLDecode(encoded, nil)
			if err != nil {
				t.Errorf("filterHTMLDecode() error = %v", err)
				return
			}
			if decoded != tt.value {
				t.Errorf("filterHTMLDecode() = %v, want %v", decoded, tt.value)
			}
		})
	}
}

func TestFilterRegexp(t *testing.T) {
	tests := []struct {
		name  string
		value string
		args  []string
		want  string
	}{
		{
			name:  "extract group",
			value: "Size: 1.5 GB",
			args:  []string{`Size:\s*([\d.]+\s*\w+)`},
			want:  "1.5 GB",
		},
		{
			name:  "no match",
			value: "no match here",
			args:  []string{`not found: (\w+)`},
			want:  "",
		},
		{
			name:  "invalid regex",
			value: "test",
			args:  []string{`[invalid`},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterRegexp(tt.value, tt.args)
			if err != nil {
				t.Errorf("filterRegexp() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("filterRegexp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterMultiplyDivide(t *testing.T) {
	t.Run("multiply", func(t *testing.T) {
		got, err := filterMultiply("10", []string{"5"})
		if err != nil {
			t.Errorf("filterMultiply() error = %v", err)
			return
		}
		// Result may be formatted as float (e.g., "50.000000" or "50")
		if got != "50" && got != "50.000000" {
			t.Errorf("filterMultiply() = %v, want 50 or 50.000000", got)
		}
	})

	t.Run("divide", func(t *testing.T) {
		got, err := filterDivide("100", []string{"4"})
		if err != nil {
			t.Errorf("filterDivide() error = %v", err)
			return
		}
		// Result may be formatted as float (e.g., "25.000000" or "25")
		if got != "25" && got != "25.000000" {
			t.Errorf("filterDivide() = %v, want 25 or 25.000000", got)
		}
	})
}

func TestNormalizeFilterArgs(t *testing.T) {
	tests := []struct {
		name string
		args interface{}
		want []string
	}{
		{
			name: "nil",
			args: nil,
			want: nil,
		},
		{
			name: "string",
			args: "single",
			want: []string{"single"},
		},
		{
			name: "string slice",
			args: []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "interface slice",
			args: []interface{}{"a", 123},
			want: []string{"a", "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFilterArgs(tt.args)
			if tt.want == nil && got != nil {
				t.Errorf("normalizeFilterArgs() = %v, want nil", got)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("normalizeFilterArgs() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normalizeFilterArgs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
