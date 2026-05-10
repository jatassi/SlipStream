package netutil

import "testing"

func TestNormalizeLoopbackHost(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"localhost", "127.0.0.1"},
		{"LOCALHOST", "127.0.0.1"},
		{"Localhost", "127.0.0.1"},
		{"127.0.0.1", "127.0.0.1"},
		{"::1", "::1"},
		{"example.com", "example.com"},
		{"localhost.localdomain", "localhost.localdomain"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeLoopbackHost(tt.in); got != tt.want {
			t.Errorf("NormalizeLoopbackHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeLoopbackURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"http://localhost:9091/transmission/rpc", "http://127.0.0.1:9091/transmission/rpc"},
		{"https://localhost:9696", "https://127.0.0.1:9696"},
		{"http://LOCALHOST:8080/path?q=1", "http://127.0.0.1:8080/path?q=1"},
		{"http://localhost", "http://127.0.0.1"},
		{"http://user:pass@localhost:9091/p", "http://user:pass@127.0.0.1:9091/p"},
		{"http://127.0.0.1:9091/x", "http://127.0.0.1:9091/x"},
		{"http://[::1]:9091/x", "http://[::1]:9091/x"},
		{"http://example.com:9091/x", "http://example.com:9091/x"},
		{"not a url", "not a url"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeLoopbackURL(tt.in); got != tt.want {
			t.Errorf("NormalizeLoopbackURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
