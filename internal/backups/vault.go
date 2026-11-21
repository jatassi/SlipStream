// Package backups takes, keeps and restores the Controller's Backups.
package backups

import (
	"bytes"
	"cmp"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/secret"
)

// VaultFile is backup.key's name in the data dir, beside library.key.
const VaultFile = "backup.key"

type Generation uint32

// Vault owns backup.key: each passphrase generation still in use and one
// fence secret with its own generation. It is the file's only writer.
type Vault struct {
	mu     sync.Mutex
	path   string
	retry  time.Duration
	sealer *secret.Sealer
	ring   keyring
	raw    []byte
}

// keyring is backup.key decoded. Its passphrases ascend by generation, and
// the last one encrypts new Backups. A fence generation of 0 means the
// file doesn't exist yet.
type keyring struct {
	passphrases []generation
	fence       fenceSecret
}

type generation struct {
	gen  Generation
	pass archive.Passphrase
}

type fenceSecret struct {
	gen    uint32
	secret [32]byte
}

// OpenVault reads path. A missing file is an empty Vault: the first
// Current makes generation 1 and a fence secret. retry is the Windows
// rename retry.
func OpenVault(path string, keys *secret.Keys, retry time.Duration) (*Vault, error) {
	v := &Vault{path: path, retry: retry, sealer: keys.Sealer()}
	b, err := os.ReadFile(path) //nolint:gosec // path is the data dir's backup.key, chosen by the Node
	if errors.Is(err, fs.ErrNotExist) {
		return v, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", VaultFile, err)
	}
	if v.ring, err = decode(b, v.sealer); err != nil {
		return nil, err
	}
	v.raw = b
	return v, nil
}

// Current is the passphrase that encrypts new Backups.
func (v *Vault) Current(rand io.Reader) (Generation, archive.Passphrase, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.ring.passphrases) == 0 {
		first := keyring{passphrases: []generation{{gen: 1, pass: archive.NewPassphrase(rand)}}, fence: newFence(rand, 1)}
		if err := v.commit(first); err != nil {
			return 0, archive.Passphrase{}, err
		}
	}
	last := v.ring.passphrases[len(v.ring.passphrases)-1]
	return last.gen, last.pass, nil
}

func (v *Vault) Passphrase(g Generation) (archive.Passphrase, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	i := slices.IndexFunc(v.ring.passphrases, func(p generation) bool { return p.gen == g })
	if i < 0 {
		return archive.Passphrase{}, false
	}
	return v.ring.passphrases[i].pass, true
}

// Generations is every generation in the file, newest first, for a restore
// that tries each when the Admin typed none (a Settings restore).
func (v *Vault) Generations() []Generation {
	v.mu.Lock()
	defer v.mu.Unlock()
	gens := make([]Generation, len(v.ring.passphrases))
	for i, p := range v.ring.passphrases {
		gens[len(gens)-1-i] = p.gen
	}
	return gens
}

// Replace adds a passphrase generation and a new fence secret. Nothing is
// re-encrypted: older Backups keep opening with their own generation.
func (v *Vault) Replace(rand io.Reader) (Generation, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	gen := Generation(1)
	if n := len(v.ring.passphrases); n > 0 {
		gen = v.ring.passphrases[n-1].gen + 1
	}
	next := keyring{
		passphrases: append(slices.Clone(v.ring.passphrases), generation{gen: gen, pass: archive.NewPassphrase(rand)}),
		fence:       newFence(rand, v.ring.fence.gen+1),
	}
	if err := v.commit(next); err != nil {
		return 0, err
	}
	return gen, nil
}

// Drop removes each generation no listed Backup uses. The current one
// stays, since the next Backup needs it.
func (v *Vault) Drop(inUse func(Generation) bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	n := len(v.ring.passphrases)
	kept := slices.DeleteFunc(slices.Clone(v.ring.passphrases), func(p generation) bool {
		return p.gen != v.ring.passphrases[n-1].gen && !inUse(p.gen)
	})
	if len(kept) == n {
		return nil
	}
	return v.commit(keyring{passphrases: kept, fence: v.ring.fence})
}

// RegenerateFence replaces the fence secret at the next generation, so no
// proof minted before it is proven.
func (v *Vault) RegenerateFence(rand io.Reader) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.commit(keyring{passphrases: v.ring.passphrases, fence: newFence(rand, v.ring.fence.gen+1)})
}

var errNoPassphrase = errors.New("backup.key has no passphrase yet")

// Bytes is the file as the archive carries it.
func (v *Vault) Bytes() ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.raw == nil {
		return nil, errNoPassphrase
	}
	return slices.Clone(v.raw), nil
}

// Fence is the fence secret's current generation and key.
type Fence struct {
	Generation uint32
	key        [32]byte
}

func (v *Vault) Fence() Fence {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.ring.fence.derive()
}

func (f fenceSecret) derive() Fence {
	key, err := hkdf.Key(sha256.New, f.secret[:], nil, "slipstream fence v1", 32)
	if err != nil {
		panic(fmt.Sprintf("backups: derive the fence key: %v", err))
	}
	out := Fence{Generation: f.gen}
	copy(out.key[:], key)
	return out
}

const proofSize = 4 + sha256.Size

// Prove is the proof that newKey replaced oldKey under f: uint32be(gen) ||
// HMAC-SHA256(key, oldKey || newKey || uint32be(gen)), 36 bytes. oldKey is
// always a 32-byte Node key, so the concatenation is unambiguous.
func Prove(f Fence, oldKey, newKey []byte) []byte {
	out := binary.BigEndian.AppendUint32(make([]byte, 0, proofSize), f.Generation)
	m := hmac.New(sha256.New, f.key[:])
	m.Write(oldKey)
	m.Write(newKey)
	m.Write(out)
	return m.Sum(out)
}

// Proven reports whether proof names oldKey and newKey under this Vault's
// current fence generation. A proof from an earlier generation, a Resume
// or a Replace passphrase ago, is false. Constant time.
func (v *Vault) Proven(oldKey, newKey, proof []byte) bool {
	f := v.Fence()
	return f.Generation != 0 && hmac.Equal(proof, Prove(f, oldKey, newKey))
}

// MergeVault is backup.key.restore for a restore in place: current's
// passphrase generations plus each one of the archive's that current
// lacks, sealed with the staged library.key. It keeps current's fence
// secret and generation, or the archive's when current has no file, so
// every earlier Backup can still fence this Controller.
func MergeVault(current *Vault, staged []byte, keys *secret.Keys) ([]byte, error) {
	sealer := keys.Sealer()
	archived, err := decode(staged, sealer)
	if err != nil {
		return nil, err
	}
	current.mu.Lock()
	ring := keyring{passphrases: slices.Clone(current.ring.passphrases), fence: current.ring.fence}
	current.mu.Unlock()
	if ring.fence.gen == 0 {
		ring.fence = archived.fence
	}
	for _, p := range archived.passphrases {
		if !slices.ContainsFunc(ring.passphrases, func(q generation) bool { return q.gen == p.gen }) {
			ring.passphrases = append(ring.passphrases, p)
		}
	}
	slices.SortFunc(ring.passphrases, func(a, b generation) int { return cmp.Compare(a.gen, b.gen) })
	return encode(ring, sealer)
}

// VaultForNewMachine is the archive's passphrase generations with a fresh
// fence secret at the archive's fence generation plus one, so a leaked
// Backup of the old Controller can't fence the new one.
func VaultForNewMachine(staged []byte, keys *secret.Keys, rand io.Reader) ([]byte, error) {
	sealer := keys.Sealer()
	ring, err := decode(staged, sealer)
	if err != nil {
		return nil, err
	}
	ring.fence = newFence(rand, ring.fence.gen+1)
	return encode(ring, sealer)
}

func newFence(rand io.Reader, gen uint32) fenceSecret {
	f := fenceSecret{gen: gen}
	if _, err := io.ReadFull(rand, f.secret[:]); err != nil {
		panic(fmt.Sprintf("backups: draw a fence secret: %v", err))
	}
	return f
}

// commit writes next to backup.key and only then makes it the Vault's.
func (v *Vault) commit(next keyring) error {
	b, err := encode(next, v.sealer)
	if err != nil {
		return err
	}
	if err := atomicfile.Replace(v.path, b, v.retry); err != nil {
		return fmt.Errorf("write %s: %w", VaultFile, err)
	}
	v.ring, v.raw = next, b
	return nil
}

// vaultFile is backup.key on disk. Each value is sealed with library.key
// and bound to its own generation.
type vaultFile struct {
	Format      int           `json:"format"`
	Passphrases []sealedValue `json:"passphrases"`
	Fence       sealedValue   `json:"fence"`
}

type sealedValue struct {
	Generation uint32 `json:"generation"`
	Sealed     []byte `json:"sealed"`
}

const vaultFormat = 1

var (
	passphraseColumn = secret.NewColumn("backup_key.passphrase_sealed")
	fenceColumn      = secret.NewColumn("backup_key.fence_sealed")
)

func ad(c secret.Column, gen uint32) secret.AD { return c.Row(strconv.FormatUint(uint64(gen), 10)) }

func encode(r keyring, s *secret.Sealer) ([]byte, error) {
	f := vaultFile{Format: vaultFormat, Fence: sealedValue{Generation: r.fence.gen, Sealed: sealedBytes(s.Seal(r.fence.secret[:], ad(fenceColumn, r.fence.gen)))}}
	for _, p := range r.passphrases {
		f.Passphrases = append(f.Passphrases, sealedValue{Generation: uint32(p.gen), Sealed: sealedBytes(s.Seal([]byte(p.pass.Reveal()), ad(passphraseColumn, uint32(p.gen))))})
	}
	b, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", VaultFile, err)
	}
	return b, nil
}

// decode refuses anything encode doesn't write: an unknown field or
// format, no passphrase, generations out of order, and a value that
// doesn't open under keys in its own generation.
func decode(b []byte, s *secret.Sealer) (keyring, error) {
	var f vaultFile
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return keyring{}, fmt.Errorf("read %s: %w", VaultFile, err)
	}
	fail := func(format string, args ...any) (keyring, error) {
		return keyring{}, fmt.Errorf("read %s: %s", VaultFile, fmt.Sprintf(format, args...))
	}
	switch {
	case f.Format != vaultFormat:
		return fail("format %d isn't %d", f.Format, vaultFormat)
	case len(f.Passphrases) == 0:
		return fail("it holds no passphrase")
	case f.Fence.Generation == 0:
		return fail("it holds no fence secret")
	}
	var r keyring
	for i, p := range f.Passphrases {
		if p.Generation == 0 || i > 0 && p.Generation <= f.Passphrases[i-1].Generation {
			return fail("passphrase generation %d is out of order", p.Generation)
		}
		plain, err := s.Open(sealedFrom(p.Sealed), ad(passphraseColumn, p.Generation))
		if err != nil {
			return fail("passphrase generation %d doesn't open with this library.key", p.Generation)
		}
		pass, err := archive.ParsePassphrase(string(plain))
		if err != nil {
			return fail("passphrase generation %d: %v", p.Generation, err)
		}
		r.passphrases = append(r.passphrases, generation{gen: Generation(p.Generation), pass: pass})
	}
	plain, err := s.Open(sealedFrom(f.Fence.Sealed), ad(fenceColumn, f.Fence.Generation))
	if err != nil || len(plain) != len(r.fence.secret) {
		return fail("the fence secret doesn't open with this library.key")
	}
	r.fence.gen = f.Fence.Generation
	copy(r.fence.secret[:], plain)
	return r, nil
}

// sealedBytes and sealedFrom carry a Sealed through its driver.Valuer and
// sql.Scanner, the only way out of and into one.
func sealedBytes(s secret.Sealed) []byte {
	v, _ := s.Value()
	b, _ := v.([]byte)
	return b
}

func sealedFrom(b []byte) secret.Sealed {
	var s secret.Sealed
	_ = s.Scan(b)
	return s
}
