// Package passkeytest is a software authenticator: it answers the options a
// passkey ceremony begins with, as a browser and a platform authenticator
// would at a given origin, so ceremonies are proven without a browser.
package passkeytest

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/fxamacker/cbor/v2"
)

var (
	// ErrExcluded is the browser's InvalidStateError: the authenticator
	// already holds one of the excluded credentials.
	ErrExcluded = errors.New("passkeytest: a credential in excludeCredentials is already here")
	// ErrNoCredential is the browser's NotAllowedError for an assertion the
	// authenticator holds no credential for.
	ErrNoCredential = errors.New("passkeytest: no credential for this request")
)

// Authenticator holds discoverable ES256 credentials, each bound to the RP
// ID it was created for.
type Authenticator struct {
	// AAGUID is the model it reports at registration.
	AAGUID [16]byte
	// Synced makes it behave like a synced passkey provider: backup
	// eligible and backed up, with a sign count that stays 0.
	Synced bool
	// Unverified answers without user verification, as when the user check
	// fails.
	Unverified bool

	creds []*credential
}

type credential struct {
	id     []byte
	key    *ecdsa.PrivateKey
	rpID   string
	handle []byte
	count  uint32
}

// Clone is a copy holding the same keys and counts, as a cloned
// authenticator would.
func (a *Authenticator) Clone() *Authenticator {
	c := *a
	c.creds = make([]*credential, len(a.creds))
	for i, cr := range a.creds {
		copied := *cr
		c.creds[i] = &copied
	}
	return &c
}

type creationOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		ID string `json:"id"`
	} `json:"rp"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	ExcludeCredentials []descriptor `json:"excludeCredentials"`
}

type requestOptions struct {
	Challenge        string       `json:"challenge"`
	RPID             string       `json:"rpId"`
	AllowCredentials []descriptor `json:"allowCredentials"`
}

type descriptor struct {
	ID string `json:"id"`
}

var b64 = base64.RawURLEncoding

const credentialIDLength = 16

// Register answers PublicKeyCredentialCreationOptionsJSON at origin with a
// RegistrationResponseJSON carrying "none" attestation.
func (a *Authenticator) Register(origin string, options []byte) ([]byte, error) {
	var o creationOptions
	if err := json.Unmarshal(options, &o); err != nil {
		return nil, fmt.Errorf("passkeytest: creation options: %w", err)
	}
	for _, ex := range o.ExcludeCredentials {
		if a.find(o.RP.ID, ex.ID) != nil {
			return nil, ErrExcluded
		}
	}
	handle, err := b64.DecodeString(o.User.ID)
	if err != nil {
		return nil, fmt.Errorf("passkeytest: user id: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("passkeytest: generate key: %w", err)
	}
	cr := &credential{id: make([]byte, credentialIDLength), key: key, rpID: o.RP.ID, handle: handle}
	_, _ = rand.Read(cr.id)
	a.creds = slices.DeleteFunc(a.creds, func(c *credential) bool { return c.rpID == cr.rpID && bytes.Equal(c.handle, handle) })
	a.creds = append(a.creds, cr)

	point, err := key.PublicKey.Bytes()
	if err != nil {
		return nil, fmt.Errorf("passkeytest: public key: %w", err)
	}
	cose, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: point[1:33], -3: point[33:]})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode key: %w", err)
	}
	authData := a.authData(cr, 0x40)
	authData = append(authData, a.AAGUID[:]...)
	authData = binary.BigEndian.AppendUint16(authData, credentialIDLength)
	authData = append(authData, cr.id...)
	authData = append(authData, cose...)
	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode attestation: %w", err)
	}
	data, err := clientData("webauthn.create", o.Challenge, origin)
	if err != nil {
		return nil, err
	}
	return answer(cr, map[string]any{
		"clientDataJSON":    b64.EncodeToString(data),
		"attestationObject": b64.EncodeToString(attestation),
		"transports":        []string{"hybrid", "internal"},
	})
}

// Assert answers PublicKeyCredentialRequestOptionsJSON at origin with an
// AuthenticationResponseJSON. With allowCredentials it uses the first one
// it holds; without, its newest credential for the RP ID.
func (a *Authenticator) Assert(origin string, options []byte) ([]byte, error) {
	var o requestOptions
	if err := json.Unmarshal(options, &o); err != nil {
		return nil, fmt.Errorf("passkeytest: request options: %w", err)
	}
	cr := a.pick(o)
	if cr == nil {
		return nil, ErrNoCredential
	}
	if !a.Synced {
		cr.count++
	}
	data, err := clientData("webauthn.get", o.Challenge, origin)
	if err != nil {
		return nil, err
	}
	authData := a.authData(cr, 0)
	digest := sha256.Sum256(slices.Concat(authData, sha256Of(data)))
	signature, err := ecdsa.SignASN1(rand.Reader, cr.key, digest[:])
	if err != nil {
		return nil, fmt.Errorf("passkeytest: sign: %w", err)
	}
	return answer(cr, map[string]any{
		"clientDataJSON":    b64.EncodeToString(data),
		"authenticatorData": b64.EncodeToString(authData),
		"signature":         b64.EncodeToString(signature),
		"userHandle":        b64.EncodeToString(cr.handle),
	})
}

func (a *Authenticator) pick(o requestOptions) *credential {
	if len(o.AllowCredentials) == 0 {
		for _, cr := range slices.Backward(a.creds) {
			if cr.rpID == o.RPID {
				return cr
			}
		}
		return nil
	}
	for _, allowed := range o.AllowCredentials {
		if cr := a.find(o.RPID, allowed.ID); cr != nil {
			return cr
		}
	}
	return nil
}

func (a *Authenticator) find(rpID, id string) *credential {
	for _, cr := range a.creds {
		if cr.rpID == rpID && b64.EncodeToString(cr.id) == id {
			return cr
		}
	}
	return nil
}

func (a *Authenticator) authData(cr *credential, extra byte) []byte {
	flags := byte(0x01) | extra
	if !a.Unverified {
		flags |= 0x04
	}
	if a.Synced {
		flags |= 0x08 | 0x10
	}
	out := sha256Of([]byte(cr.rpID))
	out = append(out, flags)
	return binary.BigEndian.AppendUint32(out, cr.count)
}

type collectedClientData struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}

func clientData(typ, challenge, origin string) ([]byte, error) {
	data, err := json.Marshal(collectedClientData{Type: typ, Challenge: challenge, Origin: origin})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode client data: %w", err)
	}
	return data, nil
}

func answer(cr *credential, response map[string]any) ([]byte, error) {
	id := b64.EncodeToString(cr.id)
	out, err := json.Marshal(map[string]any{
		"id":                      id,
		"rawId":                   id,
		"type":                    "public-key",
		"authenticatorAttachment": "platform",
		"clientExtensionResults":  map[string]any{},
		"response":                response,
	})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode response: %w", err)
	}
	return out, nil
}

func sha256Of(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}
