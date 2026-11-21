package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/system"
)

func TestSetupCodesReadAsTyped(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "setup_code_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct{ Input, Normalized string }
	if err := json.Unmarshal(src, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("no vectors")
	}
	for _, v := range vectors {
		if got := normalizeSetupCode(v.Input); got != v.Normalized {
			t.Errorf("normalize(%q) = %q, want %q", v.Input, got, v.Normalized)
		}
	}
}

func TestPINHashIsArgon2idOverThePepperedPIN(t *testing.T) {
	p := newPINs(bytes.Repeat([]byte{1}, 32))
	pin := mustPIN(t, "1234")
	phc := p.hashWithSalt(pin, bytes.Repeat([]byte{2}, 16))
	if want := "$argon2id$v=19$m=19456,t=2,p=1$AgICAgICAgICAgICAgICAg$zaGP0OJKWokTikRUfYGPJczjEl74Ds85Su9KfyOk0MI"; phc != want {
		t.Fatalf("hash = %s, want %s", phc, want)
	}
	for pinText, want := range map[string]bool{"1234": true, "1235": false} {
		ok, err := p.verify(t.Context(), mustPIN(t, pinText), phc)
		if err != nil || ok != want {
			t.Errorf("verify(%s) = %v, %v; want %v", pinText, ok, err, want)
		}
	}
	underB := newPINs(bytes.Repeat([]byte{3}, 32))
	if ok, err := underB.verify(t.Context(), pin, phc); err != nil || ok {
		t.Errorf("verify under another key = %v, %v; want false", ok, err)
	}
}

func TestParsePINTakesExactlyFourDigits(t *testing.T) {
	for _, s := range []string{"12a4", "123", "12345", "١٢٣٤", ""} {
		if _, err := ParsePIN(s); err == nil {
			t.Errorf("ParsePIN(%q) accepted it", s)
		}
	}
	if _, err := ParsePIN("0000"); err != nil {
		t.Errorf("ParsePIN(0000): %v", err)
	}
}

func TestAtMostTwoPINHashesRunAtOnce(t *testing.T) {
	p := newPINs(nil)
	for range concurrentHashes {
		if err := p.acquire(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := p.hash(ctx, mustPIN(t, "1234")); err == nil {
		t.Fatal("a third hash ran while two held the slots")
	}
}

type claimedFlag struct {
	mu    sync.Mutex
	claim system.Claim
}

func (c *claimedFlag) claimed(context.Context) (system.Claim, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.claim, nil
}

func (c *claimedFlag) set() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.claim = system.Controller
}

func TestTheGateOpensBusiesAndCloses(t *testing.T) {
	var log bytes.Buffer
	flag := &claimedFlag{}
	g, code, err := NewGate(t.Context(), flag.claimed, slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{ReplaceAttr: noTime})))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`).MatchString(code) {
		t.Fatalf("setup code %q is not XXXX-XXXX-XXXX in Crockford base32", code)
	}
	if want := `level=INFO msg="setup code" code=` + code + ` hint="open this Node in a browser to start a new SlipStream, join one or restore a Backup"` + "\n"; log.String() != want {
		t.Fatalf("log = %q, want %q", log.String(), want)
	}
	var c Caller
	ran := func(Claim) {}
	if err := g.Run(t.Context(), c, "0000-0000-0000", ran); !errors.Is(err, ErrSetupCodeWrong) {
		t.Fatalf("wrong code: %v, want ErrSetupCodeWrong", err)
	}
	if err := g.Run(t.Context(), Caller{kind: wrongAPIKey}, code, ran); !errors.Is(err, ErrAPIKeyWrong) {
		t.Fatalf("wrong API key: %v, want ErrAPIKeyWrong", err)
	}
	typed := strings.ToLower(strings.ReplaceAll(code, "-", " "))
	err = g.Run(t.Context(), c, typed, func(Claim) {
		if err := g.Run(t.Context(), c, code, ran); !errors.Is(err, ErrSetupBusy) {
			t.Errorf("while busy: %v, want ErrSetupBusy", err)
		}
	})
	if err != nil {
		t.Fatalf("lower case with spaces: %v", err)
	}
	if err := g.Run(t.Context(), c, code, func(Claim) { flag.set() }); err != nil {
		t.Fatalf("an unclaimed run did not reopen the gate: %v", err)
	}
	if err := g.Run(t.Context(), c, code, ran); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("after claiming: %v, want ErrAlreadyClaimed", err)
	}
	if !strings.HasSuffix(log.String(), "level=INFO msg=\"setup code used\"\n") {
		t.Fatalf("log = %q, want it to end with setup code used", log.String())
	}
}

func TestAClaimedNodeHasAClosedGateAndNoCode(t *testing.T) {
	for _, claim := range []system.Claim{system.Controller, system.Worker} {
		flag := &claimedFlag{claim: claim}
		g, code, err := NewGate(t.Context(), flag.claimed, slog.New(slog.DiscardHandler))
		if err != nil || code != "" {
			t.Fatalf("%s: NewGate = %q, %v; want no code", claim, code, err)
		}
		if err := g.Run(t.Context(), Caller{}, "", func(Claim) {}); !errors.Is(err, ErrAlreadyClaimed) {
			t.Fatalf("%s: Run = %v, want ErrAlreadyClaimed", claim, err)
		}
	}
}

func TestAClaimOutsideTheGateClosesIt(t *testing.T) {
	var log bytes.Buffer
	g, code, err := NewGate(t.Context(), (&claimedFlag{}).claimed, slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{ReplaceAttr: noTime})))
	if err != nil {
		t.Fatal(err)
	}
	log.Reset()
	g.Close()
	if err := g.Run(t.Context(), Caller{}, code, func(Claim) {}); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("Run with the code after Close = %v, want ErrAlreadyClaimed", err)
	}
	g.Close()
	if want := "level=INFO msg=\"setup code used\"\n"; log.String() != want {
		t.Fatalf("log = %q, want %q once", log.String(), want)
	}
}

func mustPIN(t *testing.T, s string) PIN {
	t.Helper()
	pin, err := ParsePIN(s)
	if err != nil {
		t.Fatal(err)
	}
	return pin
}

func noTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
