package setup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

// Member is what the setup steps made of a Node: its id and Roles, both
// zero for an Unclaimed Node.
type Member struct {
	ID    string
	Roles []string
}

// Claims is what the setup steps made of a world: the Controller's API key
// and each Node's Member, in Scenario order.
type Claims struct {
	APIKey  string
	Members []Member
}

// Steps sets up every ready Node as its Scenario Node says, over c, ready
// holding each Node's readiness in Scenario order. 9a claims the
// controller Node with the Scenario's Admin. 9b joins each join Node in
// order through its own Setup code with a Join Code the Controller mints.
// Then one set gives every Node that lists Roles its Roles. 9c fails
// unless each Node the set names is linked and has applied its new
// Roles. Nothing polls: claimWorker answers once the link is live and
// setRoles once the Nodes in step it changed hold their Roles.
func Steps(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready) (Claims, error) {
	claims := Claims{Members: make([]Member, len(s.Nodes))}
	var (
		controller string
		set        []nodeapi.Assignment
	)
	for i, n := range s.Nodes {
		var err error
		switch n.Setup {
		case scenario.SetupController:
			controller = url(ready[i])
			claims.APIKey, claims.Members[i].ID, err = claimController(ctx, c, s, ready[i])
		case scenario.SetupJoin:
			claims.Members[i].ID, err = join(ctx, c, controller, claims.APIKey, ready[i])
		case scenario.SetupNone:
		}
		if err != nil {
			return Claims{}, fmt.Errorf("node %s: %w", n.Name, err)
		}
		if len(n.Roles) > 0 {
			set = append(set, nodeapi.Assignment{NodeID: claims.Members[i].ID, Roles: names(n.Roles)})
		}
	}
	if controller == "" {
		return claims, nil
	}
	if len(set) > 0 {
		if err := nodeapi.SetRoles(ctx, c, controller, claims.APIKey, set); err != nil {
			return Claims{}, err
		}
	}
	return claims, members(ctx, c, s, controller, claims, set)
}

func claimController(ctx context.Context, c *http.Client, s scenario.Scenario, ready node.Ready) (apiKey, id string, err error) {
	base := url(ready)
	admin := s.Admin
	apiKey, err = nodeapi.ClaimController(ctx, c, base, ready.SetupCode, nodeapi.Admin{Email: admin.Email, DisplayName: admin.DisplayName, PIN: admin.PIN})
	if err != nil {
		return "", "", err
	}
	rows, err := nodeapi.ListNodes(ctx, c, base, apiKey)
	if err != nil {
		return "", "", err
	}
	for _, r := range rows {
		if r.Self {
			return apiKey, r.ID, nil
		}
	}
	return "", "", errors.New("listNodes has no row for the Controller itself")
}

func join(ctx context.Context, c *http.Client, controller, apiKey string, ready node.Ready) (string, error) {
	code, err := nodeapi.CreateJoinCode(ctx, c, controller, apiKey)
	if err != nil {
		return "", err
	}
	return nodeapi.ClaimWorker(ctx, c, url(ready), ready.SetupCode, code)
}

// members fills each set-up Member's Roles from listNodes, checked as 9c
// says: each Node set names shows its link and the applied node-scope
// version equal to the committed one.
func members(ctx context.Context, c *http.Client, s scenario.Scenario, controller string, claims Claims, set []nodeapi.Assignment) error {
	rows, err := nodeapi.ListNodes(ctx, c, controller, claims.APIKey)
	if err != nil {
		return err
	}
	for i, n := range s.Nodes {
		m := &claims.Members[i]
		if n.Setup == scenario.SetupNone {
			continue
		}
		at := slices.IndexFunc(rows, func(r nodeapi.Node) bool { return r.ID == m.ID })
		if at < 0 {
			return fmt.Errorf("node %s: listNodes has no row %s", n.Name, m.ID)
		}
		row := rows[at]
		m.Roles = nodeapi.Assigned(row.Roles)
		if !slices.ContainsFunc(set, func(a nodeapi.Assignment) bool { return a.NodeID == m.ID }) {
			continue
		}
		link := "up"
		if n.Setup == scenario.SetupController {
			link = "local"
		}
		if row.Link.State != link || row.Facts == nil || row.Facts.AppliedVersion != row.IntentVersion {
			return fmt.Errorf("node %s: after setRoles listNodes shows %+v, want link %s and its Roles applied", n.Name, row, link)
		}
	}
	return nil
}

func names(roles []role.Name) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}

func url(ready node.Ready) string { return "http://" + ready.Listeners.HTTP.String() }
