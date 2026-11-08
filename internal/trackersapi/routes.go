package trackersapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
)

var (
	routeNotAHome      = api.NewCode(http.StatusConflict, "route_not_a_home", "No Indexer on that Node reaches this Tracker")
	nodeNotFound       = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	clientNotFound     = api.NewCode(http.StatusNotFound, "download_client_not_found", "No Download Client has this id")
	clientRefused      = api.NewCode(http.StatusUnprocessableEntity, "download_client_field_refused", "SlipStream can't use a value in this client's settings")
	nodeNotDownloading = api.NewCode(http.StatusConflict, "node_not_downloading", "That Node doesn't hold Downloading")
)

// target is a Download Client: a Node's built-in client, or an external
// client on it.
type target struct {
	NodeID   string  `json:"nodeId" format:"uuid"`
	ClientID *string `json:"clientId,omitempty" format:"uuid" doc:"The external client; absent for the Node's built-in client"`
}

func targetOf(t *downloadclients.Target) api.Nullable[target] {
	if t == nil {
		return api.Nullable[target]{}
	}
	out := target{NodeID: t.Node.String()}
	if t.Client != nil {
		id := t.Client.String()
		out.ClientID = &id
	}
	return api.Some(out)
}

type choice struct {
	ClientID  *string `json:"clientId,omitempty" format:"uuid" doc:"An external client; set this or builtInOn"`
	BuiltInOn *string `json:"builtInOn,omitempty" format:"uuid" doc:"A Node, for its built-in client; set this or clientId"`
}

func choiceShape(c *downloadclients.Choice) api.Nullable[choice] {
	if c == nil {
		return api.Nullable[choice]{}
	}
	if c.Client != nil {
		id := c.Client.String()
		return api.Some(choice{ClientID: &id})
	}
	on := c.BuiltInOn.String()
	return api.Some(choice{BuiltInOn: &on})
}

type route struct {
	HomeNodeID string               `json:"homeNodeId" format:"uuid" doc:"The home Node of the Indexers that find the Grabs"`
	To         api.Nullable[target] `json:"to" doc:"Where the Grabs download now: the stored client while it can take them, else the default; null while they download where the Work's library root is"`
	Default    api.Nullable[target] `json:"default" doc:"The home Node's built-in client; null while the home Node doesn't hold Downloading"`
	Stored     api.Nullable[choice] `json:"stored" doc:"The client an Admin picked; null for none"`
	Skipped    bool                 `json:"skipped" doc:"True while the stored client can't take a Grab because it is off or deleted, or its Node doesn't hold Downloading"`
}

type routeList struct {
	Routes []route `json:"routes" nullable:"false" doc:"One for each home Node of the Tracker's Indexers, by Node name; empty unless the Tracker is private"`
}

type routesOutput struct{ Body routeList }

func routesOf(rs []trackers.Route) *routesOutput {
	out := routeList{Routes: make([]route, len(rs))}
	for i, r := range rs {
		out.Routes[i] = route{
			HomeNodeID: r.Home.String(), To: targetOf(r.To), Default: targetOf(r.Default),
			Stored: choiceShape(r.Stored), Skipped: r.Skipped,
		}
	}
	return &routesOutput{Body: out}
}

func (h handlers) listRoutes(ctx context.Context, by identity.Admin, in *trackerInput) (*routesOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	rs, err := h.trackers().Routes(ctx, by, id)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	return routesOf(rs), nil
}

type routeInput struct {
	ID     string `path:"id" format:"uuid"`
	NodeID string `path:"nodeId" format:"uuid" doc:"The home Node"`
}

type setRouteInput struct {
	ID     string `path:"id" format:"uuid"`
	NodeID string `path:"nodeId" format:"uuid" doc:"The home Node"`
	Body   choice
}

func (h handlers) setRoute(ctx context.Context, by identity.Admin, in *setRouteInput) (*routesOutput, error) {
	id, home, err := parseRoute(in.ID, in.NodeID)
	if err != nil {
		return nil, err
	}
	c, err := choiceOf(in.Body)
	if err != nil {
		return nil, err
	}
	rs, err := h.trackers().SetRoute(ctx, by, id, home, c)
	if err != nil {
		return nil, routeAnswer(err)
	}
	return routesOf(rs), nil
}

func (h handlers) clearRoute(ctx context.Context, by identity.Admin, in *routeInput) (*routesOutput, error) {
	id, home, err := parseRoute(in.ID, in.NodeID)
	if err != nil {
		return nil, err
	}
	rs, err := h.trackers().ClearRoute(ctx, by, id, home)
	if err != nil {
		return nil, routeAnswer(err)
	}
	return routesOf(rs), nil
}

func parseRoute(rawID, rawNode string) (indexers.TrackerID, role.NodeID, error) {
	id, err := parseTrackerID(rawID)
	if err != nil {
		return indexers.TrackerID{}, role.NodeID{}, err
	}
	node, err := storage.ParseUUID(rawNode)
	if err != nil {
		return indexers.TrackerID{}, role.NodeID{}, nodeNotFound
	}
	return id, role.NodeID{UUID: node}, nil
}

// choiceOf is the client a route's body names: exactly one of an external
// client's id or a Node for its built-in client.
func choiceOf(c choice) (downloadclients.Choice, error) {
	switch {
	case (c.ClientID == nil) == (c.BuiltInOn == nil):
		return downloadclients.Choice{}, api.Invalid.Fields(api.FieldError{Location: "body", Message: "Send exactly one of clientId and builtInOn."})
	case c.ClientID != nil:
		u, err := storage.ParseUUID(*c.ClientID)
		if err != nil {
			return downloadclients.Choice{}, clientNotFound
		}
		return downloadclients.Choice{Client: &downloadclients.ClientID{UUID: u}}, nil
	}
	u, err := storage.ParseUUID(*c.BuiltInOn)
	if err != nil {
		return downloadclients.Choice{}, nodeNotDownloading
	}
	return downloadclients.Choice{BuiltInOn: role.NodeID{UUID: u}}, nil
}

func routeAnswer(err error) error {
	if public, ok := errors.AsType[*trackers.PublicError](err); ok {
		return trackerPublic.Because(public.Name + " is public. Only a private Tracker's Grabs can go to a client you pick.")
	}
	if notHome, ok := errors.AsType[*trackers.NotAHomeError](err); ok {
		return routeNotAHome.Because("No Indexer on " + notHome.Node + " reaches " + notHome.Tracker + ". Pick the home Node of one of its Indexers.")
	}
	if off, ok := errors.AsType[*downloadclients.ClientOffError](err); ok {
		return clientRefused.On("clientId", off.Name+" is off. Turn it on, then pick it again.")
	}
	switch {
	case errors.Is(err, trackers.ErrNoNode):
		return nodeNotFound
	case errors.Is(err, downloadclients.ErrNoClient):
		return clientNotFound
	case errors.Is(err, downloadclients.ErrNotDownloading):
		return nodeNotDownloading
	}
	return trackerAnswer(err)
}
