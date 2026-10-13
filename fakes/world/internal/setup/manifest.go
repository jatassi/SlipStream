package setup

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

type Node struct {
	Name       string            `json:"name"`
	Setup      scenario.Setup    `json:"setup"`
	Claim      Claim             `json:"claim"`
	Address    string            `json:"address"`
	Host       string            `json:"host"`
	URL        string            `json:"url"`
	BrowserURL string            `json:"browserUrl"`
	Listeners  map[string]string `json:"listeners"`
	DataDir    string            `json:"dataDir"`
	PID        int               `json:"pid"`
	Seed       string            `json:"seed"`
	Log        string            `json:"log"`
	Overrides  []node.Key        `json:"overrides"`
	// NodeKey is node.key's fingerprint. NodeID and Roles are the Node's
	// row on the Controller, absent while it is Unclaimed.
	NodeKey string   `json:"nodeKey"`
	NodeID  string   `json:"nodeId,omitempty"`
	Roles   []string `json:"roles,omitempty"`
}

// Claim is what a Node is once the world has set it up.
type Claim string

const (
	ClaimUnclaimed  Claim = "unclaimed"
	ClaimController Claim = "controller"
	ClaimWorker     Claim = "worker"
)

var claimAfter = [...]Claim{scenario.SetupNone: ClaimUnclaimed, scenario.SetupController: ClaimController, scenario.SetupJoin: ClaimWorker}

// Ran is one ready Node as its world ran it.
type Ran struct {
	Spec    scenario.Node
	Address netip.Addr
	Ready   node.Ready
	DataDir string
	Log     string
}

// Entry is n's manifest entry, as the setup steps left it in m.
func Entry(n Ran, m Member) (Node, error) {
	listeners, err := Listeners(n.Ready.Listeners)
	if err != nil {
		return Node{}, err
	}
	host := n.Spec.Name + ".localhost"
	return Node{
		Name:       n.Spec.Name,
		Setup:      n.Spec.Setup,
		Claim:      claimAfter[n.Spec.Setup],
		Address:    n.Address.String(),
		Host:       host,
		URL:        url(n.Ready),
		BrowserURL: fmt.Sprintf("http://%s:%d", host, n.Ready.Listeners.HTTP.Port()),
		Listeners:  listeners,
		DataDir:    n.DataDir,
		PID:        n.Ready.PID,
		Seed:       n.Ready.Seed,
		Log:        n.Log,
		Overrides:  append([]node.Key{}, n.Ready.Overrides...),
		NodeKey:    n.Ready.NodeKey,
		NodeID:     m.ID,
		Roles:      m.Roles,
	}, nil
}

func Listeners(l node.Listeners) (map[string]string, error) {
	data, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("listeners: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("listeners: %w", err)
	}
	return m, nil
}
