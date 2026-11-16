package notifications

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestResult is how a Test went. A Test writes no delivery and no line.
type TestResult struct {
	Passed bool
	// Node is the Node that sent, or would have.
	Node string
	// Reason is why it failed, "" when it passed.
	Reason string
	// Field is the field the failure names, "" for none.
	Field string
}

// Test sends the Test message for d from the Node that would send it now.
func (s *Service) Test(ctx context.Context, d Draft) (TestResult, error) {
	c, secrets, err := s.draft(d, time.Now())
	if err != nil {
		return TestResult{}, err
	}
	return s.test(ctx, c, secrets), nil
}

// TestSaved lays e over saved connection id and sends the Test. A pass
// with no edits moves the revision, which unpauses it. With edits only the
// Save does.
func (s *Service) TestSaved(ctx context.Context, id ConnectionID, e Edit) (TestResult, error) {
	var (
		c       Connection
		secrets providers.Values
	)
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		c, secrets, err = s.saved(ctx, db.New(r), id)
		return err
	})
	if err != nil {
		return TestResult{}, err
	}
	if c, secrets, err = s.edited(c, secrets, e, time.Now()); err != nil {
		return TestResult{}, err
	}
	res := s.test(ctx, c, secrets)
	if !res.Passed || !e.empty() {
		return res, nil
	}
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).BumpRevision(ctx, id)
	})
	if err != nil {
		return TestResult{}, err
	}
	s.changed(ctx)
	return res, nil
}

func (s *Service) test(ctx context.Context, c Connection, secrets providers.Values) TestResult {
	now := time.Now()
	v, node := s.attempt(ctx, c, secrets, event{Kind: testRow.Kind, At: now, Subject: []byte("{}")}, nil)
	if node.IsZero() && c.script() {
		node = c.Node
	}
	res := TestResult{Passed: v.Outcome == providers.Sent}
	if !node.IsZero() {
		res.Node = s.names(node)
	}
	if !res.Passed {
		res.Reason = s.wording(now, 0).reason(s.provider(c.Provider), v.Reason)
		res.Field = v.Reason.Field
	}
	return res
}
