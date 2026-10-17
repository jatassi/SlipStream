// Package nodesapi holds the nodes owner's operations: Join Codes, the
// Nodes an Admin sees, their Roles and the nodes settings.
package nodesapi

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
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
)

type joinCode struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type joinCodeOutput struct {
	Body joinCode
}

type node struct {
	ID        string    `json:"id" format:"uuid"`
	Name      string    `json:"name"`
	Key       string    `json:"key" doc:"The key's fingerprint"`
	Self      bool      `json:"self"`
	PairedAt  time.Time `json:"pairedAt"`
	Addresses []string  `json:"addresses" nullable:"false"`
	Roles     []string  `json:"roles" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" nullable:"false"`
	// IntentVersion is the node-scope version the library derives now.
	IntentVersion string `json:"intentVersion" pattern:"^[0-9a-f]{32}$"`
	Link          link   `json:"link"`
	Facts         *facts `json:"facts,omitempty" doc:"Absent while the Node has no session"`
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
		Roles:         []string{},
		IntentVersion: hex.EncodeToString(n.Version),
		Link:          link{State: n.Link.String(), Since: n.Since},
	}
	for _, a := range n.Addresses.All() {
		out.Addresses = append(out.Addresses, a.String())
	}
	for _, r := range n.Roles {
		out.Roles = append(out.Roles, string(r))
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
	ID   string `path:"id" format:"uuid"`
	Body roles
}

type roles struct {
	Roles []string `json:"roles" enum:"searching,downloading,storage,converting,subtitles,metadata,media-server-link,notifying,relaying,serving" nullable:"false"`
}

type settings struct {
	JoinCodeLifetime setting.DurationField `json:"joinCodeLifetime"`
}

type settingsOutput struct {
	Body settings
}

type settingsInput struct {
	Body settingsPatch
}

type settingsPatch struct {
	JoinCodeLifetime api.Patch[int64] `json:"joinCodeLifetime,omitempty" doc:"Seconds; null resets it to its default"`
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
		Path:    "/nodes/join-codes",
		Summary: "Mint a Join Code that pairs one Node with this Controller",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{noLinkAddress},
	}, h.createJoinCode)

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
		ID:      "setNodeRoles",
		Method:  http.MethodPut,
		Path:    "/nodes/{id}/roles",
		Summary: "Set the Roles a Node holds; a connected Node in step has them when this answers",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound},
	}, h.setNodeRoles)

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

func (h handlers) createJoinCode(ctx context.Context, _ identity.Admin, _ *struct{}) (*joinCodeOutput, error) {
	code, err := open(h.roster).MintJoinCode(ctx)
	if errors.Is(err, nodes.ErrNoLinkAddress) {
		return nil, noLinkAddress
	}
	if err != nil {
		return nil, err
	}
	return &joinCodeOutput{Body: joinCode{Code: code.Code.String(), ExpiresAt: code.ExpiresAt}}, nil
}

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

func (h handlers) setNodeRoles(ctx context.Context, _ identity.Admin, in *rolesInput) (*nodeOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	names := make([]role.Name, len(in.Body.Roles))
	for i, s := range in.Body.Roles {
		if names[i], err = role.ParseName(s); err != nil {
			return nil, api.Invalid
		}
	}
	n, err := open(h.roster).SetRoles(ctx, id, names)
	if errors.Is(err, nodes.ErrNotFound) {
		return nil, nodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &nodeOutput{Body: nodeOf(n)}, nil
}

func (h handlers) updateNodeSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	s := open(h.roster)
	if p := in.Body.JoinCodeLifetime; p.Sent() {
		v, err := nodes.JoinCodeLifetime.Parse(p.Value())
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		if err := s.SetJoinCodeLifetime(ctx, v); err != nil {
			return nil, err
		}
	}
	return currentSettings(ctx, s)
}

func currentSettings(ctx context.Context, s *nodes.Service) (*settingsOutput, error) {
	lifetime, err := s.JoinCodeLifetimeField(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{JoinCodeLifetime: lifetime}}, nil
}
