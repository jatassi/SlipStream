package indexers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

var cookiesColumn = secret.NewColumn("indexers_session.cookies_sealed")

func sessionAD(id ID, node role.NodeID) secret.AD {
	return cookiesColumn.Row(id.String() + "/" + node.String())
}

// offer stores s as id's session on node unless the stored one was earned
// as late or later, so a replayed or older answer changes nothing.
func (x *Indexers) offer(ctx context.Context, tx *storage.Tx[file], id ID, node role.NodeID, s searching.Session) error {
	cookies, err := json.Marshal(s.Cookies)
	if err != nil {
		return fmt.Errorf("encode the session's cookies: %w", err)
	}
	_, err = db.New(tx).OfferSession(ctx, db.OfferSessionParams{
		IndexerID: id, NodeID: node, CookiesSealed: x.cfg.Sealer.Seal(cookies, sessionAD(id, node)), UserAgent: s.UserAgent,
		Pasted: s.Pasted, EarnedAt: storage.FromTime(s.Earned),
	})
	return err
}

// session is id's stored session on node, opened, or nil when it has none.
func (x *Indexers) session(ctx context.Context, r storage.Reader[file], id ID, node role.NodeID) (*searching.Session, error) {
	row, err := db.New(r).GetSession(ctx, db.GetSessionParams{IndexerID: id, NodeID: node})
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plain, err := x.cfg.Sealer.Open(row.CookiesSealed, sessionAD(id, node))
	if err != nil {
		return nil, fmt.Errorf("indexer %s: open the session on node %s: %w", id, node, err)
	}
	s := &searching.Session{UserAgent: row.UserAgent, Pasted: row.Pasted, Earned: row.EarnedAt.Time()}
	return s, decode(string(plain), &s.Cookies)
}

// SessionFact is the newest session an Indexer holds, without its
// cookies.
type SessionFact struct {
	Node   role.NodeID
	Pasted bool
	Earned time.Time
}

// sessionFacts is each Indexer's newest session.
func sessionFacts(ctx context.Context, r storage.Reader[file]) (map[ID]SessionFact, error) {
	rows, err := db.New(r).ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	out := map[ID]SessionFact{}
	for _, row := range rows {
		if _, ok := out[row.IndexerID]; !ok {
			out[row.IndexerID] = SessionFact{Node: row.NodeID, Pasted: row.Pasted, Earned: row.EarnedAt.Time()}
		}
	}
	return out, nil
}
