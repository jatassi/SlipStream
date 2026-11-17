package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/setting"
)

// HealthAbout is the Notifications category: one item per connection that
// is on and isn't sending.
var HealthAbout = health.About{
	ID: health.NotificationsID, Label: "Notifications", Phrase: "notification connections",
	Events: []live.Event{ChangedEvent, nodes.ChangedEvent},
}

// HealthSource builds method values only, so a Node can bind it before the
// service runs.
func (s *Service) HealthSource() health.Source {
	return health.Source{Answer: s.answer}
}

// answer words each connection that is paused, or has failed or waited for
// at least Warn when a connection has failed for. Next is when the next
// one crosses that time.
func (s *Service) answer(ctx context.Context) (health.Answer, error) {
	var a health.Answer
	views, warn, err := s.statuses(ctx)
	if err != nil {
		return a, err
	}
	w := s.wording(time.Now(), 0)
	for _, v := range views {
		st := v.Status
		if (st.Shown == ShownFailing || st.Shown == ShownWaiting) && st.Level == health.None {
			if next := st.Since.Add(warn); a.Next.IsZero() || next.Before(a.Next) {
				a.Next = next
			}
		}
		if st.Level == health.None {
			continue
		}
		label, reason := s.label(v.Connection), w.reason(s.provider(v.Provider), st.Reason)
		it := health.Item{Key: health.KeyOf(v.ID.String()), Level: st.Level}
		it.Action = health.LinkAction("/settings/notifications", "Open Notifications")
		switch st.Shown {
		case ShownPaused:
			it.Subject = label + " is paused"
			it.Text = reason + " Edit the connection to fix it, then test it."
		case ShownWaiting:
			it.Subject = "Messages to " + label + " are waiting"
			it.Text = fmt.Sprintf("Messages have waited since %s. %s %s", s.clock(st.Since), reason, waitingWords(st.Waiting))
		case ShownFailing:
			it.Subject = "Messages to " + label + " are failing"
			it.Text = fmt.Sprintf("Messages have failed since %s. %s %s", s.clock(st.Since), reason, waitingWords(st.Waiting))
		case ShownOff, ShownOK, ShownHolding:
		}
		a.Items = append(a.Items, it)
	}
	return a, nil
}

func waitingWords(n int) string {
	if n == 1 {
		return "1 message is waiting."
	}
	return setting.Grouped(n) + " messages are waiting."
}

func (s *Service) clock(t time.Time) string { return t.In(s.cfg.Zone).Format("15:04") }
