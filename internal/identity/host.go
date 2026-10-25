package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/storage"
)

// ResetRequest is `slipstream admin reset <email> [--address <origin>]`.
type ResetRequest struct {
	Email   string `json:"email"`
	Address string `json:"address,omitempty"`
}

// ResetReply holds one of: the Reset link, or why the command refused.
type ResetReply struct {
	URL     string       `json:"url,omitempty"`
	Refused *HostRefusal `json:"refused,omitempty"`
}

// ResetOp issues a Reset link on a running Controller, whose hub ends the
// Account's open streams. A Node serves it once ready.
var ResetOp = op.Op[ResetRequest, ResetReply]{Name: "identity.reset"}

// HostRefusal is a reset the command refuses as a usage error: the machine
// isn't a Controller, or the address is missing, an IP or not the one set.
type HostRefusal struct {
	Text string `json:"text"`
}

func (e *HostRefusal) Error() string { return e.Text }

func NotAController() *HostRefusal {
	return &HostRefusal{Text: "this machine isn't a Controller, so it has no Accounts to reset"}
}

// ResetFromHost issues a Reset link for the Account with req's email and
// ends its sessions, in one Write, and sends no email: the operator gets
// the URL. Nobody issues it, so its dead card says "your Admin".
// req.Address sets the address only while none is set.
func (s *Service) ResetFromHost(ctx context.Context, req ResetRequest) (string, error) {
	var address Address
	if req.Address != "" {
		var err error
		if address, err = ParseAddress(req.Address); err != nil {
			return "", &HostRefusal{Text: fmt.Sprintf("--address %s: %v", req.Address, hostAddressError(err))}
		}
	}
	email := normalizeEmail(req.Email)
	var out issued
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if err := hostAddress(ctx, q, address); err != nil {
			return err
		}
		row, err := q.GetAccountByEmail(ctx, email)
		if errors.Is(err, storage.ErrNoRows) {
			return fmt.Errorf("no Account has the email %s", email)
		}
		if err != nil {
			return err
		}
		if !row.DisabledAt.IsZero() {
			return fmt.Errorf("the Account %s is turned off. An Admin can turn it back on", email)
		}
		if out, err = s.issue(ctx, q, fx, resetKind, linkTarget{email: email, account: row.ID}, AccountID{}); err != nil {
			return err
		}
		return endAllSessions(ctx, q, fx, row.ID)
	})
	return out.url, err
}

func hostAddressError(err error) string {
	if errors.Is(err, ErrAddressIP) {
		return "the address must be a name, not an IP address"
	}
	return "expected http:// or https:// then a host name and an optional port"
}

func hostAddress(ctx context.Context, q *db.Queries, address Address) error {
	set, err := readSettings(ctx, q)
	if err != nil {
		return err
	}
	switch {
	case set.Address.IsZero() && address.IsZero():
		return &HostRefusal{Text: "SlipStream has no address for links yet. Pass --address <origin>, such as --address https://slipstream.example.com"}
	case set.Address.IsZero():
		return q.SetAddress(ctx, address.origin)
	case !address.IsZero() && address != set.Address:
		return &HostRefusal{Text: fmt.Sprintf("SlipStream's address is already %s. Leave out --address, or change it in Settings > Address", set.Address)}
	}
	return nil
}
