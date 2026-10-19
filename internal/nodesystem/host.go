package nodesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/system"
)

// Host is what one Node offers about itself. An error that wraps
// ErrOffline means the Node is starting or stopping.
type Host interface {
	Facts() (system.Facts, error)
	Config() (Config, error)
	// SetConfig refuses with a *Refusal and changes nothing, or applies
	// every change.
	SetConfig(changes []Change) (Config, error)
	// Restart stops the Node to start again, or answers
	// ErrAlreadyRestarting.
	Restart() (Restarting, error)
	ReadLogs(ctx context.Context, after uint64, wait time.Duration) (system.LogBatch, error)
	Archive() (Archive, error)
	// Reach dials t from the Node until ctx ends.
	Reach(ctx context.Context, t Target) Reached
	// Probe runs every probe now, reports the verdicts in the Node's facts
	// and answers them.
	Probe() ([]role.ProbeResult, error)
}

// Target is a host and port a Role reaches from its Node, over TLS when
// TLS is set.
type Target struct {
	Host string
	Port uint16
	TLS  bool
}

func (t Target) String() string { return net.JoinHostPort(t.Host, strconv.Itoa(int(t.Port))) }

// Reached is how a reach test went: Err says why it failed, "" when OK.
type Reached struct {
	OK   bool
	Took time.Duration
	Err  string
}

var (
	ErrNotFound          = errors.New("no Node has this id")
	ErrOffline           = errors.New("the Node is offline")
	ErrMismatched        = errors.New("the Node runs another version")
	ErrAlreadyRestarting = errors.New("the Node is already restarting")
)

// SectionID is the Nodes section a setting shows in.
type SectionID string

const (
	Web SectionID = "web"
	Log SectionID = "log"
)

// Config is a Node's settings as the Nodes sections show them: each field
// in the JSON its setting.Field marshals to.
type Config struct {
	Sections []Section
}

type Section struct {
	ID     SectionID
	Fields []json.RawMessage
}

// Revision is a digest of the whole view, saved and running values and
// sources alike: a new one means a surface showing the view should
// refetch it.
func (c Config) Revision() string {
	b, err := json.Marshal(c)
	if err != nil {
		panic("nodesystem: encode a config: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

// Restarting is a restart the Node accepted. NextPort is http.listen's
// saved port while it waits for this restart, else 0.
type Restarting struct {
	StartedAt time.Time
	BoundPort uint16
	NextPort  uint16
}

// Archive is a zip of a Node's log folder at offset 0, stamped
// YYYYMMDD-HHMMSS in the Node's time zone. Its reader deletes it.
type Archive struct {
	File  *os.File
	Size  int64
	Stamp string
}
