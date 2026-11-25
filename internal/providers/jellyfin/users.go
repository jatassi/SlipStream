package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
)

// User is one server user.
type User struct {
	ID     string      `json:"Id"`
	Name   string      `json:"Name"`
	Policy PolicyFlags `json:"Policy"`
}

// PolicyFlags are the fields of a user's policy that SlipStream reads.
// EnabledFolders holds library ItemIds on Jellyfin and library Guids on
// Emby.
type PolicyFlags struct {
	IsAdministrator  bool     `json:"IsAdministrator"`
	IsDisabled       bool     `json:"IsDisabled"`
	EnableAllFolders bool     `json:"EnableAllFolders"`
	EnabledFolders   []string `json:"EnabledFolders"`
}

func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out []User
	err := c.do(ctx, http.MethodGet, "/Users", nil, true, nil, &out)
	return out, err
}

func userPath(user string) string { return "/Users/" + url.PathEscape(user) }

// CreateUser makes the user name with password. Emby's /Users/New ignores
// a password, so on Emby a second call sets it. When that call fails, the
// returned User is the user the first call made.
func (c *Client) CreateUser(ctx context.Context, name, password string) (User, error) {
	var out User
	if err := c.do(ctx, http.MethodPost, "/Users/New", nil, true, map[string]string{"Name": name, "Password": password}, &out); err != nil {
		return User{}, err
	}
	if c.flavor == Emby {
		return out, c.SetPassword(ctx, out.ID, password)
	}
	return out, nil
}

// SetPassword gives user password. Jellyfin's POST /Users/Password takes
// the new password from an administrator's key. Emby's POST
// /Users/{id}/Password asks for the current one, so the password is reset
// to empty first and then changed from empty.
func (c *Client) SetPassword(ctx context.Context, user, password string) error {
	if c.flavor == Emby {
		path := userPath(user) + "/Password"
		if err := c.do(ctx, http.MethodPost, path, nil, true, map[string]any{"Id": user, "ResetPassword": true}, nil); err != nil {
			return err
		}
		return c.do(ctx, http.MethodPost, path, nil, true, map[string]any{"Id": user, "CurrentPw": "", "NewPw": password}, nil)
	}
	return c.do(ctx, http.MethodPost, "/Users/Password", url.Values{"userId": {user}}, true, map[string]any{"NewPw": password}, nil)
}

// UpdateUser renames user to name. It posts back the user as the server
// sent it, with the new Name.
func (c *Client) UpdateUser(ctx context.Context, user, name string) error {
	fields, err := c.Fields(ctx, userPath(user))
	if err != nil {
		return err
	}
	if fields["Name"], err = json.Marshal(name); err != nil {
		return fmt.Errorf("%s user %s: %w", c.flavor, user, err)
	}
	return c.do(ctx, http.MethodPost, userPath(user), nil, true, fields, nil)
}

// DeleteUser deletes user. A user already gone is a 404.
func (c *Client) DeleteUser(ctx context.Context, user string) error {
	return c.do(ctx, http.MethodDelete, userPath(user), nil, true, nil, nil)
}

// Policy is a user's whole policy with every field as the server sent it,
// so a write posts back the fields SlipStream doesn't set.
type Policy map[string]json.RawMessage

// UserPolicy is user's policy, read from GET /Users/{id}.
func (c *Client) UserPolicy(ctx context.Context, user string) (Policy, error) {
	fields, err := c.Fields(ctx, userPath(user))
	if err != nil {
		return nil, err
	}
	var p Policy
	if err := json.Unmarshal(fields["Policy"], &p); err != nil {
		return nil, fmt.Errorf("%s user %s: decode the policy: %w", c.flavor, user, err)
	}
	return p, nil
}

// SetPolicy replaces user's policy with p.
func (c *Client) SetPolicy(ctx context.Context, user string, p Policy) error {
	return c.do(ctx, http.MethodPost, userPath(user)+"/Policy", nil, true, p, nil)
}

// Restricted is p for a Member: not an administrator, not disabled, and
// seeing only folders, the libraries as EnabledFolders names them.
func (p Policy) Restricted(folders []string) (Policy, error) {
	enabled, err := json.Marshal(append([]string{}, folders...))
	if err != nil {
		return nil, fmt.Errorf("the enabled folders: %w", err)
	}
	return p.with(map[string]json.RawMessage{
		"IsAdministrator": jsonFalse, "IsDisabled": jsonFalse, "EnableAllFolders": jsonFalse,
		"EnabledFolders": enabled, "BlockedMediaFolders": json.RawMessage("[]"),
	}), nil
}

// Disabled is p with the user unable to sign in.
func (p Policy) Disabled() Policy {
	return p.with(map[string]json.RawMessage{"IsDisabled": json.RawMessage("true")})
}

var jsonFalse = json.RawMessage("false")

func (p Policy) with(set map[string]json.RawMessage) Policy {
	out := maps.Clone(p)
	maps.Copy(out, set)
	return out
}

// Taken is whether err is the server's 400 for a user name another user
// has in any case. Jellyfin's UserManager and Emby both answer "A user
// with the name '{name}' already exists." to a create and to a rename.
func Taken(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusBadRequest && strings.Contains(e.Detail, "A user with the name ")
}
