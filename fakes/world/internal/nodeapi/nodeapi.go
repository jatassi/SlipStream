// Package nodeapi is the Node API as the world speaks it: hand-written
// requests and responses, the way a script calls the API. Each call sends
// one request and never retries, and an answer it did not expect fails
// with the answer's body.
package nodeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/identity"
)

type Admin struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PIN         string `json:"pin"`
}

// ClaimController creates the first Admin through the Setup code and
// returns its API key.
func ClaimController(ctx context.Context, c *http.Client, baseURL, setupCode string, admin Admin) (string, error) {
	var claimed struct {
		APIKey string `json:"apiKey"`
	}
	_, err := call(ctx, c, request{
		what: "claim the Controller", method: http.MethodPost, url: baseURL + "/api/v1/setup/controller",
		header: http.Header{"X-Setup-Code": {setupCode}}, body: admin, want: http.StatusCreated,
	}, &claimed)
	return claimed.APIKey, err
}

func CreateJoinCode(ctx context.Context, c *http.Client, baseURL, apiKey string) (string, error) {
	var minted struct {
		Code string `json:"code"`
	}
	_, err := call(ctx, c, request{
		what: "create a Join Code", method: http.MethodPost, url: baseURL + "/api/v1/nodes/join-codes",
		header: apiKeyHeader(apiKey), want: http.StatusCreated,
	}, &minted)
	return minted.Code, err
}

// ClaimWorker joins an Unclaimed Node with code through its Setup code,
// and returns the id its Controller gave it. It answers once the link is
// live.
func ClaimWorker(ctx context.Context, c *http.Client, baseURL, setupCode, code string) (string, error) {
	var joined struct {
		NodeID string `json:"nodeId"`
	}
	_, err := call(ctx, c, request{
		what: "claim the Worker", method: http.MethodPost, url: baseURL + "/api/v1/setup/worker",
		header: http.Header{"X-Setup-Code": {setupCode}}, body: struct {
			Code string `json:"code"`
		}{code}, want: http.StatusCreated,
	}, &joined)
	return joined.NodeID, err
}

// Node is a listNodes row, as much of it as the world reads.
type Node struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Self          bool     `json:"self"`
	Roles         []string `json:"roles"`
	IntentVersion string   `json:"intentVersion"`
	Link          struct {
		State string `json:"state"`
	} `json:"link"`
	Facts *struct {
		AppliedVersion string `json:"appliedVersion"`
	} `json:"facts"`
	// Body is the answer the row came in.
	Body string `json:"-"`
}

func ListNodes(ctx context.Context, c *http.Client, baseURL, apiKey string) ([]Node, error) {
	var list struct {
		Nodes []Node `json:"nodes"`
	}
	_, err := call(ctx, c, request{
		what: "list Nodes", method: http.MethodGet, url: baseURL + "/api/v1/nodes",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &list)
	return list.Nodes, err
}

// SetNodeRoles makes roles the Roles of Node id. The Controller answers
// once a Node in step holds them.
func SetNodeRoles(ctx context.Context, c *http.Client, baseURL, apiKey, id string, roles []string) (Node, error) {
	var row Node
	reply, err := call(ctx, c, request{
		what: "set the Roles of Node " + id, method: http.MethodPut, url: baseURL + "/api/v1/nodes/" + id + "/roles",
		header: apiKeyHeader(apiKey), body: struct {
			Roles []string `json:"roles"`
		}{append([]string{}, roles...)}, want: http.StatusOK,
	}, &row)
	row.Body = string(reply)
	return row, err
}

// CreateRoot makes path the root of contentType and returns its id.
func CreateRoot(ctx context.Context, c *http.Client, baseURL, apiKey, contentType, path string) (string, error) {
	var root struct {
		ID string `json:"id"`
	}
	_, err := call(ctx, c, request{
		what: "create the " + contentType + " root", method: http.MethodPost, url: baseURL + "/api/v1/roots",
		header: apiKeyHeader(apiKey), body: struct {
			ContentType string `json:"contentType"`
			Path        string `json:"path"`
		}{contentType, path}, want: http.StatusCreated,
	}, &root)
	return root.ID, err
}

// CreateIndexer creates an Indexer and returns its id.
func CreateIndexer(ctx context.Context, c *http.Client, baseURL, apiKey, name, url, indexerKey string) (string, error) {
	var indexer struct {
		ID string `json:"id"`
	}
	_, err := call(ctx, c, request{
		what: "create Indexer " + name, method: http.MethodPost, url: baseURL + "/api/v1/indexers",
		header: apiKeyHeader(apiKey), body: struct {
			Name   string `json:"name"`
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		}{name, url, indexerKey}, want: http.StatusCreated,
	}, &indexer)
	return indexer.ID, err
}

// CreateTarget creates the Scenario Target t and returns its id.
func CreateTarget(ctx context.Context, c *http.Client, baseURL, apiKey string, t scenario.LibraryTarget) (string, error) {
	type filter struct {
		Attribute string   `json:"attribute"`
		Mode      string   `json:"mode"`
		Values    []string `json:"values"`
	}
	type size struct {
		Quality scenario.QualityKey `json:"quality"`
		Min     float64             `json:"min"`
		Max     *float64            `json:"max,omitempty"`
	}
	body := struct {
		Name    string                  `json:"name"`
		Tiers   [][]scenario.QualityKey `json:"tiers"`
		Cutoff  scenario.QualityKey     `json:"cutoff"`
		Filters []filter                `json:"filters,omitempty"`
		Sizes   []size                  `json:"sizes,omitempty"`
	}{Name: t.Name, Tiers: t.Tiers, Cutoff: t.Cutoff}
	for _, f := range t.Filters {
		body.Filters = append(body.Filters, filter(f))
	}
	for _, z := range t.Sizes {
		body.Sizes = append(body.Sizes, size(z))
	}
	var target struct {
		ID string `json:"id"`
	}
	_, err := call(ctx, c, request{
		what: "create Target " + t.Name, method: http.MethodPost, url: baseURL + "/api/v1/targets",
		header: apiKeyHeader(apiKey), body: body, want: http.StatusCreated,
	}, &target)
	return target.ID, err
}

// CreateProfile creates a Profile of the Targets with ids targetIDs and
// returns its id.
func CreateProfile(ctx context.Context, c *http.Client, baseURL, apiKey, name, contentType string, targetIDs []string) (string, error) {
	var profile struct {
		ID string `json:"id"`
	}
	_, err := call(ctx, c, request{
		what: "create Profile " + name, method: http.MethodPost, url: baseURL + "/api/v1/profiles",
		header: apiKeyHeader(apiKey), body: struct {
			Name        string   `json:"name"`
			ContentType string   `json:"contentType"`
			TargetIDs   []string `json:"targetIds"`
		}{name, contentType, targetIDs}, want: http.StatusCreated,
	}, &profile)
	return profile.ID, err
}

// SetAddress sets the address the Controller builds its links on.
func SetAddress(ctx context.Context, c *http.Client, baseURL, apiKey, address string) error {
	var set struct{}
	_, err := call(ctx, c, request{
		what: "set the address", method: http.MethodPut, url: baseURL + "/api/v1/identity/address",
		header: apiKeyHeader(apiKey), body: struct {
			Address string `json:"address"`
		}{address}, want: http.StatusOK,
	}, &set)
	return err
}

func CreateSignupLink(ctx context.Context, c *http.Client, baseURL, apiKey, email string) (string, error) {
	var issued struct {
		URL string `json:"url"`
	}
	_, err := call(ctx, c, request{
		what: "issue a Signup link for " + email, method: http.MethodPost, url: baseURL + "/api/v1/signup-links",
		header: apiKeyHeader(apiKey), body: struct {
			Email string `json:"email"`
		}{email}, want: http.StatusCreated,
	}, &issued)
	return issued.URL, err
}

// RedeemSignupLink makes the Account a Signup link's token was issued for,
// as its page does, and returns the Account's id and the session cookie
// the redeem signed in with.
func RedeemSignupLink(ctx context.Context, c *http.Client, baseURL, token, displayName, pin string) (string, *http.Cookie, error) {
	var joined struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	var cookies []*http.Cookie
	_, err := call(ctx, c, request{
		what: "redeem the Signup link for " + displayName, method: http.MethodPost, url: baseURL + "/api/v1/signup-links/by-token/" + token + "/account",
		body: struct {
			DisplayName string `json:"displayName"`
			PIN         string `json:"pin"`
		}{displayName, pin}, want: http.StatusOK, cookies: &cookies,
	}, &joined)
	if err != nil {
		return "", nil, err
	}
	for _, ck := range cookies {
		if ck.Name == identity.CookieName {
			return joined.Account.ID, ck, nil
		}
	}
	return "", nil, fmt.Errorf("redeem the Signup link for %s: no session cookie", displayName)
}

func SignOut(ctx context.Context, c *http.Client, baseURL string, cookie *http.Cookie) error {
	_, err := call(ctx, c, request{
		what: "sign out", method: http.MethodDelete, url: baseURL + "/api/v1/session",
		header: http.Header{"Cookie": {cookie.String()}}, want: http.StatusNoContent,
	}, nil)
	return err
}

func MakeAdmin(ctx context.Context, c *http.Client, baseURL, apiKey, id string) error {
	var account struct{}
	_, err := call(ctx, c, request{
		what: "make Account " + id + " an Admin", method: http.MethodPatch, url: baseURL + "/api/v1/accounts/" + id,
		header: apiKeyHeader(apiKey), body: struct {
			Admin bool `json:"admin"`
		}{true}, want: http.StatusOK,
	}, &account)
	return err
}

// Mail is the email setting as the world saves it: a server without TLS
// or sign-in.
type Mail struct {
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	Security    string `json:"security"`
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
}

func SetMail(ctx context.Context, c *http.Client, baseURL, apiKey string, m Mail) error {
	var settings struct{}
	_, err := call(ctx, c, request{
		what: "save the email setting", method: http.MethodPatch, url: baseURL + "/api/v1/mail/settings",
		header: apiKeyHeader(apiKey), body: m, want: http.StatusOK,
	}, &settings)
	return err
}

func apiKeyHeader(key string) http.Header { return http.Header{"X-Api-Key": {key}} }

type request struct {
	what, method, url string
	header            http.Header
	body              any
	want              int
	cookies           *[]*http.Cookie
}

// call sends r once and decodes a reply with the status r wants into out.
// It returns the reply.
func call(ctx context.Context, c *http.Client, r request, out any) ([]byte, error) {
	var body io.Reader = http.NoBody
	if r.body != nil {
		data, err := json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", r.what, err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, r.url, body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.what, err)
	}
	for k, v := range r.header {
		req.Header[k] = v
	}
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.what, err)
	}
	defer func() { _ = res.Body.Close() }()
	reply, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.what, err)
	}
	if res.StatusCode != r.want {
		return nil, fmt.Errorf("%s: %d %s", r.what, res.StatusCode, bytes.TrimSpace(reply))
	}
	if r.cookies != nil {
		*r.cookies = res.Cookies()
	}
	if out == nil {
		return nil, nil
	}
	if err := json.Unmarshal(reply, out); err != nil {
		return nil, fmt.Errorf("%s: %w", r.what, err)
	}
	return bytes.TrimSpace(reply), nil
}
