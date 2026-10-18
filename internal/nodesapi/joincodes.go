package nodesapi

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	joinCodeNotFound   = api.NewCode(http.StatusNotFound, "join_code_not_found", "No Join Code has this id")
	joinCodeClosed     = api.NewCode(http.StatusConflict, "join_code_closed", "The Join Code is used or expired")
	nodeAddressDamaged = api.NewCode(http.StatusUnprocessableEntity, "node_address_damaged", "The Node address is mistyped, cut short or not a Node address")
	nodeKeyMismatch    = api.NewCode(http.StatusConflict, "node_key_mismatch", "The Node address names another Node")
)

type joinCode struct {
	ID        string    `json:"id" format:"uuid"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type joinCodeOutput struct {
	Body joinCode
}

type openCode struct {
	ID        string    `json:"id" format:"uuid"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type openCodes struct {
	Codes []openCode `json:"codes" nullable:"false" doc:"Every Join Code that may still pair a Node, oldest first"`
}

type openCodesOutput struct {
	Body openCodes
}

type codeState struct {
	State   string   `json:"state" enum:"open,used,expired"`
	NodeID  string   `json:"nodeId,omitempty" format:"uuid" doc:"The Node that paired with the code; absent unless used"`
	Dialing *dialing `json:"dialing,omitempty" doc:"The Node address the Controller dials for an open code; absent without one"`
}

type dialing struct {
	Address     string    `json:"address"`
	Addresses   []string  `json:"addresses" nullable:"false" doc:"Where the Controller dials the Node, in order"`
	Since       time.Time `json:"since"`
	Unreachable bool      `json:"unreachable" doc:"The Node hasn't paired within timings.nodes.dial_report of the paste"`
}

type codeStateOutput struct {
	Body codeState
}

type codeInput struct {
	ID string `path:"id" format:"uuid"`
}

type dialInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		NodeAddress string `json:"nodeAddress" doc:"The SLN1 text the waiting Node printed"`
	}
}

type nodeDialInput struct {
	ID   string `path:"id" format:"uuid"`
	Body nodeDial
}

type nodeDial struct {
	NodeAddress *string   `json:"nodeAddress,omitempty" doc:"Dial the Node at this Node address from now on"`
	Addresses   *[]string `json:"addresses,omitempty" maxItems:"0" doc:"Send [] to stop dialing the Node, which then dials the Controller"`
}

type dialView struct {
	By              string   `json:"by,omitempty" enum:"node,controller" doc:"The side that dialed the live session; absent with none"`
	ControllerDials []string `json:"controllerDials" nullable:"false" doc:"The addresses the Controller dials the Node at; empty when the Node dials the Controller"`
}

func dialOf(d nodes.Dial) dialView {
	out := dialView{ControllerDials: addressesOf(d.ControllerDials)}
	if d.By != nodes.NoSession {
		out.By = d.By.String()
	}
	return out
}

func addressesOf(as []netip.AddrPort) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.String())
	}
	return out
}

func joinCodeID(s string) (nodes.JoinCodeID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return nodes.JoinCodeID{}, joinCodeNotFound
	}
	return nodes.JoinCodeID{UUID: u}, nil
}

// codeRefusal is err from a Join Code call as the operation answers it.
func codeRefusal(err error) error {
	switch {
	case errors.Is(err, nodes.ErrJoinCodeNotFound):
		return joinCodeNotFound
	case errors.Is(err, nodes.ErrJoinCodeClosed):
		return joinCodeClosed
	case errors.Is(err, nodes.ErrAddressUnusable):
		return nodeAddressDamaged
	case errors.Is(err, nodes.ErrNotFound):
		return nodeNotFound
	case errors.Is(err, nodes.ErrNodeKeyMismatch):
		return nodeKeyMismatch
	}
	return err
}

func parseNodeAddress(field, s string) (joincode.NodeAddress, error) {
	a, err := joincode.ParseNodeAddress(s)
	if err != nil {
		return joincode.NodeAddress{}, nodeAddressDamaged.On(field, err.Error())
	}
	return a, nil
}

func (h handlers) createJoinCode(ctx context.Context, _ identity.Admin, _ *struct{}) (*joinCodeOutput, error) {
	code, err := open(h.roster).MintJoinCode(ctx)
	if errors.Is(err, nodes.ErrNoLinkAddress) {
		return nil, noLinkAddress
	}
	if err != nil {
		return nil, err
	}
	return &joinCodeOutput{Body: joinCode{ID: code.ID.String(), Code: code.Code.String(), ExpiresAt: code.ExpiresAt}}, nil
}

func (h handlers) listJoinCodes(ctx context.Context, _ identity.Admin, _ *struct{}) (*openCodesOutput, error) {
	list, err := open(h.roster).JoinCodes(ctx)
	if err != nil {
		return nil, err
	}
	out := openCodes{Codes: []openCode{}}
	for _, c := range list {
		out.Codes = append(out.Codes, openCode{ID: c.ID.String(), CreatedAt: c.CreatedAt, ExpiresAt: c.ExpiresAt})
	}
	return &openCodesOutput{Body: out}, nil
}

func (h handlers) getJoinCode(ctx context.Context, _ identity.Admin, in *codeInput) (*codeStateOutput, error) {
	id, err := joinCodeID(in.ID)
	if err != nil {
		return nil, err
	}
	st, err := open(h.roster).JoinCodeState(ctx, id)
	if err != nil {
		return nil, codeRefusal(err)
	}
	out := codeState{State: st.State.String()}
	if !st.Node.IsZero() {
		out.NodeID = st.Node.String()
	}
	if d := st.Dialing; d != nil {
		out.Dialing = &dialing{Address: d.Address.String(), Addresses: addressesOf(d.Address.Addresses()), Since: d.Since, Unreachable: d.Unreachable}
	}
	return &codeStateOutput{Body: out}, nil
}

func (h handlers) revokeJoinCode(ctx context.Context, _ identity.Admin, in *codeInput) (*struct{}, error) {
	id, err := joinCodeID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := open(h.roster).RevokeJoinCode(ctx, id); err != nil {
		return nil, codeRefusal(err)
	}
	return &struct{}{}, nil
}

func (h handlers) dialForCode(ctx context.Context, _ identity.Admin, in *dialInput) (*struct{}, error) {
	id, err := joinCodeID(in.ID)
	if err != nil {
		return nil, err
	}
	addr, err := parseNodeAddress("nodeAddress", in.Body.NodeAddress)
	if err != nil {
		return nil, err
	}
	if err := open(h.roster).DialForCode(ctx, id, addr); err != nil {
		return nil, codeRefusal(err)
	}
	return &struct{}{}, nil
}

func (h handlers) setNodeDial(ctx context.Context, _ identity.Admin, in *nodeDialInput) (*nodeOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	var addr *joincode.NodeAddress
	switch b := in.Body; {
	case b.NodeAddress != nil && b.Addresses == nil:
		a, err := parseNodeAddress("nodeAddress", *b.NodeAddress)
		if err != nil {
			return nil, err
		}
		addr = &a
	case b.NodeAddress == nil && b.Addresses != nil:
	default:
		return nil, api.Invalid.On("nodeAddress", "Send either nodeAddress or addresses: [].")
	}
	s := open(h.roster)
	if err := s.SetDial(ctx, id, addr); err != nil {
		return nil, codeRefusal(err)
	}
	n, err := s.Get(ctx, id)
	if err != nil {
		return nil, refusal(err)
	}
	return &nodeOutput{Body: nodeOf(n)}, nil
}
