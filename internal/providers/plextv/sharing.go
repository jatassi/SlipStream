package plextv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Found is a Plex account's standing on one of the Admin's servers: its
// share, accepted or pending, and whether it is in the Admin's Plex Home.
// A zero Share is none.
type Found struct {
	Share    ShareID
	Accepted bool
	Home     bool
}

// wireID is a plex.tv id, which JSON carries as a number and XML as text.
type wireID string

func (id *wireID) UnmarshalJSON(b []byte) error {
	*id = wireID(bytes.Trim(b, `"`))
	return nil
}

// is whether the account plex.tv names by id or uuid is w.
func (w Who) is(id, uuid string) bool {
	return id == strconv.FormatInt(int64(w.ID), 10) || (uuid != "" && uuid == w.UUID)
}

func machineVars(machine string) []string { return []string{"{machine}", machine} }

// Sections are the plex.tv ids of machine's library sections, by the
// section key the server itself uses.
func (c *Client) Sections(ctx context.Context, admin, machine string) (map[string]SectionID, error) {
	var out struct {
		Server struct {
			Sections []struct {
				ID  string `xml:"id,attr"`
				Key string `xml:"key,attr"`
			} `xml:"Section"`
		} `xml:"Server"`
	}
	if err := c.t.do(ctx, request{route: routes[sections], vars: machineVars(machine), token: admin}, &out); err != nil {
		return nil, err
	}
	ids := make(map[string]SectionID, len(out.Server.Sections))
	for _, s := range out.Server.Sections {
		ids[s.Key] = SectionID(s.ID)
	}
	return ids, nil
}

type ownedShare struct {
	ID                wireID `json:"id"`
	MachineIdentifier string `json:"machineIdentifier"`
	InvitedID         wireID `json:"invitedId"`
	Invited           struct {
		UUID string `json:"uuid"`
	} `json:"invited"`
}

// FindShare reads owned/accepted, then owned/pending, for member's share
// of machine, and the Admin's Home users.
func (c *Client) FindShare(ctx context.Context, admin, machine string, member Who) (Found, error) {
	var found Found
	for _, listing := range []struct {
		call     call
		accepted bool
	}{{ownedAccepted, true}, {ownedPending, false}} {
		var out []ownedShare
		if err := c.t.do(ctx, request{route: routes[listing.call], token: admin}, &out); err != nil {
			return Found{}, err
		}
		for _, s := range out {
			if found.Share == "" && s.MachineIdentifier == machine && member.is(string(s.InvitedID), s.Invited.UUID) {
				found.Share, found.Accepted = ShareID(s.ID), listing.accepted
			}
		}
	}
	home, err := c.homeUser(ctx, admin, member)
	found.Home = home != ""
	return found, err
}

// Share shares sections of machine with member. On plex.tv's 400 "You're
// already sharing this server with {user}" it answers the share FindShare
// finds instead.
func (c *Client) Share(ctx context.Context, admin, machine string, member AccountID, sections []SectionID) (ShareID, error) {
	ids := make([]json.Number, len(sections))
	for i, s := range sections {
		ids[i] = json.Number(s)
	}
	body := map[string]any{
		"server_id":        machine,
		"shared_server":    map[string]any{"library_section_ids": ids, "invited_id": member},
		"sharing_settings": map[string]any{},
	}
	var out struct {
		SharedServer struct {
			ID string `xml:"id,attr"`
		} `xml:"SharedServer"`
	}
	err := c.t.do(ctx, request{route: routes[share], vars: machineVars(machine), token: admin, body: body}, &out)
	if err == nil {
		return ShareID(out.SharedServer.ID), nil
	}
	if !alreadySharing(err) {
		return "", err
	}
	found, ferr := c.FindShare(ctx, admin, machine, Who{ID: member})
	switch {
	case ferr != nil:
		return "", ferr
	case found.Share == "":
		return "", err
	}
	return found.Share, nil
}

func alreadySharing(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusBadRequest && strings.Contains(e.Detail, "already sharing this server")
}

// InviteHome invites member's existing Plex account into the Admin's
// Plex Home. [unverified] past plex.tv's Plex Pass 403.
func (c *Client) InviteHome(ctx context.Context, admin string, member Invitee) error {
	return c.t.do(ctx, request{route: routes[inviteHome], query: url.Values{"invitedEmail": {member.Email}}, token: admin}, nil)
}

type invite struct {
	ID     string `xml:"id,attr"`
	Friend string `xml:"friend,attr"`
	Server string `xml:"server,attr"`
	Home   string `xml:"home,attr"`
}

// Unshare deletes that one share, keeping the friendship, then cancels
// member's invite when invites/requested still lists it with no pending
// share, as a share deleted before accept leaves it. A 404 is nil.
func (c *Client) Unshare(ctx context.Context, admin, machine string, share ShareID, member Who) error {
	vars := append(machineVars(machine), "{share}", string(share))
	if err := c.t.do(ctx, request{route: routes[deleteShare], vars: vars, token: admin}, nil); status(err) != http.StatusNotFound && err != nil {
		return err
	}
	var out struct {
		Invites []invite `xml:"Invite"`
	}
	if err := c.t.do(ctx, request{route: routes[invites], token: admin}, &out); err != nil {
		return err
	}
	for _, inv := range out.Invites {
		if !member.is(inv.ID, "") || inv.Server != "0" {
			continue
		}
		q := url.Values{"friend": {inv.Friend}, "server": {inv.Server}, "home": {inv.Home}}
		err := c.t.do(ctx, request{route: routes[cancelInvite], vars: []string{"{invite}", inv.ID}, query: q, token: admin}, nil)
		if status(err) != http.StatusNotFound && err != nil {
			return err
		}
	}
	return nil
}

// LeaveHome removes member from the Admin's Plex Home. Not in the Home is
// nil. [unverified]
func (c *Client) LeaveHome(ctx context.Context, admin string, member Who) error {
	id, err := c.homeUser(ctx, admin, member)
	if err != nil || id == "" {
		return err
	}
	err = c.t.do(ctx, request{route: routes[leaveHome], vars: []string{"{member}", id}, token: admin}, nil)
	if status(err) == http.StatusNotFound {
		return nil
	}
	return err
}

// homeUser is member's id among the Admin's Home users, "" when it isn't
// one.
func (c *Client) homeUser(ctx context.Context, admin string, member Who) (string, error) {
	var out struct {
		Users []struct {
			ID   string `xml:"id,attr"`
			UUID string `xml:"uuid,attr"`
		} `xml:"User"`
	}
	if err := c.t.do(ctx, request{route: routes[homeUsers], token: admin}, &out); err != nil {
		return "", err
	}
	for _, u := range out.Users {
		if member.is(u.ID, u.UUID) {
			return u.ID, nil
		}
	}
	return "", nil
}
