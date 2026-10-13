package setup

import (
	"context"
	"fmt"
	"net/http"

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
// controller Node with the Scenario's Admin and sets its Roles if it has
// any. 9b joins each join Node in order through its own Setup code with a
// Join Code the Controller mints, then sets its Roles. 9c fails unless
// every setNodeRoles answer shows the Node linked and holding its new
// Roles. Nothing polls: claimWorker answers once the link is live and
// setNodeRoles once a Node in step holds the Roles.
func Steps(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready) (Claims, error) {
	claims := Claims{Members: make([]Member, len(s.Nodes))}
	var controller string
	for i, n := range s.Nodes {
		var err error
		switch n.Setup {
		case scenario.SetupController:
			controller = url(ready[i])
			claims.APIKey, claims.Members[i], err = claimController(ctx, c, s, n, ready[i])
		case scenario.SetupJoin:
			claims.Members[i], err = join(ctx, c, controller, claims.APIKey, n, ready[i])
		case scenario.SetupNone:
		}
		if err != nil {
			return Claims{}, fmt.Errorf("node %s: %w", n.Name, err)
		}
	}
	return claims, nil
}

func claimController(ctx context.Context, c *http.Client, s scenario.Scenario, n scenario.Node, ready node.Ready) (string, Member, error) {
	base := url(ready)
	admin := s.Admin
	apiKey, err := nodeapi.ClaimController(ctx, c, base, ready.SetupCode, nodeapi.Admin{Email: admin.Email, DisplayName: admin.DisplayName, PIN: admin.PIN})
	if err != nil {
		return "", Member{}, err
	}
	rows, err := nodeapi.ListNodes(ctx, c, base, apiKey)
	if err != nil {
		return "", Member{}, err
	}
	var self nodeapi.Node
	for _, r := range rows {
		if r.Self {
			self = r
		}
	}
	if len(n.Roles) == 0 {
		return apiKey, Member{ID: self.ID, Roles: self.Roles}, nil
	}
	m, err := setRoles(ctx, c, base, apiKey, self.ID, n.Roles, "local")
	return apiKey, m, err
}

func join(ctx context.Context, c *http.Client, controller, apiKey string, n scenario.Node, ready node.Ready) (Member, error) {
	code, err := nodeapi.CreateJoinCode(ctx, c, controller, apiKey)
	if err != nil {
		return Member{}, err
	}
	id, err := nodeapi.ClaimWorker(ctx, c, url(ready), ready.SetupCode, code)
	if err != nil {
		return Member{}, err
	}
	return setRoles(ctx, c, controller, apiKey, id, n.Roles, "up")
}

// setRoles is setNodeRoles checked as 9c says: the answer shows the link
// in state link and the applied node-scope version equal to the committed
// one.
func setRoles(ctx context.Context, c *http.Client, controller, apiKey, id string, roles []role.Name, link string) (Member, error) {
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = string(r)
	}
	row, err := nodeapi.SetNodeRoles(ctx, c, controller, apiKey, id, names)
	if err != nil {
		return Member{}, err
	}
	if row.Link.State != link || row.Facts == nil || row.Facts.AppliedVersion != row.IntentVersion {
		return Member{}, fmt.Errorf("setNodeRoles did not answer with link %s and the Roles applied: %s", link, row.Body)
	}
	return Member{ID: row.ID, Roles: row.Roles}, nil
}

// url is the base URL of a ready Node's HTTP listener.
func url(ready node.Ready) string { return "http://" + ready.Listeners.HTTP.String() }
