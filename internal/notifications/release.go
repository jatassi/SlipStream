package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/storage"
)

// releaseBound is how many events one pass releases, so a backlog is many
// short Writes.
const releaseBound = 200

// bellTarget is the bell's name among told targets; a connection's is its
// id.
const bellTarget = "bell"

type event struct {
	ID      EventID
	Kind    notice.Kind
	At      time.Time
	Subject []byte
	// Items are a Health event's items.
	Items []item
}

// cut is a Health event with only the items keys names, as its subject.
// Other kinds keep their subject.
func (e event) cut(keys []string) (event, error) {
	if !isHealth(e.Kind) {
		return e, nil
	}
	e.Items = slices.DeleteFunc(slices.Clone(e.Items), func(it item) bool { return !slices.Contains(keys, it.key()) })
	subject, err := json.Marshal(healthFacts{Items: e.Items})
	if err != nil {
		return event{}, fmt.Errorf("encode the Health items of event %d: %w", e.ID, err)
	}
	e.Subject = subject
	return e, nil
}

func isHealth(k notice.Kind) bool { return k == newItemsRow.Kind || k == clearedRow.Kind }

// itemKey is a Health item's key among notifications: "<category>.<key>".
// Category ids hold no ".".
func itemKey(c health.ID, k health.Key) string { return string(c) + "." + string(k) }

func (it item) key() string { return string(it.Category) + "." + it.Key }

func itemOf(row db.NotificationsItem) item {
	category, key, _ := strings.Cut(row.Key, ".")
	return item{
		Category: health.ID(category), Key: key, Level: row.Level, Subject: row.Subject, Text: row.Text,
		Since: row.SinceAt.Time(), Cleared: row.ClearedAt.Time(),
	}
}

func (it item) failing() bool { return it.Level == health.Fail.String() }

// bellView is what a release reads of the bell for one kind.
type bellView struct {
	On       bool
	Warnings bool
}

// planned is one delivery a release makes.
type planned struct {
	Conn ConnectionID
	// Items are the Health keys it carries, nil for other kinds.
	Items []string
}

// released is what one event's release writes.
type released struct {
	Deliveries []planned
	Show       bool
	// Shown are the Health keys the bell row shows, nil for other kinds.
	Shown []string
	// Tell are the bell's told rows to upsert; Untell the keys to delete.
	Tell   []item
	Untell []string
}

// targetsOf is every filter of a release in one place. A connection gets
// an event when it is on, it was on before the event, and it sends the
// kind. A New Health item goes with the items that pass the connection's
// warnings switch and aren't about the connection itself. A Health item
// cleared ignores the warnings switch and goes with the items the
// connection was told, or has a due delivery carrying. The bell shows the
// kinds it is on for, New Health items through its own warnings switch,
// and clears of what it showed. An empty subset gets nothing.
func targetsOf(e event, conns []Connection, bell bellView, told map[string]map[string]bool, carrying map[ConnectionID]map[string]bool) released {
	var out released
	for _, c := range conns {
		if d, ok := deliveryFor(e, c, told, carrying); ok {
			out.Deliveries = append(out.Deliveries, d)
		}
	}
	if bell.On {
		shownOn(&out, e, bell, told)
	}
	return out
}

func deliveryFor(e event, c Connection, told map[string]map[string]bool, carrying map[ConnectionID]map[string]bool) (planned, bool) {
	if !c.On || !c.wants(e.Kind) {
		return planned{}, false
	}
	var keys []string
	switch e.Kind {
	case newItemsRow.Kind:
		if c.OnAt.After(e.At) {
			return planned{}, false
		}
		own := itemKey(HealthAbout.ID, health.KeyOf(c.ID.String()))
		keys = keysOf(e.Items, func(it item) bool { return (it.failing() || c.Warnings) && it.key() != own })
	case clearedRow.Kind:
		id := c.ID.String()
		keys = keysOf(e.Items, func(it item) bool { return told[id][it.key()] || carrying[c.ID][it.key()] })
	default:
		return planned{Conn: c.ID}, !c.OnAt.After(e.At)
	}
	return planned{Conn: c.ID, Items: keys}, len(keys) > 0
}

func shownOn(out *released, e event, bell bellView, told map[string]map[string]bool) {
	switch e.Kind {
	case newItemsRow.Kind:
		for _, it := range e.Items {
			if it.failing() || bell.Warnings {
				out.Tell = append(out.Tell, it)
				out.Shown = append(out.Shown, it.key())
			}
		}
		out.Show = len(out.Shown) > 0
	case clearedRow.Kind:
		out.Untell = keysOf(e.Items, func(it item) bool { return told[bellTarget][it.key()] })
		out.Shown, out.Show = out.Untell, len(out.Untell) > 0
	default:
		out.Show = true
	}
}

func keysOf(items []item, keep func(item) bool) []string {
	var out []string
	for _, it := range items {
		if keep(it) {
			out = append(out, it.key())
		}
	}
	return out
}

// narrow is a cleared delivery's items cut to the keys its connection was
// told, at claim.
func narrow(items []string, told map[string]bool) []string {
	var out []string
	for _, k := range items {
		if told[k] {
			out = append(out, k)
		}
	}
	return out
}

// releaseDue releases the events due by now, at most releaseBound. hit is
// whether the bound stopped it.
func (s *Service) releaseDue(ctx context.Context, q *db.Queries, now time.Time, conns []Connection) (pub published, hit bool, err error) {
	due, err := q.ListDueUnreleased(ctx, db.ListDueUnreleasedParams{Now: storage.FromTime(now), Limit: releaseBound})
	if err != nil {
		return pub, false, err
	}
	bell, err := s.bellIn(ctx, q)
	if err != nil {
		return pub, false, err
	}
	for _, row := range due {
		e, err := eventOf(ctx, q, row)
		if err != nil {
			return pub, false, err
		}
		told, carrying, err := pairing(ctx, q, e)
		if err != nil {
			return pub, false, err
		}
		r := targetsOf(e, conns, bellView{On: bell.has(e.Kind, s.catalog), Warnings: bell.warnings}, told, carrying)
		if err := s.write(ctx, q, e, r, now); err != nil {
			return pub, false, err
		}
		pub.feed = pub.feed || r.Show
		pub.connections = pub.connections || len(r.Deliveries) > 0
	}
	return pub, len(due) == releaseBound, nil
}

func eventOf(ctx context.Context, q *db.Queries, row db.NotificationsEvent) (event, error) {
	e := event{ID: row.ID, Kind: row.Kind, At: row.At.Time(), Subject: []byte(row.Subject)}
	if !isHealth(e.Kind) {
		return e, nil
	}
	rows, err := q.ListItems(ctx, e.ID)
	if err != nil {
		return event{}, err
	}
	for _, r := range rows {
		e.Items = append(e.Items, itemOf(r))
	}
	return e, nil
}

// pairing is, for a cleared event's keys, which targets were told each
// and which connections have a due delivery carrying it.
func pairing(ctx context.Context, q *db.Queries, e event) (told map[string]map[string]bool, carrying map[ConnectionID]map[string]bool, err error) {
	if e.Kind != clearedRow.Kind {
		return nil, nil, nil
	}
	keys := keysOf(e.Items, func(item) bool { return true })
	toldRows, err := q.ListToldFor(ctx, keys)
	if err != nil {
		return nil, nil, err
	}
	told = map[string]map[string]bool{}
	for _, t := range toldRows {
		set(told, t.Target, t.Key)
	}
	rows, err := q.ConnectionsCarrying(ctx, keys)
	if err != nil {
		return nil, nil, err
	}
	carrying = map[ConnectionID]map[string]bool{}
	for _, r := range rows {
		set(carrying, r.ConnectionID, r.Key)
	}
	return told, carrying, nil
}

func set[K comparable](m map[K]map[string]bool, k K, v string) {
	if m[k] == nil {
		m[k] = map[string]bool{}
	}
	m[k][v] = true
}

func (s *Service) write(ctx context.Context, q *db.Queries, e event, r released, now time.Time) error {
	for _, d := range r.Deliveries {
		if err := q.InsertDelivery(ctx, db.InsertDeliveryParams{EventID: e.ID, ConnectionID: d.Conn, Items: keysText(d.Items, e.Kind)}); err != nil {
			return fmt.Errorf("release event %d: %w", e.ID, err)
		}
	}
	if r.Show {
		if _, err := q.InsertShown(ctx, db.InsertShownParams{EventID: e.ID, ShownAt: storage.FromTime(now), Items: keysText(r.Shown, e.Kind)}); err != nil {
			return fmt.Errorf("show event %d: %w", e.ID, err)
		}
	}
	for _, it := range r.Tell {
		if err := q.UpsertTold(ctx, toldParams(bellTarget, it, now)); err != nil {
			return err
		}
	}
	for _, k := range r.Untell {
		if err := q.DeleteTold(ctx, db.DeleteToldParams{Target: bellTarget, Key: k}); err != nil {
			return err
		}
	}
	return q.SetReleased(ctx, db.SetReleasedParams{ID: e.ID, ReleasedAt: storage.FromTime(now)})
}

func toldParams(target string, it item, now time.Time) db.UpsertToldParams {
	return db.UpsertToldParams{Target: target, Key: it.key(), Subject: it.Subject, SinceAt: storage.FromTime(it.Since), ToldAt: storage.FromTime(now)}
}

// keysText is keys as stored: NULL for a kind that carries no Health
// items.
func keysText(keys []string, k notice.Kind) *string {
	if !isHealth(k) {
		return nil
	}
	return jsonText(slices.Clip(keys))
}

func keysFrom(text *string) ([]string, error) {
	if text == nil {
		return nil, nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(*text), &keys); err != nil {
		return nil, fmt.Errorf("decode Health keys: %w", err)
	}
	return keys, nil
}
