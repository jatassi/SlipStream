package setup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
)

// MediaWorld is what the world gives the Media Server step: the URL of
// the front for a Node's reach of the fake of kind, and the approval of a
// plex.tv PIN by its code, as the Admin gives it on plex.tv's page.
type MediaWorld struct {
	Front   func(node, kind string) string
	Approve func(ctx context.Context, code string) error
}

var versions = map[string][]string{"plex": plex.Versions, "jellyfin": jellyfin.Versions, "emby": emby.Versions}

// MediaServers adds each Scenario Media Server as an Admin would, with
// its reaching Node's front as its address and the Controller's front as
// its fallback: Plex through the plex.tv PIN the world approves, Jellyfin
// and Emby with the fake's key or its administrator's sign-in. Then it
// saves the Path mappings of the roots the server holds, and every other
// root as not shown.
func MediaServers(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, mediaDir string, w MediaWorld) error {
	controller := controllerURL(s, ready)
	var roots []nodeapi.Root
	for _, kind := range scenario.MediaServerKinds {
		p, ok, err := serverfake.Playing(s, kind, versions[kind])
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if roots == nil {
			if roots, err = nodeapi.ListRoots(ctx, c, controller, claims.APIKey); err != nil {
				return err
			}
		}
		id, err := addMediaServer(ctx, c, s, claims, controller, p, w)
		if err != nil {
			return fmt.Errorf("media server %s: %w", p.Name, err)
		}
		if err := saveMappings(ctx, c, s, claims, controller, id, p, roots, mediaDir); err != nil {
			return fmt.Errorf("media server %s: %w", p.Name, err)
		}
	}
	return nil
}

func addMediaServer(ctx context.Context, c *http.Client, s scenario.Scenario, claims Claims, controller string, p serverfake.Played, w MediaWorld) (string, error) {
	controllerNode := s.Nodes[slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Setup == scenario.SetupController })].Name
	reaching := claims.Members[slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Name == p.Node })].ID
	body := nodeapi.NewMediaServer{Kind: p.Kind, Name: p.Name, Address: w.Front(p.Node, p.Kind), ReachingNodeID: reaching}
	if !p.NoFallback && p.Node != controllerNode {
		body.FallbackAddress = w.Front(controllerNode, p.Kind)
	}
	switch {
	case p.Kind == "plex":
		identifier, err := signInToPlex(ctx, c, controller, claims.APIKey, p.Name, w)
		if err != nil {
			return "", err
		}
		body.PlexServer = identifier
	case p.SignIn == "admin":
		signed, err := nodeapi.SignInToMediaServer(ctx, c, controller, claims.APIKey, p.Kind, body.Address, reaching, serverfake.Admin, serverfake.AdminPassword)
		if err != nil {
			return "", err
		}
		body.APIKey, body.DeviceID = signed.APIKey, signed.DeviceID
	default:
		body.APIKey = p.Key()
	}
	return nodeapi.AddMediaServer(ctx, c, controller, claims.APIKey, body)
}

// plexSignInWait is how long the Controller has to see the approved PIN.
const plexSignInWait = 30 * time.Second

var errPlexSignIn = errors.New("the Plex sign-in never read signed in")

// signInToPlex signs the Controller in to plex.tv, approving the PIN as
// the Admin would, and answers the machineIdentifier of the account's
// server named name.
func signInToPlex(ctx context.Context, c *http.Client, controller, apiKey, name string, w MediaWorld) (string, error) {
	pin, err := nodeapi.StartPlexSignIn(ctx, c, controller, apiKey, controller+"/settings/system/media-servers")
	if err != nil {
		return "", err
	}
	_, fragment, _ := strings.Cut(pin.AuthURL, "#?")
	q, err := neturl.ParseQuery(fragment)
	if err != nil {
		return "", fmt.Errorf("the Plex sign-in page %s: %w", pin.AuthURL, err)
	}
	if err := w.Approve(ctx, q.Get("code")); err != nil {
		return "", err
	}
	for deadline := time.Now().Add(plexSignInWait); ; {
		state, err := nodeapi.PlexSignInState(ctx, c, controller, apiKey, pin.ID)
		if err != nil {
			return "", err
		}
		if state == "signed_in" {
			break
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("%w: it reads %s", errPlexSignIn, state)
		}
		time.Sleep(250 * time.Millisecond)
	}
	servers, err := nodeapi.ListPlexServers(ctx, c, controller, apiKey)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(servers, func(srv nodeapi.PlexServer) bool { return srv.Name == name })
	if i < 0 {
		return "", fmt.Errorf("the Plex account lists no server named %s", name)
	}
	return servers[i].MachineIdentifier, nil
}

func saveMappings(ctx context.Context, c *http.Client, s scenario.Scenario, claims Claims, controller, id string, p serverfake.Played, roots []nodeapi.Root, mediaDir string) error {
	prefilled, err := nodeapi.MediaServerLibraries(ctx, c, controller, claims.APIKey, id)
	if err != nil {
		return err
	}
	folders := map[string]string{}
	for _, r := range s.HeldRoots(p.MediaServerAt) {
		i := slices.IndexFunc(roots, func(nr nodeapi.Root) bool { return nr.Path == filepath.Join(mediaDir, r.Folder) })
		hidden := slices.ContainsFunc(p.Roots, func(h scenario.MediaServerRoot) bool { return h.Folder == r.Folder && h.Hidden })
		if i >= 0 && !hidden {
			folders[roots[i].ID] = p.Folder(r.Folder)
		}
	}
	mappings := make([]nodeapi.MediaServerMapping, len(prefilled))
	for i, m := range prefilled {
		folder, shown := folders[m.RootID]
		mappings[i] = nodeapi.MediaServerMapping{RootID: m.RootID, Shown: shown, ServerFolder: folder}
	}
	return nodeapi.SaveMediaServerMappings(ctx, c, controller, claims.APIKey, id, mappings)
}
