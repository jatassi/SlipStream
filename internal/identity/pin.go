package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PIN is exactly four ASCII digits. Only ParsePIN builds one.
type PIN struct{ digits string }

var errPINShape = errors.New("identity: a PIN is exactly four digits")

func ParsePIN(s string) (PIN, error) {
	if len(s) != 4 {
		return PIN{}, errPINShape
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return PIN{}, errPINShape
		}
	}
	return PIN{digits: s}, nil
}

const (
	argonTime    = 2
	argonMemory  = 19456
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
	// concurrentHashes bounds argon2id's memory: each hash takes 19 MiB.
	concurrentHashes = 2
)

var b64 = base64.RawStdEncoding

// pins hashes PINs as argon2id over HMAC-SHA256 under the pepper, so a copy
// of library.db without library.key reveals no PIN.
type pins struct {
	pepper []byte
	slots  chan struct{}
}

func newPINs(pepper []byte) *pins {
	return &pins{pepper: pepper, slots: make(chan struct{}, concurrentHashes)}
}

func (p *pins) hash(ctx context.Context, pin PIN) (string, error) {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	if err := p.acquire(ctx); err != nil {
		return "", err
	}
	defer p.release()
	return p.hashWithSalt(pin, salt), nil
}

func (p *pins) hashWithSalt(pin PIN, salt []byte) string {
	key := argon2.IDKey(p.peppered(pin), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

// verify reports whether pin matches phc, under the parameters phc records.
func (p *pins) verify(ctx context.Context, pin PIN, phc string) (bool, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false, fmt.Errorf("identity: %q is not an argon2id PHC string", phc)
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, fmt.Errorf("identity: argon2id parameters %q: %w", parts[3], err)
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("identity: argon2id salt: %w", err)
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("identity: argon2id hash: %w", err)
	}
	if err := p.acquire(ctx); err != nil {
		return false, err
	}
	defer p.release()
	got := argon2.IDKey(p.peppered(pin), salt, time, memory, threads, uint32(len(want))) //nolint:gosec // len(want) is a decoded hash of a few dozen bytes
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func (p *pins) peppered(pin PIN) []byte {
	mac := hmac.New(sha256.New, p.pepper)
	mac.Write([]byte(pin.digits))
	return mac.Sum(nil)
}

func (p *pins) acquire(ctx context.Context) error {
	select {
	case p.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("identity: wait to hash a PIN: %w", context.Cause(ctx))
	}
}

func (p *pins) release() { <-p.slots }
