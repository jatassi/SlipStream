package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/storage"
)

// The bounds of one pass's bulk steps. A pass that reaches one runs again
// at once, so a backlog is many short Writes and never one long one.
const (
	giveUpBound = 1000
	dropBound   = 1000
	// claimBound is how many cleared messages narrowed to nothing one
	// connection may skip in one pass.
	claimBound = 50
)

// published is what a pass changed that screens read.
type published struct{ feed, connections bool }

// job is one message to send, outside any Write.
type job struct {
	conn Connection
	// event is the event with its Health items cut to the delivery's.
	event    event
	items    []string
	accepted []string
}

// result is what a job's send answered, kept in memory until a pass that
// settles it commits.
type result struct {
	conn     ConnectionID
	event    EventID
	kind     notice.Kind
	revision int64
	verdict  providers.Verdict
}

// Run is the dispatcher: one pass after each doorbell, roster change, job
// result or due time. A pass is one Write; sends run outside it, at most
// one per connection. A result leaves memory only with the commit that
// settles it, so a failed pass sends nothing twice. It returns when ctx
// ends.
func (s *Service) Run(ctx context.Context) error {
	s.catalog.seal()
	results := make(chan result)
	busy := map[ConnectionID]bool{}
	var unsettled []result
	failed := 0
	for {
		wake := s.cfg.Roster.Changed()
		next, claimed, pub, err := s.pass(ctx, unsettled, busy)
		switch {
		case ctx.Err() != nil:
			return nil
		case err != nil:
			s.cfg.Log.Error("run the notifications pass", "err", err.Error())
			next = time.Now().Add(s.cfg.Timings.Ladder[min(failed, len(s.cfg.Timings.Ladder)-1)])
			failed++
		default:
			failed = 0
			for _, r := range unsettled {
				delete(busy, r.conn)
			}
			unsettled = unsettled[:0]
			s.publish(ctx, pub)
			for _, j := range claimed {
				busy[j.conn.ID] = true
				go func() {
					r := s.send(ctx, j)
					select {
					case results <- r:
					case <-ctx.Done():
					}
				}()
			}
		}
		timer := time.NewTimer(time.Until(next))
		if next.IsZero() {
			timer.Stop()
		}
		select {
		case <-ctx.Done():
			return nil
		case r := <-results:
			unsettled = append(unsettled, r)
			for more := true; more; {
				select {
				case r := <-results:
					unsettled = append(unsettled, r)
				default:
					more = false
				}
			}
		case <-s.doorbell:
		case <-wake:
		case <-timer.C:
		}
		timer.Stop()
	}
}

func (s *Service) publish(ctx context.Context, p published) {
	if p.feed || p.connections {
		s.cfg.Changed.Publish(ctx, Changed{Feed: p.feed, Connections: p.connections})
	}
}

// pass is one Write: settle what came back, release due events, give up
// old messages, drop those of connections and kinds turned off, start
// lines whose connection changed afresh, and claim each idle connection's
// oldest message. On an error nothing it did is kept.
func (s *Service) pass(ctx context.Context, unsettled []result, busy map[ConnectionID]bool) (time.Time, []job, published, error) {
	if s.cfg.BeforePass != nil {
		if err := s.cfg.BeforePass(ctx); err != nil {
			return time.Time{}, nil, published{}, err
		}
	}
	var p *passing
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if p, err = s.begin(ctx, tx); err != nil {
			return err
		}
		for _, step := range []func(context.Context) error{
			func(ctx context.Context) error { return p.settle(ctx, unsettled) },
			p.release, p.giveUp, p.drop, p.refresh,
			func(ctx context.Context) error { return p.claim(ctx, busy) },
			func(ctx context.Context) error { return p.nextAt(ctx, busy) },
		} {
			if err := step(ctx); err != nil {
				return err
			}
		}
		return p.save(ctx)
	})
	if err != nil {
		return time.Time{}, nil, published{}, err
	}
	return p.next, p.claimed, p.pub, nil
}

// passing is one pass's state inside its Write.
type passing struct {
	s        *Service
	q        *db.Queries
	now      time.Time
	conns    []Connection
	byID     map[ConnectionID]Connection
	lines    map[ConnectionID]Line
	dirty    map[ConnectionID]bool
	giveUpAfter time.Duration
	warn     time.Duration
	hit      bool
	pub      published
	claimed  []job
	next     time.Time
}

func (s *Service) begin(ctx context.Context, tx *storage.Tx[file]) (*passing, error) {
	conns, err := connectionsIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	lines, err := linesIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	q := db.New(tx)
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	p := &passing{
		s: s, q: q, now: time.Now(), conns: conns, byID: map[ConnectionID]Connection{}, lines: lines, dirty: map[ConnectionID]bool{},
		giveUpAfter: GiveUp.Value(settings.GiveUpSeconds), warn: WarnAfter.Value(settings.WarnSeconds),
	}
	for _, c := range conns {
		p.byID[c.ID] = c
	}
	return p, nil
}

func (p *passing) line(c Connection) Line { return lineFor(p.lines, c) }

func (p *passing) setLine(id ConnectionID, l Line) {
	p.lines[id], p.dirty[id] = l, true
}

func (p *passing) settle(ctx context.Context, unsettled []result) error {
	for _, r := range unsettled {
		c, ok := p.byID[r.conn]
		if !ok {
			continue
		}
		row, err := p.q.GetDelivery(ctx, db.GetDeliveryParams{EventID: r.event, ConnectionID: r.conn})
		due := err == nil && DeliveryState(row.State) == Due
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		l := p.line(c)
		if !due {
			nl, _ := applyVerdict(l, Delivery{}, r.revision, r.verdict, p.now, p.s.cfg.Timings.Ladder, p.s.cfg.Rand)
			nl.GivenUp = l.GivenUp
			p.setLine(c.ID, nl)
			continue
		}
		d, err := deliveryOf(row)
		if err != nil {
			return err
		}
		nl, nd := applyVerdict(l, d, r.revision, r.verdict, p.now, p.s.cfg.Timings.Ladder, p.s.cfg.Rand)
		p.setLine(c.ID, nl)
		p.pub.connections = p.pub.connections || nl.State != l.State || nd.State != Due
		if err := p.store(ctx, c, r, nd); err != nil {
			return err
		}
	}
	return nil
}

// store writes a settled delivery: a sent one goes, and tells its
// connection the Health items it carried or untells those it cleared.
func (p *passing) store(ctx context.Context, c Connection, r result, d Delivery) error {
	key := db.GetDeliveryParams{EventID: r.event, ConnectionID: c.ID}
	switch d.State {
	case Due:
		return p.q.UpdateDue(ctx, db.UpdateDueParams{EventID: key.EventID, ConnectionID: c.ID, Attempts: int64(d.Attempts), Accepted: *jsonText(slices.Clip(d.Accepted))})
	case GivenUp:
		return p.giveUpOne(ctx, key.EventID, c.ID, d.Reason, d.Response)
	case Sent, Dropped:
	}
	if err := p.q.DeleteDelivery(ctx, db.DeleteDeliveryParams(key)); err != nil {
		return err
	}
	switch r.kind {
	case newItemsRow.Kind:
		items, err := p.q.ListItems(ctx, r.event)
		if err != nil {
			return err
		}
		for _, row := range items {
			if slices.Contains(d.Items, row.Key) {
				if err := p.q.UpsertTold(ctx, toldParams(c.ID.String(), itemOf(row), p.now)); err != nil {
					return err
				}
			}
		}
	case clearedRow.Kind:
		for _, k := range d.Items {
			if err := p.q.DeleteTold(ctx, db.DeleteToldParams{Target: c.ID.String(), Key: k}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *passing) giveUpOne(ctx context.Context, e EventID, conn ConnectionID, why providers.Reason, response []byte) error {
	var text *string
	if response != nil {
		t := string(response)
		text = &t
	}
	_, err := p.q.GiveUp(ctx, db.GiveUpParams{EventID: e, ConnectionID: conn, Reason: jsonText(why), Response: text, At: storage.FromTime(p.now)})
	return err
}

func (p *passing) release(ctx context.Context) error {
	pub, hit, err := p.s.releaseDue(ctx, p.q, p.now, p.conns)
	p.pub.feed, p.pub.connections = p.pub.feed || pub.feed, p.pub.connections || pub.connections
	p.hit = p.hit || hit
	return err
}

// giveUp gives up the messages that waited longer than Give up on a
// message after, counting each on its connection's line.
func (p *passing) giveUp(ctx context.Context) error {
	old, err := p.q.ListDueOlderThan(ctx, db.ListDueOlderThanParams{At: storage.FromTime(p.now.Add(-p.giveUpAfter)), Limit: giveUpBound})
	if err != nil {
		return err
	}
	for _, d := range old {
		if err := p.giveUpOne(ctx, d.EventID, d.ConnectionID, providers.Reason{Code: providers.TooOld}, nil); err != nil {
			return err
		}
		l := p.line(p.byID[d.ConnectionID])
		l.GivenUp++
		p.setLine(d.ConnectionID, l)
	}
	p.pub.connections = p.pub.connections || len(old) > 0
	p.hit = p.hit || len(old) == giveUpBound
	return nil
}

// drop deletes the messages of connections turned off and of kinds a
// connection no longer sends.
func (p *passing) drop(ctx context.Context) error {
	off, err := p.q.ListDueOff(ctx, dropBound)
	if err != nil {
		return err
	}
	for _, d := range off {
		if err := p.q.DeleteDelivery(ctx, db.DeleteDeliveryParams{EventID: d.EventID, ConnectionID: d.ConnectionID}); err != nil {
			return err
		}
	}
	p.pub.connections = p.pub.connections || len(off) > 0
	p.hit = p.hit || len(off) == dropBound
	return nil
}

// refresh starts afresh each line whose connection was edited, or passed a
// Test of its saved values, since the line was judged.
func (p *passing) refresh(context.Context) error {
	for _, c := range p.conns {
		if l := p.line(c); l.Revision < c.Revision {
			p.setLine(c.ID, l.fresh(c.Revision, p.now))
			p.pub.connections = true
		}
	}
	return nil
}

// claim reads the oldest message of each idle connection that may send.
// A clear is first cut to the items its connection was told, and goes
// when none are left.
func (p *passing) claim(ctx context.Context, busy map[ConnectionID]bool) error {
	for _, c := range p.conns {
		l := p.line(c)
		if busy[c.ID] || !statusOf(c, l, p.s.nodeView(c), 0, p.now, p.warn).MaySend(l, p.now) {
			continue
		}
		claimed := false
		for range claimBound {
			j, ok, err := p.oldest(ctx, c)
			if err != nil {
				return err
			}
			if !ok {
				claimed = true
				break
			}
			if j != nil {
				p.claimed, claimed = append(p.claimed, *j), true
				break
			}
		}
		p.hit = p.hit || !claimed
	}
	return nil
}

// oldest is c's oldest message as a job. It answers no job and true when
// it dropped a clear narrowed to nothing, and false when c has none due.
func (p *passing) oldest(ctx context.Context, c Connection) (*job, bool, error) {
	row, err := p.q.OldestDue(ctx, c.ID)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	d, err := deliveryOf(row)
	if err != nil {
		return nil, false, err
	}
	ev, err := p.q.GetEvent(ctx, d.Event)
	if err != nil {
		return nil, false, err
	}
	e, err := eventOf(ctx, p.q, ev)
	if err != nil {
		return nil, false, err
	}
	if e.Kind == clearedRow.Kind {
		told, err := p.q.ListToldFor(ctx, d.Items)
		if err != nil {
			return nil, false, err
		}
		was := map[string]bool{}
		for _, t := range told {
			if t.Target == c.ID.String() {
				was[t.Key] = true
			}
		}
		keep := narrow(d.Items, was)
		key := db.DeleteDeliveryParams{EventID: d.Event, ConnectionID: c.ID}
		if len(keep) == 0 {
			p.pub.connections = true
			return nil, true, p.q.DeleteDelivery(ctx, key)
		}
		if len(keep) < len(d.Items) {
			if err := p.q.SetItems(ctx, db.SetItemsParams{EventID: d.Event, ConnectionID: c.ID, Items: jsonText(keep)}); err != nil {
				return nil, false, err
			}
			d.Items = keep
		}
	}
	if isHealth(e.Kind) {
		e.Items = slices.DeleteFunc(e.Items, func(it item) bool { return !slices.Contains(d.Items, it.key()) })
		subject, err := json.Marshal(healthFacts{Items: e.Items})
		if err != nil {
			return nil, false, fmt.Errorf("encode the Health items of event %d: %w", e.ID, err)
		}
		e.Subject = subject
	}
	return &job{conn: c, event: e, items: d.Items, accepted: d.Accepted}, true, nil
}

// nextAt is when the next pass is due if nothing wakes the dispatcher:
// now after a bound was reached, else the earliest of the next event's due
// time, the next retry of a connection with a message due, and the next
// give-up.
func (p *passing) nextAt(ctx context.Context, busy map[ConnectionID]bool) error {
	if p.hit {
		p.next = p.now
		return nil
	}
	earliest := func(t time.Time) {
		if !t.IsZero() && (p.next.IsZero() || t.Before(p.next)) {
			p.next = t
		}
	}
	due, err := p.q.NextUnreleased(ctx)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	earliest(due.Time())
	counts, err := p.q.CountDue(ctx)
	if err != nil {
		return err
	}
	for _, n := range counts {
		c, ok := p.byID[n.ConnectionID]
		if !ok || busy[c.ID] || slices.ContainsFunc(p.claimed, func(j job) bool { return j.conn.ID == c.ID }) {
			continue
		}
		l := p.line(c)
		if st := statusOf(c, l, p.s.nodeView(c), 0, p.now, p.warn).Shown; st == ShownFailing || st == ShownWaiting {
			earliest(l.Next)
		}
	}
	oldest, err := p.q.OldestDueAt(ctx)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	if !oldest.IsZero() {
		// A message is given up once it is older than the limit, so the
		// first instant after it, at storage's precision.
		earliest(oldest.Time().Add(p.giveUpAfter + time.Microsecond))
	}
	return nil
}

func (p *passing) save(ctx context.Context) error {
	for id := range p.dirty {
		if err := p.q.UpsertLine(ctx, p.lines[id].params(id)); err != nil {
			return err
		}
	}
	return nil
}

func deliveryOf(row db.NotificationsDelivery) (Delivery, error) {
	d := Delivery{Event: row.EventID, State: DeliveryState(row.State), Attempts: int(row.Attempts)}
	var err error
	if d.Items, err = keysFrom(row.Items); err != nil {
		return Delivery{}, err
	}
	if err := json.Unmarshal([]byte(row.Accepted), &d.Accepted); err != nil {
		return Delivery{}, fmt.Errorf("decode the accepted addresses of a delivery: %w", err)
	}
	return d, nil
}
