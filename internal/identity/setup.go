package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"log/slog"
	"strings"
	"sync"
)

const (
	crockford      = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	setupCodeLen   = 12
	setupCodeGroup = 4
	setupCodeHint  = "open this Node in a browser to start a new SlipStream, join one or restore a Backup"
)

// newSetupCode draws 60 bits as 12 Crockford base32 symbols, shown as
// XXXX-XXXX-XXXX.
func newSetupCode() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint64(b[:])
	var out strings.Builder
	for i := range setupCodeLen {
		if i > 0 && i%setupCodeGroup == 0 {
			out.WriteByte('-')
		}
		out.WriteByte(crockford[n&31])
		n >>= 5
	}
	return out.String()
}

// normalizeSetupCode reads a typed code the way a person means it: any
// case, spaces and dashes dropped, O as 0, I and L as 1.
func normalizeSetupCode(typed string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-':
			return -1
		case 'o', 'O':
			return '0'
		case 'i', 'I', 'l', 'L':
			return '1'
		}
		if 'a' <= r && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, typed)
}

type gateState uint8

const (
	gateOpen gateState = iota
	gateBusy
	gateClosed
)

// Gate admits setup operations while the Node is Unclaimed: open, busy
// while one runs, and closed for the life of the process once the Node is
// claimed.
type Gate struct {
	claimed func(context.Context) (bool, error)
	log     *slog.Logger

	mu    sync.Mutex
	state gateState
	code  string
}

// NewGate asks claimed once. An Unclaimed Node draws a Setup code, logs it
// and returns it for the readiness record; a claimed one returns "" and a
// closed gate. claimed is the composition root's one definition of claimed.
func NewGate(ctx context.Context, claimed func(context.Context) (bool, error), log *slog.Logger) (*Gate, string, error) {
	g := &Gate{claimed: claimed, log: log}
	done, err := claimed(ctx)
	if err != nil {
		return nil, "", err
	}
	if done {
		g.state = gateClosed
		return g, "", nil
	}
	code := newSetupCode()
	g.code = normalizeSetupCode(code)
	log.Info("setup code", "code", code, "hint", setupCodeHint)
	return g, code, nil
}

// Run admits c with the typed Setup code and runs fn with its Claim. When
// fn returns, the gate closes if the Node is now claimed and reopens
// otherwise. A caller with a wrong X-Api-Key, a wrong code, a busy gate or
// a closed one gets ErrAPIKeyWrong, ErrSetupCodeWrong, ErrSetupBusy or
// ErrAlreadyClaimed.
func (g *Gate) Run(ctx context.Context, c Caller, typed string, fn func(Claim)) error {
	if c.kind == wrongAPIKey {
		return ErrAPIKeyWrong
	}
	if err := g.enter(typed); err != nil {
		return err
	}
	defer g.leave(context.WithoutCancel(ctx))
	fn(Claim{c.client})
	return nil
}

func (g *Gate) enter(typed string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state == gateClosed {
		return ErrAlreadyClaimed
	}
	if subtle.ConstantTimeCompare([]byte(normalizeSetupCode(typed)), []byte(g.code)) != 1 {
		return ErrSetupCodeWrong
	}
	if g.state == gateBusy {
		return ErrSetupBusy
	}
	g.state = gateBusy
	return nil
}

func (g *Gate) leave(ctx context.Context) {
	done, err := g.claimed(ctx)
	if err != nil {
		g.log.Error("setup gate cannot tell whether the Node is claimed", "error", err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if done {
		g.state = gateClosed
		g.code = ""
		g.log.Info("setup code used")
		return
	}
	g.state = gateOpen
}
