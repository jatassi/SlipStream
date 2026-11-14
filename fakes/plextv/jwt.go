package plextv

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

// jwk is a device's public key as a PIN carries it.
type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	key ed25519.PublicKey
}

func (k *jwk) parse() error {
	x, err := base64.RawURLEncoding.DecodeString(k.X)
	if k.Kty != "OKP" || k.Crv != "Ed25519" || k.Kid == "" || err != nil || len(x) != ed25519.PublicKeySize {
		return errors.New("jwk is not an Ed25519 key with a kid")
	}
	k.key = x
	return nil
}

func signJWT(key ed25519.PrivateKey, claims map[string]any) string {
	pub, _ := key.Public().(ed25519.PublicKey)
	header := mustJSON(map[string]string{"alg": "EdDSA", "kid": base64.RawURLEncoding.EncodeToString(pub[:16]), "typ": "JWT"})
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(mustJSON(claims))
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(input)))
}

// mustJSON is v encoded. The fake encodes only maps of strings, numbers
// and lists, which always encode.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// audience is a JWT's aud, one string or a list.
type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*a = audience{one}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(a)) //nolint:wrapcheck // the decoder names the field
}

// deviceClaims are what plex.tv reads from a device JWT.
type deviceClaims struct {
	Aud   audience `json:"aud"`
	Iss   string   `json:"iss"`
	Exp   int64    `json:"exp"`
	Nonce string   `json:"nonce"`
}

// kidOf is a JWT's kid, before plex.tv checks its signature.
func kidOf(token string) string {
	header, _, ok := strings.Cut(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(header)
	if !ok || err != nil {
		return ""
	}
	var h struct {
		Kid string `json:"kid"`
	}
	_ = json.Unmarshal(raw, &h)
	return h.Kid
}

// verifyDevice is token's claims when key signed it for plex.tv, from
// issuer clientID, and it hasn't expired.
func verifyDevice(token string, key ed25519.PublicKey, clientID string, now time.Time) (deviceClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return deviceClaims{}, errors.New("device JWT is not a JWS")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), sig) {
		return deviceClaims{}, errors.New("device JWT signature doesn't match the device key")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	var c deviceClaims
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return deviceClaims{}, errors.New("device JWT claims don't decode")
	}
	if !slices.Contains(c.Aud, "plex.tv") || c.Iss != clientID || !now.Before(time.Unix(c.Exp, 0)) {
		return deviceClaims{}, errors.New("device JWT is not for plex.tv from this client, or it expired")
	}
	return c, nil
}
