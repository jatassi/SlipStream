package mediaservers

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// usersWay gives access with a user on the server, along a route.
type usersWay struct{ m *MediaServers }

func (usersWay) way() access.Way { return access.WayUsers }

func (usersWay) offered(context.Context, *db.Queries, identity.AccountID) (step, error) {
	return step{}, nil
}

func (w usersWay) howToWatch(a db.MediaserversMemberAccess, srv db.MediaserversServer, _ db.MediaserversMemberPlex) HowToWatch {
	return HowToWatch{App: w.m.kind(srv.Kind).About().Label, SignInAs: deref(a.Username), Address: deref(srv.MemberAddress)}
}

// remove disables the row's user, which keeps its Watch data for a
// re-offer. A user already deleted on the server has nothing to remove.
func (w usersWay) remove(ctx context.Context, r accessRow) (step, error) {
	st, err := w.m.onServer(ctx, r, func(u usersAt) error {
		if err := u.Disable(ctx, *r.ServerUserID); !link.Gone(err) {
			return err
		}
		return nil
	})
	return left(r, st), err
}

// reoffer enables the row's user again with the current Libraries for
// Members: Gone when the server deleted it.
func (w usersWay) reoffer(ctx context.Context, r accessRow) (step, error) {
	gone := false
	st, err := w.m.onServer(ctx, r, func(u usersAt) error {
		err := u.Restrict(ctx, *r.ServerUserID, keysOf(r.libraries))
		gone = link.Gone(err)
		if gone {
			return nil
		}
		return err
	})
	if gone {
		st.outcome = access.Gone
	}
	return st, err
}

// MakeAccount makes the Member's user on the server. Making is written
// before the create and the user id after it, the user carries a name only
// this row could have made until its id is recorded, and the Member's name
// is set last, so no stop between the calls leaves a user that unmake
// can't find, and unmake never deletes a stranger.
func (s service) MakeAccount(ctx context.Context, me identity.Account, id AccessID, name string, pw link.Password) (MyAccessRow, error) {
	r, release, err := s.m.mine(ctx, me, id)
	if err != nil {
		return MyAccessRow{}, err
	}
	defer release()
	if r.State == access.Making {
		if r, err = s.m.unmadeAgain(ctx, r); err != nil {
			return MyAccessRow{}, err
		}
	}
	if !access.Starts(r.State, access.MakeAccount, r.Way) {
		return MyAccessRow{}, &AccessStateError{Member: r.MemberName, Server: r.server.Name}
	}
	u, err := s.m.usersOf(ctx, r.ServerID)
	if err != nil {
		return MyAccessRow{}, err
	}
	made, err := s.m.move(ctx, r, access.Must(access.Offered, access.MakeAccount, access.Done, access.WayUsers), step{set: rowSet{username: &name}})
	if err != nil {
		return MyAccessRow{}, err
	}
	s.m.afterMove(ctx, made)
	r = made.row
	uid, err := u.CreateUser(ctx, tempName(r.ID))
	if err != nil {
		return MyAccessRow{}, s.m.failMake(ctx, u, r, uid, err)
	}
	created, err := s.m.move(ctx, r, access.Must(access.Making, access.Created, access.Done, access.WayUsers), step{set: rowSet{userID: &uid}})
	if err != nil {
		return MyAccessRow{}, err
	}
	r = created.row
	for _, call := range []func() error{
		func() error { return u.Restrict(ctx, uid, keysOf(r.libraries)) },
		func() error { return u.SetPassword(ctx, uid, pw) },
		func() error { return u.Rename(ctx, uid, name) },
	} {
		if err := call(); err != nil {
			return MyAccessRow{}, s.m.failMake(ctx, u, r, uid, err)
		}
	}
	finished, err := s.m.move(ctx, r, access.Must(access.Making, access.Finish, access.Done, access.WayUsers), step{})
	if err != nil {
		return MyAccessRow{}, err
	}
	s.m.afterMove(ctx, finished)
	return s.myRowOf(ctx, me, id)
}

// unmadeAgain deletes the user a make the Member is retrying left, and
// answers the row offered again.
func (m *MediaServers) unmadeAgain(ctx context.Context, r accessRow) (accessRow, error) {
	if _, err := m.actLocked(ctx, r, access.Unmake, m.unmake); err != nil {
		return accessRow{}, err
	}
	again, err := m.row(ctx, r.ID)
	if err == nil && again.State == access.Making {
		return accessRow{}, &AccessUnreachedError{Server: r.server.Name}
	}
	return again, err
}

// tempName is the name a made user carries until its id is recorded:
// the random end of the access id, which no other row shares.
func tempName(id AccessID) string {
	b := id.Bytes()
	return "slipstream-" + hex.EncodeToString(b[len(b)-6:])
}

// failMake deletes what a failed make left on the server and offers the
// row again with the problem, or leaves it making for the sweep when the
// delete fails too. It answers the Member the server's reply to the name,
// or that the server isn't responding.
func (m *MediaServers) failMake(ctx context.Context, u usersAt, r accessRow, uid string, cause error) error {
	st := step{outcome: access.Done, problem: u.problemOf(cause)}
	if err := u.deleteMade(ctx, r, uid); err != nil {
		st = step{outcome: access.Failed, problem: u.problemOf(err)}
	}
	w, err := m.move(ctx, r, access.Must(access.Making, access.Unmake, st.outcome, access.WayUsers), st)
	if err != nil {
		return err
	}
	m.afterMove(ctx, w)
	if refused, ok := errors.AsType[*link.NameRefusedError](cause); ok {
		return refused
	}
	return &AccessUnreachedError{Server: u.server}
}

// unmake deletes the user a making row may have made: Done, Left once the
// row's Account is gone, or Failed.
func (m *MediaServers) unmake(ctx context.Context, r accessRow) (step, error) {
	st, err := m.onServer(ctx, r, func(u usersAt) error { return u.deleteMade(ctx, r, "") })
	return left(r, st), err
}

// deleteMade deletes the user r's make made: by its id, or uid when the
// create answered one the row doesn't hold yet, else by r's temporary
// name. It is the only user SlipStream deletes, a removal disables, and
// it never had watch data: no one signs in under its temporary name and a
// carry writes no data a new item holds. Jellyfin 12.2 and 12.4 exit when
// a user is deleted within seconds of a change to its watch data (online
// run 2026-11-26).
func (u usersAt) deleteMade(ctx context.Context, r accessRow, uid string) error {
	if r.ServerUserID != nil {
		uid = *r.ServerUserID
	}
	if uid == "" {
		users, err := u.ListUsers(ctx)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(users, func(user link.User) bool { return user.Name == tempName(r.ID) })
		if i < 0 {
			return nil
		}
		uid = users[i].ID
	}
	return u.DeleteUser(ctx, uid)
}

// unmakeAll unmakes every making row, one turn at a time: the makes a
// stop cut short, and the rows whose server was unreached last time.
func (m *MediaServers) unmakeAll(ctx context.Context) {
	var ids []AccessID
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		as, err := db.New(r).ListAccessIn(ctx, access.Making)
		for _, a := range as {
			ids = append(ids, a.ID)
		}
		return err
	})
	if err != nil {
		m.logSweep(ctx, err)
		return
	}
	for _, id := range ids {
		if err := m.unmakeOne(ctx, id); err != nil && !errors.Is(err, ErrAccessNotFound) {
			m.logSweep(ctx, err)
		}
	}
}

// unmakeOne unmakes row id under its turn, unless a make that held the
// turn finished it meanwhile.
func (m *MediaServers) unmakeOne(ctx context.Context, id AccessID) error {
	r, release, err := m.lockRow(ctx, id)
	if err != nil {
		return err
	}
	defer release()
	if r.State != access.Making {
		return nil
	}
	_, err = m.actLocked(ctx, r, access.Unmake, m.unmake)
	return err
}

func (m *MediaServers) logSweep(ctx context.Context, err error) {
	if ctx.Err() == nil {
		m.cfg.Log.Error("delete the users of Members' unfinished accounts", "err", err.Error())
	}
}

// sweepMaking runs unmakeAll at the start and at each change of the
// roster, when a server's Node may be back.
func (m *MediaServers) sweepMaking(ctx context.Context) {
	for {
		changed := m.cfg.Roster.Changed()
		m.unmakeAll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-changed:
		}
	}
}

// LinkUser gives the Member an existing user of the server, which keeps
// its own libraries.
func (s service) LinkUser(ctx context.Context, _ identity.Admin, id AccessID, user string) (Acted, error) {
	return s.m.act(ctx, id, access.Link, func(ctx context.Context, r accessRow) (step, error) {
		u, err := s.m.usersOf(ctx, r.ServerID)
		if err != nil {
			return step{}, err
		}
		users, err := s.m.linkable(ctx, u, r.ServerID)
		if err != nil {
			return step{}, err
		}
		i := slices.IndexFunc(users, func(l link.User) bool { return l.ID == user })
		if i < 0 {
			return step{}, &AccessStateError{Member: r.MemberName, Server: r.server.Name}
		}
		return step{set: rowSet{userID: &users[i].ID, username: &users[i].Name}}, nil
	})
}

func (s service) LinkableUsers(ctx context.Context, server link.ServerID) ([]ServerUser, error) {
	u, err := s.m.usersOf(ctx, server)
	if err != nil {
		return nil, err
	}
	users, err := s.m.linkable(ctx, u, server)
	out := make([]ServerUser, len(users))
	for i, l := range users {
		out[i] = ServerUser{ID: l.ID, Name: l.Name}
	}
	return out, err
}

// linkable is the server's users with no administrator, no disabled user
// and no user a row already holds.
func (m *MediaServers) linkable(ctx context.Context, u usersAt, server link.ServerID) ([]link.User, error) {
	users, err := u.ListUsers(ctx)
	if err != nil {
		return nil, &AccessUnreachedError{Server: u.server}
	}
	var held []*string
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		held, err = db.New(r).ListServerUserIDs(ctx, server)
		return err
	})
	return slices.DeleteFunc(users, func(l link.User) bool {
		return l.Administrator || l.Disabled || slices.ContainsFunc(held, func(id *string) bool { return *id == l.ID })
	}), err
}

// mine reads the Member's own row id and takes its turn: not found for
// another Account's row.
func (m *MediaServers) mine(ctx context.Context, me identity.Account, id AccessID) (accessRow, func(), error) {
	r, release, err := m.lockRow(ctx, id)
	if err != nil {
		return accessRow{}, nil, err
	}
	if r.AccountID != me.Profile().ID {
		release()
		return accessRow{}, nil, ErrAccessNotFound
	}
	return r, release, nil
}

// myRowOf is row id as its Member reads it.
func (s service) myRowOf(ctx context.Context, me identity.Account, id AccessID) (MyAccessRow, error) {
	all, err := s.Mine(ctx, me)
	if err != nil {
		return MyAccessRow{}, err
	}
	i := slices.IndexFunc(all.Rows, func(row MyAccessRow) bool { return row.ID == id.String() })
	if i < 0 {
		return MyAccessRow{}, ErrAccessNotFound
	}
	return all.Rows[i], nil
}

// onServer runs call on r's server along its route. A failure on the
// server or its route is the step's Failed outcome with the problem; an
// error is a local one.
func (m *MediaServers) onServer(ctx context.Context, r accessRow, call func(usersAt) error) (step, error) {
	u, err := m.usersOf(ctx, r.ServerID)
	if held, ok := errors.AsType[*AccessUnreachedError](err); ok {
		return step{outcome: access.Failed, problem: &AccessProblem{
			Code: ProblemMediaServerUnreached, Detail: held.Server + "'s Node is offline, and the Controller can't use a fallback address.",
		}}, nil
	}
	if err != nil {
		return step{}, err
	}
	if err := call(u); err != nil {
		return step{outcome: access.Failed, problem: u.problemOf(err)}, nil
	}
	return step{}, nil
}

// left is st for a row whose Account is gone: its success keeps nothing.
func left(r accessRow, st step) step {
	if st.outcome == access.Done && r.AccountID.IsZero() {
		st.outcome = access.Left
	}
	return st
}

// problemOf is a failed call on u's server as an Admin reads it.
func (u usersAt) problemOf(err error) *AccessProblem {
	if refused, ok := errors.AsType[*link.NameRefusedError](err); ok {
		if refused.Taken {
			return &AccessProblem{Code: ProblemUsernameTaken, Detail: u.server + " has a user with this username."}
		}
		return &AccessProblem{Code: ProblemServerRefused, Detail: fmt.Sprintf("%s refused the username. Its response was %q.", u.server, refused.Reply)}
	}
	if p, ok := link.ServerProblem(err); ok && p.Reason != link.ReasonUnreachable {
		detail := fmt.Sprintf("%s responded with status %d.", u.server, p.Status)
		if p.Detail != "" {
			detail += fmt.Sprintf(" Its response was %q.", p.Detail)
		}
		return &AccessProblem{Code: ProblemFailed, Detail: detail}
	}
	if u.fallback {
		return &AccessProblem{Code: ProblemMediaServerUnreached, Detail: "The Controller couldn't use the fallback address. Test the server again."}
	}
	return &AccessProblem{Code: ProblemMediaServerUnreached, Detail: u.server + " isn't responding to SlipStream."}
}

func keysOf(libs []LibraryChoice) []string {
	out := make([]string, len(libs))
	for i, l := range libs {
		out[i] = l.Key
	}
	return out
}
