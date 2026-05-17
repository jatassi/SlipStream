package auth

import (
	"strings"
	"testing"
)

func TestRequestHostname(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"bare hostname", "example.com", "example.com"},
		{"hostname with port", "example.com:8080", "example.com"},
		{"uppercase normalized", "Example.COM:443", "example.com"},
		{"ipv6 with port", "[::1]:8080", "::1"},
		{"ipv6 without port", "[::1]", "[::1]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requestHostname(tc.in)
			if got != tc.want {
				t.Fatalf("requestHostname(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizedOriginHost(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"http with port", "http://localhost:3000", "localhost"},
		{"https no port", "https://example.com", "example.com"},
		{"uppercase normalized", "https://Example.COM", "example.com"},
		{"ipv6", "https://[::1]:8443", "::1"},
		{"no scheme returns empty", "example.com:443", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizedOriginHost(tc.in)
			if got != tc.want {
				t.Fatalf("normalizedOriginHost(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveRPForRequest(t *testing.T) {
	svc := &PasskeyService{
		config: PasskeyConfig{
			RPID: "localhost",
			RPOrigins: []string{
				"http://localhost:3000",
				"https://slipstream.example.org",
			},
		},
	}

	t.Run("matches localhost ignoring port", func(t *testing.T) {
		rpID, origin, err := svc.resolveRPForRequest("localhost:3000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rpID != "localhost" || origin != "http://localhost:3000" {
			t.Fatalf("got (%q, %q), want (localhost, http://localhost:3000)", rpID, origin)
		}
	})

	t.Run("matches public host case-insensitively", func(t *testing.T) {
		rpID, origin, err := svc.resolveRPForRequest("Slipstream.Example.ORG")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rpID != "slipstream.example.org" || origin != "https://slipstream.example.org" {
			t.Fatalf("got (%q, %q), want (slipstream.example.org, https://slipstream.example.org)", rpID, origin)
		}
	})

	t.Run("rejects unknown host", func(t *testing.T) {
		_, _, err := svc.resolveRPForRequest("attacker.example.net")
		if err == nil {
			t.Fatal("expected error for unknown host")
		}
		if !strings.Contains(err.Error(), "attacker.example.net") {
			t.Fatalf("error should mention the unconfigured host, got: %v", err)
		}
	})

	t.Run("rejects empty host", func(t *testing.T) {
		_, _, err := svc.resolveRPForRequest("")
		if err == nil {
			t.Fatal("expected error for empty host")
		}
	})

	t.Run("rejects when no origins configured", func(t *testing.T) {
		empty := &PasskeyService{config: PasskeyConfig{RPID: "localhost"}}
		_, _, err := empty.resolveRPForRequest("localhost")
		if err == nil {
			t.Fatal("expected error when RPOrigins is empty")
		}
	})
}
