package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

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

// ErrNotFound is a connection id no connection has.
var ErrNotFound = errors.New("notifications: no connection has this id")

// FieldError refuses one field of a connection, for the sheet to mark.
type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Reason }

// Draft is a new connection, or a Test of one, as the Admin typed it.
type Draft struct {
	Name     string
	Provider providers.ID
	On       bool
	// Events nil is the recommended events.
	Events   []notice.Kind
	Warnings bool
	Posters  bool
	Node     role.NodeID
	Settings providers.Values
	Secrets  map[string]string
}

// Edit changes a saved connection. A nil field keeps the saved value, and
// Settings are laid over the saved ones key by key. A secret absent or ""
// keeps the saved value, and nil removes an optional one.
type Edit struct {
	Name     *string
	On       *bool
	Events   *[]notice.Kind
	Warnings *bool
	Posters  *bool
	Node     *role.NodeID
	Settings providers.Values
	Secrets  map[string]*string
}

// empty is whether e changes nothing, so a Test of it tests the saved
// values.
func (e Edit) empty() bool {
	for _, v := range e.Secrets {
		if v == nil || *v != "" {
			return false
		}
	}
	return e.Name == nil && e.On == nil && e.Events == nil && e.Warnings == nil && e.Posters == nil && e.Node == nil && len(e.Settings) == 0
}

// draft is d as a connection not yet stored, with its secrets.
func (s *Service) draft(d Draft, now time.Time) (Connection, providers.Values, error) {
	c := Connection{
		ID: ConnectionID{UUID: storage.NewUUID()}, Name: d.Name, Provider: d.Provider, On: d.On, OnAt: now, Events: d.Events,
		Warnings: d.Warnings, Posters: d.Posters, Node: d.Node, Revision: 1, Created: now,
	}
	if c.Events == nil {
		c.Events = s.recommended()
	}
	c.Settings = maps.Clone(d.Settings)
	if c.Settings == nil {
		c.Settings = providers.Values{}
	}
	secrets := providers.Values{}
	for k, v := range d.Secrets {
		if v == "" {
			continue
		}
		raw, err := s.secret(c.Provider, k, v)
		if err != nil {
			return c, secrets, err
		}
		secrets[k] = raw
	}
	if err := s.check(c, secrets); err != nil {
		return c, secrets, err
	}
	for _, f := range s.provider(c.Provider).Fields {
		if _, ok := c.Settings[f.Key]; !ok && f.Default != nil {
			c.Settings[f.Key] = f.Default
		}
	}
	return c, secrets, nil
}

// edited is c with e laid over it, and its secrets so.
func (s *Service) edited(c Connection, secrets providers.Values, e Edit, now time.Time) (Connection, providers.Values, error) {
	if e.Name != nil {
		c.Name = *e.Name
	}
	if e.On != nil {
		if *e.On && !c.On {
			c.OnAt = now
		}
		c.On = *e.On
	}
	if e.Events != nil {
		c.Events = *e.Events
	}
	if e.Warnings != nil {
		c.Warnings = *e.Warnings
	}
	if e.Posters != nil {
		c.Posters = *e.Posters
	}
	if e.Node != nil {
		c.Node = *e.Node
	}
	settings := maps.Clone(c.Settings)
	maps.Copy(settings, e.Settings)
	c.Settings = settings
	secrets = maps.Clone(secrets)
	for k, v := range e.Secrets {
		switch {
		case v == nil:
			if f, ok := s.field(c.Provider, k); ok && !f.Optional {
				return c, nil, &FieldError{Field: k, Reason: f.Label + " can't be removed. Enter a new one instead."}
			}
			delete(secrets, k)
		case *v != "":
			raw, err := s.secret(c.Provider, k, *v)
			if err != nil {
				return c, nil, err
			}
			secrets[k] = raw
		}
	}
	return c, secrets, s.check(c, secrets)
}

func (s *Service) recommended() []notice.Kind {
	var out []notice.Kind
	for _, spec := range s.Events() {
		if spec.OnByDefault && !spec.Fixed {
			out = append(out, spec.Kind)
		}
	}
	return out
}

func (s *Service) field(id providers.ID, key string) (providers.Field, bool) {
	i := slices.IndexFunc(s.cfg.Providers, func(p providers.Provider) bool { return p.ID == id })
	if i < 0 {
		return providers.Field{}, false
	}
	fields := s.cfg.Providers[i].Fields
	j := slices.IndexFunc(fields, func(f providers.Field) bool { return f.Key == key })
	if j < 0 {
		return providers.Field{}, false
	}
	return fields[j], true
}

// scriptProvider is Custom Script's id, the one provider whose connections
// name a Node.
const scriptProvider providers.ID = "script"

// check refuses a connection its provider's fields don't allow.
func (s *Service) check(c Connection, secrets providers.Values) error {
	if !slices.ContainsFunc(s.cfg.Providers, func(p providers.Provider) bool { return p.ID == c.Provider }) {
		return &FieldError{Field: "provider", Reason: "Pick one of the providers in the list."}
	}
	if n := utf8.RuneCountInString(c.Name); n < 1 || n > 100 {
		return &FieldError{Field: "name", Reason: "Use 1 to 100 characters."}
	}
	if (c.Provider == scriptProvider) != c.script() {
		return &FieldError{Field: "node", Reason: "Pick the Node that runs this script."}
	}
	if c.script() && c.Node != s.cfg.Roster.Self() && !s.cfg.Roster.Holds(role.Notifying, c.Node) {
		return &FieldError{Field: "node", Reason: s.names(c.Node) + " doesn't have the Notifying Role. Pick a Node that has it."}
	}
	if err := s.checkEvents(c.Events); err != nil {
		return &FieldError{Field: "events", Reason: "Pick events from the list."}
	}
	return s.checkValues(c, secrets)
}

// checkValues refuses a setting or secret the provider doesn't have, and a
// required one left out.
func (s *Service) checkValues(c Connection, secrets providers.Values) error {
	for _, values := range []struct {
		v      providers.Values
		secret bool
	}{{c.Settings, false}, {secrets, true}} {
		for k := range values.v {
			if f, ok := s.field(c.Provider, k); !ok || f.Secret != values.secret {
				return &FieldError{Field: k, Reason: s.provider(c.Provider).Label + " has no setting with this name."}
			}
		}
	}
	for _, f := range s.provider(c.Provider).Fields {
		if f.Optional || f.Default != nil || f.Kind == providers.NodeField {
			continue
		}
		values := c.Settings
		if f.Secret {
			values = secrets
		}
		if v, ok := values[f.Key]; !ok || string(v) == `""` || string(v) == "null" {
			return &FieldError{Field: f.Key, Reason: "Enter " + f.Label + "."}
		}
	}
	return nil
}

// secret is the text an Admin typed for the secret key as its field's
// value: a list one entry a line, headers one "Name: value" a line, and
// any other kind the text itself. A key the provider lacks keeps the text,
// for check to refuse.
func (s *Service) secret(id providers.ID, key, text string) (json.RawMessage, error) {
	f, _ := s.field(id, key)
	switch f.Kind {
	case providers.List:
		list := []string{}
		for line := range strings.Lines(text) {
			if line = strings.TrimSpace(line); line != "" {
				list = append(list, line)
			}
		}
		return jsonValue(list), nil
	case providers.Headers:
		headers := []providers.Header{}
		for line := range strings.Lines(text) {
			if line = strings.TrimSpace(line); line == "" {
				continue
			}
			name, value, ok := strings.Cut(line, ":")
			if name = strings.TrimSpace(name); !ok || name == "" {
				return nil, &FieldError{Field: key, Reason: "Write each header on its own line as Name: value."}
			}
			headers = append(headers, providers.Header{Name: name, Value: strings.TrimSpace(value)})
		}
		return jsonValue(headers), nil
	case providers.Text, providers.URL, providers.Number, providers.Select, providers.Switch, providers.NodeField, providers.Duration:
	}
	return jsonValue(text), nil
}

func jsonValue(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("notifications: encode a secret: %v", err))
	}
	return b
}

func (c Connection) params() (db.InsertConnectionParams, error) {
	events, err := json.Marshal(slices.Sorted(slices.Values(c.Events)))
	if err != nil {
		return db.InsertConnectionParams{}, fmt.Errorf("encode a connection's events: %w", err)
	}
	settings, err := json.Marshal(c.Settings)
	if err != nil {
		return db.InsertConnectionParams{}, fmt.Errorf("encode a connection's settings: %w", err)
	}
	return db.InsertConnectionParams{
		ID: c.ID, Name: c.Name, Provider: string(c.Provider), IsOn: c.On, OnAt: storage.FromTime(c.OnAt), Events: string(events),
		Warnings: c.Warnings, Posters: c.Posters, NodeID: c.Node, Settings: string(settings), SecretsSealed: c.Sealed,
		CreatedAt: storage.FromTime(c.Created),
	}, nil
}

// nameTaken refuses a name another connection has.
func nameTaken(ctx context.Context, q *db.Queries, c Connection) error {
	conns, err := q.ListConnections(ctx)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(conns, func(o db.NotificationsConnection) bool { return o.Name == c.Name && o.ID != c.ID }) {
		return &FieldError{Field: "name", Reason: "Another connection already uses this name."}
	}
	return nil
}

func (s *Service) Connections(ctx context.Context) ([]View, error) {
	views, _, err := s.statuses(ctx)
	return views, err
}

func (s *Service) Connection(ctx context.Context, id ConnectionID) (View, error) {
	views, err := s.Connections(ctx)
	if err != nil {
		return View{}, err
	}
	i := slices.IndexFunc(views, func(v View) bool { return v.ID == id })
	if i < 0 {
		return View{}, ErrNotFound
	}
	return views[i], nil
}

// Create stores d as a new connection and wakes the dispatcher.
func (s *Service) Create(ctx context.Context, d Draft) (View, error) {
	c, secrets, err := s.draft(d, time.Now())
	if err != nil {
		return View{}, err
	}
	if c.Sealed, err = s.seal(c.ID, secrets); err != nil {
		return View{}, err
	}
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := nameTaken(ctx, q, c); err != nil {
			return err
		}
		p, err := c.params()
		if err != nil {
			return err
		}
		return q.InsertConnection(ctx, p)
	})
	if err != nil {
		return View{}, err
	}
	s.changed(ctx)
	return s.Connection(ctx, c.ID)
}

// Edit lays e over connection id. Any edit moves its revision, which starts
// its line afresh and so unpauses it.
func (s *Service) Edit(ctx context.Context, id ConnectionID, e Edit) (View, error) {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		c, secrets, err := s.saved(ctx, q, id)
		if err != nil {
			return err
		}
		if c, secrets, err = s.edited(c, secrets, e, time.Now()); err != nil {
			return err
		}
		if c.Sealed, err = s.seal(c.ID, secrets); err != nil {
			return err
		}
		if err := nameTaken(ctx, q, c); err != nil {
			return err
		}
		p, err := c.params()
		if err != nil {
			return err
		}
		return q.UpdateConnection(ctx, db.UpdateConnectionParams{
			ID: id, Name: p.Name, IsOn: p.IsOn, OnAt: p.OnAt, Events: p.Events, Warnings: p.Warnings, Posters: p.Posters,
			NodeID: p.NodeID, Settings: p.Settings, SecretsSealed: p.SecretsSealed,
		})
	})
	if err != nil {
		return View{}, err
	}
	s.changed(ctx)
	return s.Connection(ctx, id)
}

// Delete removes connection id with its waiting messages and its Health
// pairing.
func (s *Service) Delete(ctx context.Context, id ConnectionID) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if _, err := q.GetConnection(ctx, id); errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if err := q.DeleteToldOf(ctx, id.String()); err != nil {
			return err
		}
		return q.DeleteConnection(ctx, id)
	})
	if err != nil {
		return err
	}
	s.changed(ctx)
	return nil
}

// saved is connection id with its opened secrets.
func (s *Service) saved(ctx context.Context, q *db.Queries, id ConnectionID) (Connection, providers.Values, error) {
	row, err := q.GetConnection(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Connection{}, nil, ErrNotFound
	}
	if err != nil {
		return Connection{}, nil, err
	}
	c, err := connectionOf(row)
	if err != nil {
		return Connection{}, nil, err
	}
	secrets, err := s.open(c.ID, c.Sealed)
	return c, secrets, err
}

// changed wakes the dispatcher and tells screens after a connection
// changed.
func (s *Service) changed(ctx context.Context) {
	s.ring()
	s.cfg.Changed.Publish(ctx, Changed{Connections: true})
}
