// Package indexerstest is fixtures for the tests of the indexers' callers:
// a roster and a saved Torznab or catalog Indexer.
package indexerstest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Roster is an indexers.Roster where Holders hold Searching, the
// Controller first, and Absent are away.
type Roster struct {
	Controller role.NodeID
	Holders    []role.NodeID
	Absent     map[role.NodeID]nodes.Absence
}

// Alone is a roster of one Node holding Searching.
func Alone(self role.NodeID) Roster { return Roster{Controller: self, Holders: []role.NodeID{self}} }

func (r Roster) Self() role.NodeID { return r.Controller }

func (r Roster) Holding(role.Name) []role.NodeID { return r.Holders }

func (r Roster) may(n role.NodeID) bool {
	_, away := r.Absent[n]
	return slices.Contains(r.Holders, n) && !away
}

func (r Roster) Pick(_ role.Name, prefer role.NodeID) (role.NodeID, bool) {
	if r.may(prefer) {
		return prefer, true
	}
	return r.PickOther(role.Searching, prefer)
}

func (r Roster) PickOther(_ role.Name, besides role.NodeID) (role.NodeID, bool) {
	for _, n := range r.Holders {
		if n != besides && r.may(n) {
			return n, true
		}
	}
	return role.NodeID{}, false
}

func (r Roster) Away(_ role.Name, n role.NodeID) (nodes.Absence, bool) {
	a, ok := r.Absent[n]
	return a, ok
}

func (r Roster) Name(n role.NodeID) string {
	if n == r.Controller {
		return "home"
	}
	return "node " + n.String()
}

// Type is a Content Type searched in the Newznab family 2000.
const Type works.TypeID = "searched"

// Facet is Type's: the family 2000 and the zero Terms.
type Facet struct{}

func (Facet) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

func (Facet) Categories() []newznab.ID { return newznab.Family(2000) }

// Caps are a saved Torznab Indexer's: a free-text search, a search by
// id, and the categories of family 2000.
const Caps = `{"modes":{"search":["q"],"movie":["q","imdbid","tmdbid"]},"categories":[2000]}`

// Torznab saves a Torznab Indexer homed on home, at revision 1 with no
// status and Caps, in tx.
func Torznab(ctx context.Context, tx *storage.Tx[storage.Library], sealer *secret.Sealer, id indexers.ID, name string, home role.NodeID, address, key string) error {
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("parse the Indexer address: %w", err)
	}
	plain, err := json.Marshal(indexers.Secrets{Address: address, APIKey: key}) //nolint:gosec // G117: sealed below
	if err != nil {
		return fmt.Errorf("encode the secrets: %w", err)
	}
	err = db.New(tx).InsertIndexer(ctx, db.InsertIndexerParams{
		ID: id, Name: name, Kind: string(indexers.Torznab), HomeNodeID: home, IsOn: true, AutoSearch: true, Rss: true,
		Priority: 25, SearchWindow: "hour", GrabWindow: "hour", Plain: "{}",
		SecretsSealed: sealer.Seal(plain, secret.NewColumn("indexers_indexer.secrets_sealed").Row(id.String())),
		FeedHost:      &u.Host, Caps: Caps, Notes: "[]", Revision: 1, CreatedAt: storage.Now(), UpdatedAt: storage.Now(),
	})
	if err != nil {
		return fmt.Errorf("save Indexer %s: %w", name, err)
	}
	return nil
}

// Catalog saves a catalog Indexer homed on home that pins version sha of
// definition, which tx already holds, with its plain settings and its
// sealed fields, at revision 1 with no status.
func Catalog(ctx context.Context, tx *storage.Tx[storage.Library], sealer *secret.Sealer, id indexers.ID, name string, home role.NodeID, definition, sha string, plain, sealed map[string]string) error {
	secrets, err := json.Marshal(indexers.Secrets{Fields: sealed}) //nolint:gosec // G117: sealed below
	if err != nil {
		return fmt.Errorf("encode the secrets: %w", err)
	}
	settings, err := json.Marshal(plain)
	if err != nil {
		return fmt.Errorf("encode the settings: %w", err)
	}
	base := ""
	err = db.New(tx).InsertIndexer(ctx, db.InsertIndexerParams{
		ID: id, Name: name, Kind: string(indexers.Catalog), HomeNodeID: home, IsOn: true, AutoSearch: true, Rss: true,
		Priority: 25, SearchWindow: "hour", GrabWindow: "hour", DefinitionID: &definition, DefinitionSha: &sha, BaseAddress: &base,
		Plain: string(settings), SecretsSealed: sealer.Seal(secrets, secret.NewColumn("indexers_indexer.secrets_sealed").Row(id.String())),
		Caps: `{"modes":{}}`, Notes: "[]", Revision: 1, CreatedAt: storage.Now(), UpdatedAt: storage.Now(),
	})
	if err != nil {
		return fmt.Errorf("save Indexer %s: %w", name, err)
	}
	return nil
}
