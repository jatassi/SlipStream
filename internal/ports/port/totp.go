package port

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: RFC 6238 codes are HMAC-SHA-1, as every authenticator app makes them
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// totp is the RFC 6238 code of key at now: HMAC-SHA-1 over 30 s steps,
// six digits. key is the base32 text an authenticator app takes, in any
// case, with or without spaces and padding.
func totp(key string, now time.Time) (string, error) {
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimRight(strings.ReplaceAll(key, " ", ""), "=")))
	if err != nil {
		return "", fmt.Errorf("the authenticator key isn't base32: %w", err)
	}
	return hotp(secret, uint64(now.Unix()/30)), nil //nolint:gosec // G115: a time step after 1970 is positive
}

// hotp is RFC 4226's code of secret at counter, six digits.
func hotp(secret []byte, counter uint64) string {
	mac := hmac.New(sha1.New, secret)
	_ = binary.Write(mac, binary.BigEndian, counter)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%1_000_000)
}
