package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type (
	ConnectionID = db.ConnectionID
	EventID      = db.EventID
	Seq          = db.Seq
)

// Connection is a notification connection's config. Its secrets stay
// sealed until a send opens them.
type Connection struct {
	ID       ConnectionID
	Name     string
	Provider providers.ID
	On       bool
	// OnAt is when it was last turned on: it gets no event from before.
	OnAt time.Time
	// Events are the kinds it sends, sorted.
	Events   []notice.Kind
	Warnings bool
	Posters  bool
	// Node runs a Custom Script, and is zero for every other provider.
	Node     role.NodeID
	Settings providers.Values
	Sealed   secret.Sealed
	// Revision moves on every edit and on a passing Test of the saved
	// values.
	Revision int64
	Created  time.Time
}

func (c Connection) script() bool { return !c.Node.IsZero() }

func (c Connection) wants(k notice.Kind) bool { return slices.Contains(c.Events, k) }

func connectionOf(row db.NotificationsConnection) (Connection, error) {
	c := Connection{
		ID: row.ID, Name: row.Name, Provider: providers.ID(row.Provider), On: row.IsOn, OnAt: row.OnAt.Time(),
		Warnings: row.Warnings, Posters: row.Posters, Node: row.NodeID, Sealed: row.SecretsSealed,
		Revision: row.Revision, Created: row.CreatedAt.Time(),
	}
	if err := json.Unmarshal([]byte(row.Events), &c.Events); err != nil {
		return Connection{}, fmt.Errorf("decode the events of connection %s: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Settings), &c.Settings); err != nil {
		return Connection{}, fmt.Errorf("decode the settings of connection %s: %w", row.ID, err)
	}
	return c, nil
}

func connectionsIn(ctx context.Context, r storage.Reader[file]) ([]Connection, error) {
	rows, err := db.New(r).ListConnections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Connection, len(rows))
	for i, row := range rows {
		if out[i], err = connectionOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// linesIn is every stored line by connection.
func linesIn(ctx context.Context, r storage.Reader[file]) (map[ConnectionID]Line, error) {
	rows, err := db.New(r).ListLines(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ConnectionID]Line, len(rows))
	for _, row := range rows {
		if out[row.ConnectionID], err = lineOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lineFor is c's line: the stored one, or ok at c's revision.
func lineFor(lines map[ConnectionID]Line, c Connection) Line {
	if l, ok := lines[c.ID]; ok {
		return l
	}
	return Line{State: OK, Revision: c.Revision}
}

func (s *Service) provider(id providers.ID) providers.Provider {
	i := slices.IndexFunc(s.cfg.Providers, func(p providers.Provider) bool { return p.ID == id })
	if i < 0 {
		panic(fmt.Sprintf("notifications: a connection names provider %s, which isn't in the table", id))
	}
	return s.cfg.Providers[i]
}

// label is c in copy: "Discord 'Family'".
func (s *Service) label(c Connection) string {
	return fmt.Sprintf("%s '%s'", s.provider(c.Provider).Label, c.Name)
}

// nodeView is what statusOf reads of c's Node now.
func (s *Service) nodeView(c Connection) NodeView {
	if !c.script() {
		return NodeView{}
	}
	return NodeView{
		ID: c.Node, Name: s.cfg.Roster.Name(c.Node), InStep: s.cfg.Roster.InStep(c.Node),
		Removed: s.cfg.Roster.WrittenOff(role.Notifying, c.Node),
	}
}

var secretsColumn = secret.NewColumn("notifications_connection.secrets_sealed")

func (s *Service) seal(id ConnectionID, v providers.Values) (secret.Sealed, error) {
	if len(v) == 0 {
		return secret.Sealed{}, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return secret.Sealed{}, fmt.Errorf("encode the secrets: %w", err)
	}
	return s.cfg.Sealer.Seal(b, secretsColumn.Row(id.String())), nil
}

func (s *Service) open(id ConnectionID, sealed secret.Sealed) (providers.Values, error) {
	if !sealed.Set() {
		return providers.Values{}, nil
	}
	b, err := s.cfg.Sealer.Open(sealed, secretsColumn.Row(id.String()))
	if err != nil {
		return nil, fmt.Errorf("open the secrets of connection %s: %w", id, err)
	}
	var v providers.Values
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("decode the secrets of connection %s: %w", id, err)
	}
	return v, nil
}

// names is a Node's name now, for reasons.
func (s *Service) names(node role.NodeID) string {
	if name := s.cfg.Roster.Name(node); name != "" {
		return name
	}
	return "A removed Node"
}

// View is a connection with what its row says now. It holds no secret
// value.
type View struct {
	Connection
	Status Status
}

// statuses reads every connection's status, and Warn when a connection
// has failed for.
func (s *Service) statuses(ctx context.Context) ([]View, time.Duration, error) {
	var (
		conns  []Connection
		lines  map[ConnectionID]Line
		counts []db.CountDueRow
		warn   time.Duration
	)
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if conns, err = connectionsIn(ctx, r); err != nil {
			return err
		}
		if lines, err = linesIn(ctx, r); err != nil {
			return err
		}
		q := db.New(r)
		if counts, err = q.CountDue(ctx); err != nil {
			return err
		}
		settings, err := q.GetSettings(ctx)
		warn = WarnAfter.Value(settings.WarnSeconds)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	due := map[ConnectionID]int{}
	for _, c := range counts {
		due[c.ConnectionID] = int(c.Due)
	}
	now := time.Now()
	out := make([]View, len(conns))
	for i, c := range conns {
		out[i] = View{Connection: c, Status: statusOf(c, lineFor(lines, c), s.nodeView(c), due[c.ID], now, warn)}
	}
	return out, warn, nil
}

// wording words reasons as of now.
func (s *Service) wording(now time.Time, giveUp time.Duration) wording {
	return wording{names: s.names, now: now, zone: s.cfg.Zone, giveUp: giveUp}
}
