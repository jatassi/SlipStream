package metadata

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/internal/db"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("metadata", schemaFiles)
}

// bundled is the keys the release workflow's tag build carries, as
// "<provider>:<key>" pairs joined by commas. Every other build has none.
var bundled string

func parseBundled(s string) map[ProviderID]string {
	keys := map[ProviderID]string{}
	if s == "" {
		return keys
	}
	for pair := range strings.SplitSeq(s, ",") {
		p, key, ok := strings.Cut(pair, ":")
		if !ok || p == "" || key == "" {
			panic(fmt.Sprintf("metadata: the bundled keys hold %q, not <provider>:<key>", pair))
		}
		keys[ProviderID(p)] = key
	}
	return keys
}

// KeyInUse is whose key a provider is called with.
type KeyInUse uint8

const (
	// KeyNone: this build carries no key and no Admin set one, so calls go
	// without one.
	KeyNone KeyInUse = iota + 1
	// KeyBundled is SlipStream's own key, which every install shares.
	KeyBundled
	// KeyYours is the key an Admin set.
	KeyYours
)

var (
	ErrNoProvider = errors.New("no metadata provider has that id")
	// ErrKeyRefused is a key the provider refused when tested; nothing was
	// stored.
	ErrKeyRefused = errors.New("the provider refused the key")
	// ErrPinRefused is a PIN the provider refused beside a key it takes;
	// nothing was stored.
	ErrPinRefused = errors.New("the provider refused the PIN")
	// ErrTakesNoPin is a PIN sent to a provider that takes none.
	ErrTakesNoPin = errors.New("the provider takes no PIN")
)

var (
	keyColumn = secret.NewColumn("metadata_key.api_key_sealed")
	pinColumn = secret.NewColumn("metadata_key.pin_sealed")
)

// NewKey is what Set key sends: the Admin's key, and the PIN beside it,
// empty for none.
type NewKey struct {
	Key string
	Pin string
}

// SetKey tests k on a holder and keeps it, sealed, only when the provider
// takes it: a *works.SourceError wrapping ErrKeyRefused or ErrPinRefused
// for the one it refuses, and wrapping works.ErrUnavailable when the test
// couldn't tell. Nothing untested is stored. It answers the provider's
// status with the key in use.
func (m *Metadata) SetKey(ctx context.Context, _ identity.Admin, p ProviderID, k NewKey) (ProviderStatus, error) {
	label, ok := m.label(p)
	if !ok {
		return ProviderStatus{}, ErrNoProvider
	}
	if k.Pin != "" && !m.statusOf(p).TakesPin {
		return ProviderStatus{}, &works.SourceError{Source: label, Words: "takes no PIN", Err: ErrTakesNoPin}
	}
	ask := Ask{keys: map[ProviderID]string{p: k.Key}, fresh: true, generation: m.cacheGeneration()}
	if k.Pin != "" {
		ask.pins = map[ProviderID]string{p: k.Pin}
	}
	test := &metadatav1.TestRequest{Ask: askToWire(ask), Provider: string(p)}
	res, node, err := call(ctx, m, role.NodeID{}, func(ctx context.Context, s metadatav1connect.MetadataServiceHandler) (*metadatav1.TestResponse, error) {
		return s.Test(ctx, test)
	})
	if err != nil {
		return ProviderStatus{}, err
	}
	outcomes := outcomesOf(res.GetOutcomes())
	o, ok := lastOf(outcomes, p)
	switch {
	case !ok:
		return ProviderStatus{}, &works.SourceError{Source: label, Words: "returned no test result", Err: works.ErrUnavailable}
	case o.Kind == Refused:
		return ProviderStatus{}, &works.SourceError{Source: label, Words: o.Words, Err: ErrKeyRefused}
	case o.Kind == PinRefused:
		return ProviderStatus{}, &works.SourceError{Source: label, Words: o.Words, Err: ErrPinRefused}
	case o.Kind == Limited || o.Kind == Unreachable:
		return ProviderStatus{}, &works.SourceError{Source: label, Until: o.Until, Words: o.Words, Err: works.ErrUnavailable}
	}
	row := db.PutKeyParams{Provider: string(p), ApiKeySealed: m.cfg.Sealer.Seal([]byte(k.Key), keyColumn.Row(string(p))), SetAt: storage.Now()}
	if k.Pin != "" {
		pin := m.cfg.Sealer.Seal([]byte(k.Pin), pinColumn.Row(string(p)))
		row.PinSealed = &pin
	}
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).PutKey(ctx, row)
	})
	if err != nil {
		return ProviderStatus{}, err
	}
	m.rekeyed(ctx, p, yourKey{key: k.Key, pin: k.Pin, setAt: row.SetAt.Time()}, node, outcomes)
	return m.statusOf(p), nil
}

// ClearKey drops the Admin's key and its PIN for p, so calls go with
// SlipStream's key again, or none in a build without one.
func (m *Metadata) ClearKey(ctx context.Context, _ identity.Admin, p ProviderID) (ProviderStatus, error) {
	if _, ok := m.label(p); !ok {
		return ProviderStatus{}, ErrNoProvider
	}
	var cleared int64
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		cleared, err = db.New(tx).DeleteKey(ctx, string(p))
		return err
	})
	if err != nil {
		return ProviderStatus{}, err
	}
	if cleared > 0 {
		m.rekeyed(ctx, p, yourKey{}, role.NodeID{}, nil)
	}
	return m.statusOf(p), nil
}

// yourKey is an Admin's key, the PIN beside it and when they set them.
type yourKey struct {
	key, pin string
	setAt    time.Time
}

// inEffect is the key each provider is called with, and the PIN beside
// it. SlipStream's own keys take none.
func (m *Metadata) inEffect() (keys, pins map[ProviderID]string) {
	keys, pins = maps.Clone(m.bundled), map[ProviderID]string{}
	m.mu.Lock()
	defer m.mu.Unlock()
	for p, k := range m.yours {
		keys[p] = k.key
		if k.pin != "" {
			pins[p] = k.pin
		}
	}
	return keys, pins
}

// keyInUse is whose key p is called with; m.mu is held.
func (m *Metadata) keyInUse(p ProviderID) KeyInUse {
	switch {
	case m.yours[p].key != "":
		return KeyYours
	case m.bundled[p] != "":
		return KeyBundled
	}
	return KeyNone
}
