package artwork

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// Route is where the artwork route answers a token, under the API.
const Route = "/api/v1/artwork/"

var ErrForged = errors.New("SlipStream didn't issue that artwork token")

// Minter makes and opens artwork tokens: the reference in URL-safe base64,
// a dot, then the first 16 bytes of its HMAC-SHA256 under the install's
// artwork key. A new upstream path is a new token, so the image a token
// names never changes.
type Minter struct{ key []byte }

func NewMinter(key []byte) Minter { return Minter{key: key} }

const (
	tokenVersion = 1
	macBytes     = 16
)

var encoding = base64.RawURLEncoding.Strict()

// URL is the artwork route's address for r, without a size.
func (m Minter) URL(r Ref) string { return Route + m.Token(r) }

func (m Minter) Token(r Ref) string {
	payload := append([]byte{tokenVersion}, string(r.Source)+"\x00"+string(r.Shape)+"\x00"+r.Path...)
	return encoding.EncodeToString(payload) + "." + encoding.EncodeToString(m.mac(payload))
}

// Open is the reference token names, ErrForged for any token m didn't
// make.
func (m Minter) Open(token string) (Ref, error) {
	p, s, ok := strings.Cut(token, ".")
	payload, perr := encoding.DecodeString(p)
	sum, serr := encoding.DecodeString(s)
	if !ok || perr != nil || serr != nil || !hmac.Equal(sum, m.mac(payload)) || len(payload) == 0 || payload[0] != tokenVersion {
		return Ref{}, ErrForged
	}
	parts := strings.SplitN(string(payload[1:]), "\x00", 3)
	if len(parts) != 3 {
		return Ref{}, ErrForged
	}
	shape, err := ParseShape(parts[1])
	if err != nil {
		return Ref{}, ErrForged
	}
	return Ref{Source: SourceID(parts[0]), Shape: shape, Path: parts[2]}, nil
}

func (m Minter) mac(payload []byte) []byte {
	h := hmac.New(sha256.New, m.key)
	h.Write(payload)
	return h.Sum(nil)[:macBytes]
}
