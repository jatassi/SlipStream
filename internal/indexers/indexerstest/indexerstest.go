// Package indexerstest is fixtures for the tests of the indexers' callers:
// a roster, a saved Torznab, catalog or native Indexer, and a scripted
// Searching Node.
package indexerstest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Roster is an indexers.Roster where Holders hold Searching, the
// Controller first, and Absent are away. A Node missing from Names is
// "home" when it is the Controller, else "node <id>".
type Roster struct {
	Controller role.NodeID
	Holders    []role.NodeID
	Absent     map[role.NodeID]nodes.Absence
	Names      map[role.NodeID]string
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
	if name, ok := r.Names[n]; ok {
		return name
	}
	if n == r.Controller {
		return "home"
	}
	return "node " + n.String()
}

// Type is a Content Type searched in the Newznab family 2000.
const Type works.TypeID = "searched"

// Facet is Type's: the family 2000, the zero Expect and the zero Terms.
type Facet struct{}

func (Facet) Expect(context.Context, storage.Reader[storage.Library], works.WorkID) (indexers.Expect, error) {
	return indexers.Expect{}, nil
}

func (Facet) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

func (Facet) Categories() []newznab.ID { return newznab.Family(2000) }

// Caps are a saved Torznab Indexer's: a free-text search, a search by
// id, and the categories of family 2000.
const Caps = `{"modes":{"search":["q"],"movie":["q","imdbid","tmdbid"]},"categories":[2000]}`

// Torznab saves a Torznab Indexer homed on home, at revision 1 with no
// status and Caps, whose sourceless items reach a Tracker of its name, in
// tx.
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
		FeedHost:      &u.Host, TrackerName: &name, Caps: Caps, Notes: "[]", Revision: 1, CreatedAt: storage.Now(), UpdatedAt: storage.Now(),
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

// Native saves a native Indexer homed on home that runs port at address,
// with its plain settings and its sealed fields, at revision 1 with no
// status, even for a port the registry no longer carries.
func Native(ctx context.Context, tx *storage.Tx[storage.Library], sealer *secret.Sealer, id indexers.ID, name string, home role.NodeID, port, address string, plain, sealed map[string]string) error {
	secrets, err := json.Marshal(indexers.Secrets{Fields: sealed}) //nolint:gosec // G117: sealed below
	if err != nil {
		return fmt.Errorf("encode the secrets: %w", err)
	}
	settings, err := json.Marshal(plain)
	if err != nil {
		return fmt.Errorf("encode the settings: %w", err)
	}
	err = db.New(tx).InsertIndexer(ctx, db.InsertIndexerParams{
		ID: id, Name: name, Kind: string(indexers.Native), HomeNodeID: home, IsOn: true, AutoSearch: true, Rss: true,
		Priority: 25, SearchWindow: "hour", GrabWindow: "hour", PortID: &port, BaseAddress: &address,
		Plain: string(settings), SecretsSealed: sealer.Seal(secrets, secret.NewColumn("indexers_indexer.secrets_sealed").Row(id.String())),
		Caps: Caps, Notes: "[]", Revision: 1, CreatedAt: storage.Now(), UpdatedAt: storage.Now(),
	})
	if err != nil {
		return fmt.Errorf("save Indexer %s: %w", name, err)
	}
	return nil
}

// TVCaps are a Torznab Indexer's that searches TV by TVDB id, season and
// episode, beside Caps' modes, in the families 2000 and 5000.
const TVCaps = `{"modes":{"search":["q"],"movie":["q","imdbid","tmdbid"],"tvsearch":["q","season","ep","tvdbid"]},"categories":[2000,5000]}`

// SetCaps sets the caps of Indexer id, as a test of it would, in tx.
func SetCaps(ctx context.Context, tx *storage.Tx[storage.Library], id indexers.ID, caps string) error {
	if err := db.New(tx).SetCaps(ctx, db.SetCapsParams{ID: id, Caps: caps}); err != nil {
		return fmt.Errorf("set the caps of Indexer %s: %w", id, err)
	}
	return nil
}

// Node is a searching.Remote whose Indexers answer each job a set time
// after its first Run: inside a testing/synctest bubble, at that virtual
// time. An Indexer with no answer set has none, which reads as no
// Releases, at once. A Run holds up to hold, as a Node's does, so a caller
// learns when the job started and its deadline can pass.
type Node struct {
	mu       sync.Mutex
	answers  map[string]timed
	started  map[string]time.Time
	jobs     []searching.Job
	canceled []string
}

// hold is a Node's default hold: the longest a Run waits before it answers
// how the job stands.
const hold = 20 * time.Second

type timed struct {
	after  time.Duration
	answer searching.Answer
}

// Answer has Indexer id answer every later job with a, after the job's
// first Run.
func (n *Node) Answer(id indexers.ID, after time.Duration, a searching.Answer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.answers == nil {
		n.answers = map[string]timed{}
	}
	n.answers[id.String()] = timed{after: after, answer: a}
}

func (*Node) ReachSolver(context.Context, searching.Solver) (int, error) {
	return 0, errors.New("the scripted Node has no Challenge solver")
}

func (n *Node) Run(ctx context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	if n.started == nil {
		n.started = map[string]time.Time{}
	}
	started, ok := n.started[j.ID]
	if !ok {
		started = time.Now()
		n.started[j.ID] = started
		n.jobs = append(n.jobs, j)
	}
	script := n.answers[j.Indexer]
	n.mu.Unlock()
	at := started.Add(script.after)
	held := time.NewTimer(min(time.Until(at), hold))
	defer held.Stop()
	select {
	case <-held.C:
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			n.mu.Lock()
			n.canceled = append(n.canceled, j.Indexer)
			n.mu.Unlock()
		}
		return searching.Poll{}, fmt.Errorf("hold the call: %w", ctx.Err())
	}
	if time.Now().Before(at) {
		return searching.Poll{Started: started}, nil
	}
	a := script.answer
	a.Started = started
	return searching.Poll{Started: started, Answer: &a}, nil
}

// Jobs is every job the Node ran, in the order each first reached it.
func (n *Node) Jobs() []searching.Job {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.jobs)
}

// Searches is the query of each search job the Node ran, in the order
// each first reached it.
func (n *Node) Searches() []searching.Query {
	var out []searching.Query
	for _, j := range n.Jobs() {
		if c, ok := j.Call.(searching.SearchCall); ok {
			out = append(out, c.Query)
		}
	}
	return out
}

// Canceled is the Indexer of each Run its caller canceled before the job
// answered, as the job names it, in order.
func (n *Node) Canceled() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.canceled)
}
