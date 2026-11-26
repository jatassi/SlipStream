package mbkind

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

var _ link.Users = (*conn)(nil)

func (c *conn) CreateUser(ctx context.Context, name string) (string, error) {
	var u jellyfin.User
	err := c.onSaved(ctx, func() error {
		var err error
		u, err = c.c.CreateUser(ctx, name, rand.Text())
		return nameRefused(err)
	})
	return u.ID, err
}

func (c *conn) Restrict(ctx context.Context, user string, libraries []string) error {
	return c.onSaved(ctx, func() error {
		folders, err := c.folders(ctx, libraries)
		if err != nil {
			return err
		}
		p, err := c.c.UserPolicy(ctx, user)
		if err != nil {
			return err
		}
		restricted, err := p.Restricted(folders)
		if err != nil {
			return err
		}
		return c.c.SetPolicy(ctx, user, restricted)
	})
}

// folders are the libraries as a policy's EnabledFolders names them: the
// key, or Emby's Guid.
func (c *conn) folders(ctx context.Context, libraries []string) ([]string, error) {
	if !c.f.foldersByGUID {
		return libraries, nil
	}
	vfs, err := c.c.Libraries(ctx)
	if err != nil {
		return nil, err
	}
	guid := make(map[string]string, len(vfs))
	for _, vf := range vfs {
		guid[vf.ItemID] = vf.GUID
	}
	out := make([]string, len(libraries))
	for i, l := range libraries {
		g, ok := guid[l]
		if !ok {
			return nil, failed("the Media Server has no library " + l)
		}
		out[i] = g
	}
	return out, nil
}

func (c *conn) SetPassword(ctx context.Context, user string, pw link.Password) error {
	return c.onSaved(ctx, func() error { return c.c.SetPassword(ctx, user, pw.Reveal()) })
}

// Rename reads the users first: Jellyfin answers a taken name with a 400
// that doesn't say why, "Error processing request.".
func (c *conn) Rename(ctx context.Context, user, name string) error {
	return c.onSaved(ctx, func() error {
		users, err := c.c.Users(ctx)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(users, func(u jellyfin.User) bool { return u.ID != user && strings.EqualFold(u.Name, name) }) {
			return &link.NameRefusedError{Taken: true}
		}
		return nameRefused(c.c.UpdateUser(ctx, user, name))
	})
}

func (c *conn) Disable(ctx context.Context, user string) error {
	return c.onSaved(ctx, func() error {
		p, err := c.c.UserPolicy(ctx, user)
		if err != nil {
			return err
		}
		return c.c.SetPolicy(ctx, user, p.Disabled())
	})
}

func (c *conn) DeleteUser(ctx context.Context, user string) error {
	return c.onSaved(ctx, func() error {
		if err := c.c.DeleteUser(ctx, user); !jellyfin.NotFound(err) {
			return err
		}
		return nil
	})
}

func (c *conn) ListUsers(ctx context.Context) ([]link.User, error) {
	var users []jellyfin.User
	err := c.onSaved(ctx, func() error {
		var err error
		users, err = c.c.Users(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]link.User, len(users))
	for i, u := range users {
		out[i] = link.User{ID: u.ID, Name: u.Name, Administrator: u.Policy.IsAdministrator, Disabled: u.Policy.IsDisabled}
	}
	return out, nil
}

// onSaved reads the server's identity first, as a step does, so no user
// is made, changed or deleted on another server at the address, then runs
// call, its error in the link's words.
func (c *conn) onSaved(ctx context.Context, call func() error) error {
	if _, err := c.same(ctx); err != nil {
		return problem(ctx, err)
	}
	err := call()
	if _, ok := errors.AsType[*link.NameRefusedError](err); ok {
		return err
	}
	return problem(ctx, err)
}

// nameRefused is err with the server's 400 to a create or a rename as the
// one reply about a user's name. A name taken since Rename read the users
// reads as refused.
func nameRefused(err error) error {
	if e, ok := errors.AsType[*jellyfin.Error](err); ok && e.Status == http.StatusBadRequest {
		return &link.NameRefusedError{Reply: e.Detail}
	}
	return err
}
