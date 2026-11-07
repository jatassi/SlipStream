package cardigann

import "testing"

func TestCoerceLong(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "1234", want: 1234},
		{in: "1,234", want: 1234},
		{in: "1.234", want: 1234},
		{in: "1,234.00", want: 1234},
		{in: "12 seeders", want: 12},
		{in: "-5", want: 5},
		{in: "", want: 0},
		{in: "none", want: 0},
		{in: "5,5", want: 55},
		{in: "1,234.56", wantErr: true},
		{in: "٣", wantErr: true},
		{in: "99999999999999999999", wantErr: true},
	}
	for _, tt := range tests {
		got, err := coerceLong(tt.in)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("coerceLong(%q) = %d, %v; want %d, error %v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
	if _, err := coerceInt("3000000000"); err == nil {
		t.Error("coerceInt(3000000000): want an error")
	}
	if got, err := coerceInt("2,000"); err != nil || got != 2000 {
		t.Errorf("coerceInt(2,000) = %d, %v", got, err)
	}
}

func TestCoerceDouble(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"1.5", 1.5},
		{"1,5", 1.5},
		{"1.234,56", 1234.56},
		{"1,234.56", 1234.56},
		{"ratio 0.75x", 0.75},
		{"", 0},
		{".5", 0.5},
	}
	for _, tt := range tests {
		got, err := coerceDouble(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("coerceDouble(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
	if _, err := coerceDouble("."); err == nil {
		t.Error(`coerceDouble("."): want an error`)
	}
	if got, _ := coerceFloat("0.1"); got != float64(float32(0.1)) {
		t.Errorf("coerceFloat(0.1) = %v, want float32 precision", got)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"1.5 GB", 1610612736},
		{"1,5 GiB", 1610612736},
		{"500 KB", 512000},
		{"10 Kb", 10240},
		{"700 MiB", 734003200},
		{"2 TB", 2199023255552},
		{"100", 100},
		{"1 B", 1},
		{"3.7 Go", 3},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("parseSize(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
}

func TestLongFromString(t *testing.T) {
	tests := []struct {
		in     string
		want   int64
		wantOK bool
	}{
		{"abc 123 def 456", 123, true},
		{"id=0042", 42, true},
		{"none", 0, true},
		{"  ", 0, false},
	}
	for _, tt := range tests {
		got, ok, err := longFromString(tt.in)
		if err != nil || got != tt.want || ok != tt.wantOK {
			t.Errorf("longFromString(%q) = %d, %v, %v; want %d, %v", tt.in, got, ok, err, tt.want, tt.wantOK)
		}
	}
}
