// Package nodesapi holds the nodes owner's operations: Join Codes, the
// Nodes an Admin sees, their Roles and the nodes settings.
package nodesapi

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

// Roster is the Node's nodes owner as its operations reach it.
type Roster interface {
	// Nodes is the nodes service, false unless the Node is a Controller.
	Nodes() (*nodes.Service, bool)
}

// open is the nodes service of an operation an Admin check admitted, which
// only a Controller admits.
func open(r Roster) *nodes.Service {
	s, ok := r.Nodes()
	if !ok {
		panic("nodesapi: an Admin operation ran on a Node that is no Controller")
	}
	return s
}

var (
	noLinkAddress = api.NewCode(http.StatusConflict, "no_link_address", "The link has no address to put in a Join Code")
	nodeNotFound  = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	rolesStale    = api.NewCarrying[outcome](http.StatusConflict, "roles_stale", "The Roles changed since this was planned", "outcome")
	rolesBlocked  = api.NewCarrying[outcome](http.StatusConflict, "roles_blocked", "A guardrail blocks this change", "outcome")
	nothingHeld   = api.NewCarrying[outcome](http.StatusConflict, "nothing_held", "The Node holds nothing to write off for these Roles", "outcome")
	keepsRole     = api.NewCarrying[outcome](http.StatusConflict, "keeps_role", "The Node is the last that can hold a Role it would write off", "outcome")
	isController  = api.NewCarrying[outcome](http.StatusConflict, "node_is_controller", "The Controller can't be removed; hand it over instead", "outcome")
	holdsWork     = api.NewCarrying[outcome](http.StatusConflict, "node_holds_work", "The Node holds work; remove it with writeOff to write the work off", "outcome")
	nameTaken     = api.NewCode(http.StatusConflict, "node_name_taken", "Another Node has this name")
	nameInvalid   = api.NewCode(http.StatusUnprocessableEntity, "node_name_invalid", "A Node's name is 1 to 63 characters, with no space at either end")
)

// blockedAs is the code each guardrail that can block a change answers
// with.
var blockedAs = map[nodes.Reason]api.Carrying[outcome]{
	nodes.RoleTaken:    rolesBlocked,
	nodes.ProbeFails:   rolesBlocked,
	nodes.ReachFailed:  rolesBlocked,
	nodes.NothingHeld:  nothingHeld,
	nodes.KeepsRole:    keepsRole,
	nodes.IsController: isController,
	nodes.HoldsWork:    holdsWork,
}

type node struct {
	ID        string     `json:"id" format:"uuid"`
	Name      string     `json:"name"`
	Key       string     `json:"key" doc:"The key's fingerprint"`
	Self      bool       `json:"self"`
	PairedAt  time.Time  `json:"pairedAt"`
	Addresses []string   `json:"addresses" nullable:"false"`
	Roles     []roleView `json:"roles" nullable:"false" doc:"Every Role the Node stands in, in table order"`
	// IntentVersion is the node-scope version the library derives now.
	IntentVersion string   `json:"intentVersion" pattern:"^[0-9a-f]{32}$"`
	Link          link     `json:"link"`
	Standing      string   `json:"standing" enum:"in-step,reconnecting,offline,lapsed,mismatched" doc:"How the Node stands with the Controller since link.since: in step, reconnecting within the blip, offline, lapsed past the offline grace, or on another version"`
	Dial          dialView `json:"dial"`
	Facts         *facts   `json:"facts,omitempty" doc:"Absent while the Node has no session"`
	Probes        []probe  `json:"probes" nullable:"false" doc:"The last verdict of each probe the Node ran, kept while it is away"`
}

type probe struct {
	Probe  string    `json:"probe" enum:"engine-starts,data-writable,listener-up"`
	Fit    string    `json:"fit" enum:"fits,weak,fails"`
	Reason string    `json:"reason,omitempty" enum:"engine_not_listening,downloads_not_writable,downloads_on_network_share,data_not_writable,http_not_listening" doc:"Why it is weak or fails; absent when it fits"`
	At     time.Time `json:"at"`
}

func inTableOrder(m map[role.Probe]role.ProbeResult) []role.ProbeResult {
	var out []role.ProbeResult
	for _, p := range role.Probes() {
		if r, ok := m[p]; ok {
			out = append(out, r)
		}
	}
	return out
}

func probesOf(results []role.ProbeResult) []probe {
	out := []probe{}
	for _, p := range results {
		out = append(out, probe{Probe: string(p.Probe), Fit: p.Fit.String(), Reason: string(p.Reason), At: p.At})
	}
	return out
}

type roleView struct {
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	State string `json:"state" enum:"holding,away,lapsed,draining,writing-off,covering"`
	Held  *held  `json:"held,omitempty" doc:"The tied work the Node holds for the Role; absent when none"`
}

type held struct {
	Units int64  `json:"units"`
	Noun  string `json:"noun" doc:"The noun that fits units: torrent, torrents"`
	Bytes int64  `json:"bytes" doc:"0 when sizes don't apply"`
}

func rolesOf(views []nodes.RoleView) []roleView {
	out := make([]roleView, len(views))
	for i, v := range views {
		out[i] = roleView{Role: string(v.Role), State: v.State.String()}
		if v.Held.Units > 0 {
			h := heldOf(v.Held)
			out[i].Held = &h
		}
	}
	return out
}

func heldOf(h nodes.Held) held {
	noun := h.Noun.Many
	if h.Units == 1 {
		noun = h.Noun.One
	}
	return held{Units: h.Units, Noun: noun, Bytes: h.Bytes}
}

type link struct {
	State string    `json:"state" enum:"local,up,down,mismatched"`
	Since time.Time `json:"since"`
	Build *build    `json:"build,omitempty" doc:"Absent while the link is down"`
}

type build struct {
	Version     string `json:"version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	InstallKind string `json:"installKind"`
}

type facts struct {
	Assigned       []string `json:"assigned" nullable:"false"`
	Running        []string `json:"running" nullable:"false"`
	Restarting     []string `json:"restarting" nullable:"false"`
	AppliedVersion string   `json:"appliedVersion"`
	HTTPAddress    string   `json:"httpAddress" doc:"The address the Node's web server bound; empty until it binds"`
}

func nodeOf(n nodes.Node) node {
	out := node{
		ID:            n.ID.String(),
		Name:          n.Name,
		Key:           n.Key.Fingerprint(),
		Self:          n.Self,
		PairedAt:      n.PairedAt,
		Addresses:     []string{},
		Roles:         rolesOf(n.Roles),
		IntentVersion: hex.EncodeToString(n.Version),
		Link:          link{State: n.Link.String(), Since: n.Since},
		Standing:      n.Standing.String(),
		Dial:          dialOf(n.Dial),
		Probes:        probesOf(n.Probes),
	}
	for _, a := range n.Addresses.All() {
		out.Addresses = append(out.Addresses, a.String())
	}
	if b := n.Build; b != nil {
		out.Link.Build = &build{Version: b.Version, OS: b.OS, Arch: b.Arch, InstallKind: b.InstallKind.String()}
	}
	if f := n.Facts; f != nil {
		out.Facts = &facts{
			Assigned:       nonNil(f.GetAssigned()),
			Running:        nonNil(f.GetRunning()),
			Restarting:     nonNil(f.GetRestarting()),
			AppliedVersion: hex.EncodeToString(f.GetAppliedVersion()),
			HTTPAddress:    f.GetHttpAddress(),
		}
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

type nodeOutput struct {
	Body node
}

type nodeList struct {
	Nodes []node `json:"nodes" nullable:"false"`
}

type listOutput struct {
	Body nodeList
}

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

type rolesInput struct {
	Body rolesChange
}

type planInput struct {
	Body planChange
}

type planChange struct {
	Basis string       `json:"basis" doc:"The basis of the board or Outcome the change was planned on"`
	Set   []assignment `json:"set" nullable:"false" doc:"The Roles of each Node the change names. A Node it doesn't name keeps its own."`
	Test  bool         `json:"test,omitempty" doc:"Run the reach test of each added Role whose target is set, from its Node; setRoles always runs them"`
}

type rolesChange struct {
	Basis string       `json:"basis" doc:"The basis of the board or Outcome the change was planned on"`
	Set   []assignment `json:"set" nullable:"false" doc:"The Roles of each Node the change names. A Node it doesn't name keeps its own."`
}

// change is the set as nodes reads it, or the problem with it.
func (b rolesChange) change() (nodes.SetRoles, error) {
	c := nodes.SetRoles{Basis: nodes.Basis(b.Basis), Want: nodes.Roles{}}
	for i, a := range b.Set {
		id, err := nodeID(a.NodeID)
		if err != nil {
			return nodes.SetRoles{}, err
		}
		if _, twice := c.Want[id]; twice {
			return nodes.SetRoles{}, api.Invalid.On(fmt.Sprintf("set[%d].nodeId", i), "This Node is already in the set.")
		}
		var set role.Set
		for _, r := range a.Roles {
			name, err := role.ParseName(r)
			if err != nil {
				return nodes.SetRoles{}, api.Invalid
			}
			set = set.With(name)
		}
		c.Want[id] = set
	}
	return c, nil
}

type writeOffInput struct {
	ID   string `path:"id" format:"uuid"`
	Body writeOffBody
}

type writeOffBody struct {
	Roles []string `json:"roles" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" minItems:"1" nullable:"false" doc:"The Roles whose work the Node holds to give up. A Role still ticked on the Node is unticked."`
}

// change is the write-off as nodes reads it, or the problem with it.
func (in writeOffInput) change() (nodes.WriteOff, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nodes.WriteOff{}, err
	}
	var set role.Set
	for _, r := range in.Body.Roles {
		name, err := role.ParseName(r)
		if err != nil {
			return nodes.WriteOff{}, api.Invalid
		}
		set = set.With(name)
	}
	return nodes.WriteOff{Node: id, Roles: set}, nil
}

type renameInput struct {
	ID   string `path:"id" format:"uuid"`
	Body renameBody
}

type renameBody struct {
	Name string `json:"name" doc:"1 to 63 characters, with no space at either end, that no other Node has"`
}

type removeInput struct {
	ID       string `path:"id" format:"uuid"`
	WriteOff bool   `query:"writeOff" doc:"Write off the work the Node holds, files kept, instead of refusing while it holds any"`
}

type assignment struct {
	NodeID string   `json:"nodeId" format:"uuid"`
	Roles  []string `json:"roles" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" nullable:"false"`
}

func assignments(rs nodes.Roles) []assignment {
	out := []assignment{}
	for _, id := range slices.SortedFunc(maps.Keys(rs), func(a, b nodes.ID) int { return a.Compare(b.UUID) }) {
		out = append(out, assignment{NodeID: id.String(), Roles: names(rs[id])})
	}
	return out
}

func names(set role.Set) []string {
	out := []string{}
	for _, r := range set.Names() {
		out = append(out, string(r))
	}
	return out
}

type named struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

func namedOf(n nodes.Named) named { return named{ID: n.ID.String(), Name: n.Name} }

// outcome is what a change of Roles does: the whole assignment and
// forfeits it stores, what happens and the guardrails' answers.
type outcome struct {
	Basis   string              `json:"basis" doc:"The basis of the rows it was planned on"`
	Next    []assignment        `json:"next" nullable:"false" doc:"Every Node's Roles once applied"`
	Forfeit []assignment        `json:"forfeit" nullable:"false" doc:"Every Node's written-off Roles once applied"`
	Effects []api.OneOf[effect] `json:"effects" nullable:"false"`
	Checks  []check             `json:"checks" nullable:"false"`
}

type outcomeOutput struct {
	Body outcome
}

func outcomeOf(o nodes.Outcome) outcome {
	out := outcome{Basis: string(o.Basis), Next: assignments(o.Next), Forfeit: assignments(o.Forfeit), Effects: []api.OneOf[effect]{}, Checks: []check{}}
	for _, e := range o.Effects {
		out.Effects = append(out.Effects, api.NewOneOf[effect](effectOf(e)))
	}
	for _, c := range o.Checks {
		out.Checks = append(out.Checks, check{
			Node: namedOf(c.Node), Role: string(c.Role), Verdict: c.Verdict.String(), Reason: string(c.Reason), Words: c.Words,
		})
	}
	return out
}

type check struct {
	Node    named  `json:"node"`
	Role    string `json:"role,omitempty" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" doc:"Absent for a check on the whole Node"`
	Verdict string `json:"verdict" enum:"allow,warn,block"`
	Reason  string `json:"reason" enum:"role_taken,probe_fails,probe_weak,offline_unchecked,reach_failed,reached,nothing_to_test,node_is_controller,node_holds_work,nothing_held,keeps_role"`
	Words   string `json:"words"`
}

type effect struct{}

func (effect) Variants() (discriminator string, variants []any) {
	return "kind", []any{gains{}, letsGo{}, drains{}, handedBack{}, pinned{}, moves{}, writesOff{}, leaves{}}
}

type gains struct {
	Kind  string `json:"kind" enum:"gains"`
	Node  named  `json:"node"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	Words string `json:"words"`
}

type letsGo struct {
	Kind  string `json:"kind" enum:"lets-go"`
	Node  named  `json:"node"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	Words string `json:"words"`
}

type drains struct {
	Kind  string `json:"kind" enum:"drains"`
	Node  named  `json:"node"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	Held  held   `json:"held"`
	Words string `json:"words"`
}

type handedBack struct {
	Kind  string  `json:"kind" enum:"handed-back"`
	Role  string  `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	From  []named `json:"from" nullable:"false" doc:"Every Node that held the Role before"`
	To    named   `json:"to"`
	Words string  `json:"words"`
}

type pinned struct {
	Kind  string `json:"kind" enum:"pinned"`
	Node  named  `json:"node"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	Words string `json:"words"`
}

type moves struct {
	Kind  string `json:"kind" enum:"moves"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	From  named  `json:"from"`
	To    named  `json:"to"`
	Held  held   `json:"held"`
	Words string `json:"words"`
}

type writesOff struct {
	Kind  string `json:"kind" enum:"writes-off"`
	Node  named  `json:"node"`
	Role  string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	Held  held   `json:"held"`
	Words string `json:"words"`
}

type leaves struct {
	Kind  string `json:"kind" enum:"leaves"`
	Node  named  `json:"node"`
	Words string `json:"words"`
}

func effectOf(e nodes.Effect) any {
	switch e := e.(type) {
	case nodes.Gains:
		return gains{Kind: "gains", Node: namedOf(e.Node), Role: string(e.Role), Words: e.Words()}
	case nodes.LetsGo:
		return letsGo{Kind: "lets-go", Node: namedOf(e.Node), Role: string(e.Role), Words: e.Words()}
	case nodes.Drains:
		return drains{Kind: "drains", Node: namedOf(e.Node), Role: string(e.Role), Held: heldOf(e.Held), Words: e.Words()}
	case nodes.HandedBack:
		from := []named{}
		for _, n := range e.From {
			from = append(from, namedOf(n))
		}
		return handedBack{Kind: "handed-back", Role: string(e.Role), From: from, To: namedOf(e.To), Words: e.Words()}
	case nodes.Pinned:
		return pinned{Kind: "pinned", Node: namedOf(e.Node), Role: string(e.Role), Words: e.Words()}
	case nodes.Moves:
		return moves{Kind: "moves", Role: string(e.Role), From: namedOf(e.From), To: namedOf(e.To), Held: heldOf(e.Held), Words: e.Words()}
	case nodes.WritesOff:
		return writesOff{Kind: "writes-off", Node: namedOf(e.Node), Role: string(e.Role), Held: heldOf(e.Held), Words: e.Words()}
	case nodes.Leaves:
		return leaves{Kind: "leaves", Node: namedOf(e.Node), Words: e.Words()}
	}
	panic(fmt.Sprintf("nodesapi: no answer for the %T effect", e))
}

type roleBoard struct {
	Basis    string      `json:"basis"`
	Nodes    []boardNode `json:"nodes" nullable:"false" doc:"Every Node, in id order"`
	Gestures []gesture   `json:"gestures" nullable:"false" doc:"The plan's verdict on every move and add a drag can make"`
}

type roleBoardOutput struct {
	Body roleBoard
}

type boardNode struct {
	ID         string     `json:"id" format:"uuid"`
	Name       string     `json:"name"`
	Controller bool       `json:"controller"`
	Away       bool       `json:"away" doc:"Out of step, so its checks can't run"`
	Assigned   []string   `json:"assigned" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" nullable:"false" doc:"The Roles ticked on the Node, as a set sends them"`
	Roles      []roleView `json:"roles" nullable:"false" doc:"Every Role the Node stands in, in table order"`
	Probes     []probe    `json:"probes" nullable:"false" doc:"The last verdict of each probe the Node ran"`
	Suggested  []string   `json:"suggested" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" nullable:"false" doc:"What a new Node's Roles step ticks first: each Acquire Role that has no probe or whose probe fits on the Node"`
}

type gesture struct {
	Role    string `json:"role" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving"`
	From    string `json:"from,omitempty" format:"uuid" doc:"The Node the Role moves from; absent for an add"`
	To      string `json:"to" format:"uuid"`
	Verdict string `json:"verdict" enum:"allow,warn,block"`
	Word    string `json:"word"`
	Note    string `json:"note,omitempty" doc:"The words of a Role the gesture hands back or pins"`
}

func boardOf(b nodes.RoleBoard) roleBoard {
	out := roleBoard{Basis: string(b.Basis()), Nodes: []boardNode{}, Gestures: []gesture{}}
	for _, seat := range b.Seats {
		out.Nodes = append(out.Nodes, boardNode{
			ID: seat.ID.String(), Name: seat.Name, Controller: seat.ID == b.Controller, Away: seat.Away,
			Assigned: names(seat.Assigned), Roles: rolesOf(b.Board.Roles(seat.ID, seat.Held)),
			Probes: probesOf(inTableOrder(seat.Probes)), Suggested: names(seat.Suggested()),
		})
	}
	for _, g := range b.Gestures {
		out.Gestures = append(out.Gestures, gesture{
			Role: string(g.Role), To: g.To.String(), Verdict: g.Verdict.String(), Word: g.Word, Note: g.Note,
		})
		if !g.From.IsZero() {
			out.Gestures[len(out.Gestures)-1].From = g.From.String()
		}
	}
	return out
}

// refusal is err from Preview or Apply as the operation answers it: a
// Node with no row, a stale basis, or the code of the first guardrail that
// blocks, each refusal of a plan with its Outcome.
func refusal(err error) error {
	if errors.Is(err, nodes.ErrNotFound) {
		return nodeNotFound
	}
	if e, ok := errors.AsType[*nodes.StaleError](err); ok {
		return rolesStale.With(outcomeOf(e.Outcome))
	}
	if e, ok := errors.AsType[*nodes.BlockedError](err); ok {
		for _, c := range e.Outcome.Checks {
			if c.Verdict == nodes.Block {
				return blockedAs[c.Reason].With(outcomeOf(e.Outcome))
			}
		}
	}
	return err
}

type settings struct {
	JoinCodeLifetime setting.DurationField `json:"joinCodeLifetime"`
	OfflineGrace     setting.DurationField `json:"offlineGrace"`
}

type settingsOutput struct {
	Body settings
}

type settingsInput struct {
	Body settingsPatch
}

type settingsPatch struct {
	JoinCodeLifetime api.Patch[int64] `json:"joinCodeLifetime,omitempty" doc:"Seconds; null resets it to its default"`
	OfflineGrace     api.Patch[int64] `json:"offlineGrace,omitempty" doc:"Seconds; null resets it to its default"`
}

func nodeID(s string) (nodes.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return nodes.ID{}, nodeNotFound
	}
	return nodes.ID{UUID: u}, nil
}

func Routes(r *api.Router, roster Roster) {
	o := r.Owner("nodes", api.Admin)
	h := handlers{roster}

	api.Register(o, api.Admin, api.Op{
		ID:      "createJoinCode",
		Method:  http.MethodPost,
		Path:    "/join-codes",
		Summary: "Mint a Join Code that pairs one Node with this Controller",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{noLinkAddress},
	}, h.createJoinCode)

	api.Register(o, api.Admin, api.Op{
		ID:      "listJoinCodes",
		Method:  http.MethodGet,
		Path:    "/join-codes",
		Summary: "List the Join Codes that may still pair a Node",
		Status:  http.StatusOK,
	}, h.listJoinCodes)

	api.Register(o, api.Admin, api.Op{
		ID:      "getJoinCode",
		Method:  http.MethodGet,
		Path:    "/join-codes/{id}",
		Summary: "Get whether a Join Code is open, used or expired, and its dial",
		Status:  http.StatusOK,
		Errors:  []*api.Code{joinCodeNotFound},
	}, h.getJoinCode)

	api.Register(o, api.Admin, api.Op{
		ID:      "revokeJoinCode",
		Method:  http.MethodDelete,
		Path:    "/join-codes/{id}",
		Summary: "Revoke a Join Code, so no Node pairs with it",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{joinCodeNotFound},
	}, h.revokeJoinCode)

	api.Register(o, api.Admin, api.Op{
		ID:      "dialForCode",
		Method:  http.MethodPost,
		Path:    "/join-codes/{id}/dial",
		Summary: "Dial the Node waiting at a Node address, which pairs with the code over that session",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{joinCodeNotFound, joinCodeClosed, nodeAddressDamaged},
	}, h.dialForCode)

	api.Register(o, api.Admin, api.Op{
		ID:      "setNodeDial",
		Method:  http.MethodPut,
		Path:    "/nodes/{id}/dial",
		Summary: "Set the Node address the Controller dials a Node at, or stop dialing it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nodeAddressDamaged, nodeKeyMismatch},
	}, h.setNodeDial)

	api.Register(o, api.Admin, api.Op{
		ID:      "listNodes",
		Method:  http.MethodGet,
		Path:    "/nodes",
		Summary: "List every Node with its Roles, link and facts",
		Status:  http.StatusOK,
	}, h.listNodes)

	api.Register(o, api.Admin, api.Op{
		ID:      "getNode",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}",
		Summary: "Get one Node with its Roles, link and facts",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound},
	}, h.getNode)

	api.Register(o, api.Admin, api.Op{
		ID:      "renameNode",
		Method:  http.MethodPatch,
		Path:    "/nodes/{id}",
		Summary: "Rename a Node; a Node in step has its new name when this answers",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nameTaken, nameInvalid},
	}, h.renameNode)

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "removeNode",
		Method:  http.MethodDelete,
		Path:    "/nodes/{id}",
		Summary: "Remove a Node: its Roles fall back as if unticked, and its row stays as a tombstone for history",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{nodeNotFound, isController.Code, holdsWork.Code},
	}, h.removeNode)

	api.Register(o, api.Admin, api.Op{
		ID:      "getRoleBoard",
		Method:  http.MethodGet,
		Path:    "/nodes/roles",
		Summary: "Get every Node's Roles and the plan's verdict on every drag",
		Status:  http.StatusOK,
	}, h.getRoleBoard)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTopology",
		Method:  http.MethodGet,
		Path:    "/nodes/topology",
		Summary: "Get every Node, what it holds and the links between them, as the canvas draws them; nodes.traffic carries the rates",
		Status:  http.StatusOK,
	}, h.getTopology)

	api.Register(o, api.Admin, api.Op{
		ID:      "planRoles",
		Method:  http.MethodPost,
		Path:    "/nodes/roles/plan",
		Summary: "Plan a change of Roles without applying it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, rolesStale.Code},
	}, h.planRoles)

	api.Register(o, api.Admin, api.Op{
		ID:      "setRoles",
		Method:  http.MethodPut,
		Path:    "/nodes/roles",
		Summary: "Apply a change of Roles; the Nodes in step it changes have their new Roles when this answers",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, rolesStale.Code, rolesBlocked.Code},
	}, h.setRoles)

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "writeOff",
		Method:  http.MethodPost,
		Path:    "/nodes/{id}/write-offs",
		Summary: "Give up the work a Node holds for Roles, live or offline; its files are kept",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nothingHeld.Code, keepsRole.Code},
	}, h.writeOff)

	api.Register(o, api.Admin, api.Op{
		ID:      "getNodeSettings",
		Method:  http.MethodGet,
		Path:    "/nodes/settings",
		Summary: "Get the nodes settings",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
		return currentSettings(ctx, open(roster))
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "updateNodeSettings",
		Method:  http.MethodPatch,
		Path:    "/nodes/settings",
		Summary: "Change the nodes settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, h.updateNodeSettings)
}

type handlers struct{ roster Roster }

func (h handlers) listNodes(ctx context.Context, _ identity.Admin, _ *struct{}) (*listOutput, error) {
	list, err := open(h.roster).List(ctx)
	if err != nil {
		return nil, err
	}
	out := nodeList{Nodes: []node{}}
	for _, n := range list {
		out.Nodes = append(out.Nodes, nodeOf(n))
	}
	return &listOutput{Body: out}, nil
}

func (h handlers) getNode(ctx context.Context, _ identity.Admin, in *idInput) (*nodeOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	n, err := open(h.roster).Get(ctx, id)
	if errors.Is(err, nodes.ErrNotFound) {
		return nil, nodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &nodeOutput{Body: nodeOf(n)}, nil
}

func (h handlers) renameNode(ctx context.Context, _ identity.Admin, in *renameInput) (*nodeOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	n, err := open(h.roster).Rename(ctx, id, in.Body.Name)
	switch {
	case errors.Is(err, nodes.ErrNotFound):
		return nil, nodeNotFound
	case errors.Is(err, nodes.ErrNameTaken):
		return nil, nameTaken
	case errors.Is(err, nodes.ErrNameInvalid):
		return nil, nameInvalid
	case err != nil:
		return nil, err
	}
	return &nodeOutput{Body: nodeOf(n)}, nil
}

func (h handlers) removeNode(ctx context.Context, _ identity.Admin, in *removeInput) (*struct{}, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	if _, err := open(h.roster).Apply(ctx, nodes.Remove{Node: id, WriteOff: in.WriteOff}); err != nil {
		return nil, refusal(err)
	}
	return &struct{}{}, nil
}

func (h handlers) getRoleBoard(ctx context.Context, _ identity.Admin, _ *struct{}) (*roleBoardOutput, error) {
	b, err := open(h.roster).RoleBoard(ctx)
	if err != nil {
		return nil, err
	}
	return &roleBoardOutput{Body: boardOf(b)}, nil
}

func (h handlers) planRoles(ctx context.Context, _ identity.Admin, in *planInput) (*outcomeOutput, error) {
	c, err := rolesChange{Basis: in.Body.Basis, Set: in.Body.Set}.change()
	if err != nil {
		return nil, err
	}
	out, err := open(h.roster).Preview(ctx, c, in.Body.Test)
	if err != nil {
		return nil, refusal(err)
	}
	return &outcomeOutput{Body: outcomeOf(out)}, nil
}

func (h handlers) setRoles(ctx context.Context, _ identity.Admin, in *rolesInput) (*outcomeOutput, error) {
	c, err := in.Body.change()
	if err != nil {
		return nil, err
	}
	out, err := open(h.roster).Apply(ctx, c)
	if err != nil {
		return nil, refusal(err)
	}
	return &outcomeOutput{Body: outcomeOf(out)}, nil
}

func (h handlers) writeOff(ctx context.Context, _ identity.Admin, in *writeOffInput) (*outcomeOutput, error) {
	c, err := in.change()
	if err != nil {
		return nil, err
	}
	out, err := open(h.roster).Apply(ctx, c)
	if err != nil {
		return nil, refusal(err)
	}
	return &outcomeOutput{Body: outcomeOf(out)}, nil
}

func (h handlers) updateNodeSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	s := open(h.roster)
	lifetime, err := parsed(in.Body.JoinCodeLifetime, nodes.JoinCodeLifetime)
	if err != nil {
		return nil, err
	}
	grace, err := parsed(in.Body.OfflineGrace, nodes.OfflineGrace)
	if err != nil {
		return nil, err
	}
	if lifetime != nil {
		if err := s.SetJoinCodeLifetime(ctx, *lifetime); err != nil {
			return nil, err
		}
	}
	if grace != nil {
		if err := s.SetOfflineGrace(ctx, *grace); err != nil {
			return nil, err
		}
	}
	return currentSettings(ctx, s)
}

// parsed is the value p sends for d, nil when it sends none.
func parsed(p api.Patch[int64], d setting.Duration) (*setting.Stored[int64], error) {
	if !p.Sent() {
		return nil, nil
	}
	v, err := d.Parse(p.Value())
	if err != nil {
		return nil, settingapi.Refused(err)
	}
	return &v, nil
}

func currentSettings(ctx context.Context, s *nodes.Service) (*settingsOutput, error) {
	lifetime, err := s.JoinCodeLifetimeField(ctx)
	if err != nil {
		return nil, err
	}
	grace, err := s.OfflineGraceField(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{JoinCodeLifetime: lifetime, OfflineGrace: grace}}, nil
}
