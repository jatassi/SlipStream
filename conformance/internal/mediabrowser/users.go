package mediabrowser

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// UserCases are the cases of a server's own users that both products
// run, each on throwaway users it makes and deletes.
func (s *Server) UserCases() []mediaservers.Case {
	return []mediaservers.Case{
		{Name: "UserMadeSignsIn", Run: s.UserMadeSignsIn},
		{Name: "UserNameTaken", Run: s.UserNameTaken},
		{Name: "UserRenamed", Run: s.UserRenamed},
		{Name: "UserPolicyRoundTrip", Run: s.UserPolicyRoundTrip},
		{Name: "EnabledFoldersRestrict", Run: s.EnabledFoldersRestrict},
		{Name: "UserDisabled", Run: s.UserDisabled},
		{Name: "UserDeletedTwice", Run: s.UserDeletedTwice},
		{Name: "FailedSignInsLock", Run: s.FailedSignInsLock},
	}
}

// Member is a throwaway user a case made.
type Member struct {
	ID, Name, Password string
}

func secret() string { return strings.ToLower(rand.Text()[:16]) }

// NewMember makes a user with a random name and password. The case
// defers Drop, since the cases share one *testing.T.
func (s *Server) NewMember(t *testing.T) Member {
	t.Helper()
	m := Member{Name: "slipstream-conformance-" + secret()[:10], Password: secret()}
	u, err := s.C.CreateUser(t.Context(), m.Name, m.Password)
	m.ID = u.ID
	if err != nil {
		if m.ID != "" {
			s.Drop(t, m)
		}
		t.Fatal(err)
	}
	return m
}

// dataSettle is how long Drop waits after a change to a user's watch data
// before it deletes the user. Jellyfin 12.2 and 12.4 exit when a user is
// deleted within about 10 s of one (online run 2026-11-26).
const dataSettle = 30 * time.Second

// Drop deletes m, dataSettle past the last change to its watch data on a
// real server. A user already deleted is no error.
func (s *Server) Drop(t *testing.T, m Member) {
	t.Helper()
	if at, ok := s.watched[m.ID]; ok && !s.tg.Fake {
		time.Sleep(time.Until(at.Add(dataSettle)))
	}
	if err := s.C.DeleteUser(t.Context(), m.ID); err != nil && !jellyfin.NotFound(err) {
		t.Errorf("deleting the throwaway user %s: %v", m.Name, err)
	}
}

// SignInAs signs in as name on a device of the suite's own, and signs out
// at once.
func (s *Server) SignInAs(t *testing.T, name, password string) error {
	t.Helper()
	ctx := t.Context()
	base, err := url.Parse(s.tg.Address)
	if err != nil {
		t.Fatal(err)
	}
	app := App
	app.DeviceID = "slipstream-conformance-member"
	auth, err := jellyfin.New(base, s.f, "", app, s.tg.HTTP).AuthenticateByName(ctx, name, password)
	if err != nil {
		return err
	}
	return jellyfin.New(base, s.f, auth.AccessToken, app, s.tg.HTTP).Logout(ctx)
}

// status is whether err is the server's answer with code.
func status(err error, code int) bool {
	e, ok := errors.AsType[*jellyfin.Error](err)
	return ok && e.Status == code
}

// Folder is the library key as a policy's EnabledFolders names it: the
// ItemId on Jellyfin, the Guid on Emby when byGUID.
func (s *Server) Folder(t *testing.T, key string, byGUID bool) string {
	t.Helper()
	if !byGUID {
		return key
	}
	libs, err := s.C.Libraries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(libs, func(l jellyfin.VirtualFolder) bool { return l.ItemID == key })
	if i < 0 || libs[i].GUID == "" {
		t.Fatalf("library %s has no Guid", key)
	}
	return libs[i].GUID
}

// folder is the target's library as this product's policy names it.
func (s *Server) folder(t *testing.T) string {
	t.Helper()
	return s.Folder(t, s.LibraryKey(t, s.tg.Library), s.f == jellyfin.Emby)
}

// Restrict posts m's policy back restricted to folders.
func (s *Server) Restrict(t *testing.T, m Member, folders ...string) {
	t.Helper()
	p, err := s.C.UserPolicy(t.Context(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	restricted, err := p.Restricted(folders)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.C.SetPolicy(t.Context(), m.ID, restricted); err != nil {
		t.Fatal(err)
	}
}

// Sees is how many videos m sees across the server.
func (s *Server) Sees(t *testing.T, m Member) int {
	t.Helper()
	page, err := s.C.Items(t.Context(), jellyfin.Query{User: m.ID, Recursive: true, Types: jellyfin.VideoTypes, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	return page.Total
}

func (s *Server) flags(t *testing.T, m Member) jellyfin.PolicyFlags {
	t.Helper()
	users, err := s.C.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(users, func(u jellyfin.User) bool { return u.ID == m.ID })
	if i < 0 {
		t.Fatalf("the users listing has no %s", m.Name)
	}
	return users[i].Policy
}

// UserMadeSignsIn makes a user with a random password with the key, then
// sets the Member's password. The listing has the user, not an
// administrator, and only the Member's password signs in. On Emby the
// key setting a password is the stop rule of S43 (Member access to Media
// Servers).
func (s *Server) UserMadeSignsIn(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	if s.flags(t, m).IsAdministrator {
		t.Fatalf("the new user %s is an administrator", m.Name)
	}
	chosen := secret()
	if err := s.C.SetPassword(t.Context(), m.ID, chosen); err != nil {
		t.Fatal(err)
	}
	if err := s.SignInAs(t, m.Name, chosen); err != nil {
		t.Fatalf("signing in with the password the key set: %v", err)
	}
	if err := s.SignInAs(t, m.Name, m.Password); !status(err, http.StatusUnauthorized) {
		t.Fatalf("signing in with the replaced password answered %v, want 401", err)
	}
}

// takenReply is each product's 400 to a name another user has in any
// case, from the online run 2026-11-26. Jellyfin's says nothing of why,
// so SlipStream reads the users before a rename.
func (s *Server) takenReply(name string) string {
	if s.f == jellyfin.Emby {
		return fmt.Sprintf("A user with the name '%s' already exists.", name)
	}
	return "Error processing request."
}

// refusedAs fails t unless err is the server's 400 to name taken.
func (s *Server) refusedAs(t *testing.T, err error, name string) {
	t.Helper()
	e, ok := errors.AsType[*jellyfin.Error](err)
	if !ok || e.Status != http.StatusBadRequest || e.Detail != s.takenReply(name) {
		t.Fatalf("the taken name %s answered %v, want 400 %q", name, err, s.takenReply(name))
	}
}

// UserNameTaken makes a user under a name another user has in other
// case, which the server refuses with its 400.
func (s *Server) UserNameTaken(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	upper := strings.ToUpper(m.Name)
	u, err := s.C.CreateUser(t.Context(), upper, secret())
	if err == nil {
		_ = s.C.DeleteUser(t.Context(), u.ID)
		t.Fatalf("a second user named %s in other case was made", m.Name)
	}
	s.refusedAs(t, err, upper)
}

// UserRenamed renames a user to another user's name in other case, which
// the server refuses with its 400, then to a free name, which signs in.
func (s *Server) UserRenamed(t *testing.T) {
	a, b := s.NewMember(t), s.NewMember(t)
	defer s.Drop(t, a)
	defer s.Drop(t, b)
	upper := strings.ToUpper(b.Name)
	s.refusedAs(t, s.C.UpdateUser(t.Context(), a.ID, upper), upper)
	renamed := a.Name + "-renamed"
	if err := s.C.UpdateUser(t.Context(), a.ID, renamed); err != nil {
		t.Fatal(err)
	}
	if err := s.SignInAs(t, renamed, a.Password); err != nil {
		t.Fatalf("signing in under the new name: %v", err)
	}
}

func decoded(t *testing.T, p jellyfin.Policy) map[string]any {
	t.Helper()
	out := map[string]any{}
	for k, v := range p {
		var x any
		if err := json.Unmarshal(v, &x); err != nil {
			t.Fatal(err)
		}
		out[k] = x
	}
	return out
}

// UserPolicyRoundTrip posts back the policy a GET read. Every field reads
// as it was.
func (s *Server) UserPolicyRoundTrip(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	before, err := s.C.UserPolicy(t.Context(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.C.SetPolicy(t.Context(), m.ID, before); err != nil {
		t.Fatal(err)
	}
	after, err := s.C.UserPolicy(t.Context(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b, a := decoded(t, before), decoded(t, after); !reflect.DeepEqual(b, a) {
		t.Fatalf("the policy changed in a round trip:\nbefore %v\nafter  %v", b, a)
	}
}

// EnabledFoldersRestrict restricts a user to no library, which then sees
// no video, and to the target's library, which it sees. EnabledFolders is
// how SlipStream hides a library: BlockedMediaFolders hides nothing on
// Jellyfin 12.2 and 12.4, and Emby 4.10 has no such field (online run
// 2026-11-26).
func (s *Server) EnabledFoldersRestrict(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	s.Restrict(t, m)
	if n := s.Sees(t, m); n != 0 {
		t.Fatalf("a user restricted to no library sees %d videos", n)
	}
	s.Restrict(t, m, s.folder(t))
	if s.Sees(t, m) == 0 {
		t.Fatalf("a user restricted to %s sees no video", s.tg.Library)
	}
}

// UserDisabled marks a video played for a restricted user, then disables
// the user, whose right password both products answer 403 and which keeps
// its watch data, and restricts it again, which lets it sign in.
func (s *Server) UserDisabled(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	s.Restrict(t, m, s.folder(t))
	page, err := s.C.Items(t.Context(), jellyfin.Query{User: m.ID, Recursive: true, Types: jellyfin.VideoTypes, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatalf("a user restricted to %s sees no video", s.tg.Library)
	}
	item := page.Items[0].ID
	s.watched[m.ID] = time.Now()
	if err := s.C.MarkPlayed(t.Context(), m.ID, item); err != nil {
		t.Fatal(err)
	}
	p, err := s.C.UserPolicy(t.Context(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.C.SetPolicy(t.Context(), m.ID, p.Disabled()); err != nil {
		t.Fatal(err)
	}
	if !s.flags(t, m).IsDisabled {
		t.Fatal("the listing shows the disabled user enabled")
	}
	if err := s.SignInAs(t, m.Name, m.Password); !status(err, http.StatusForbidden) {
		t.Fatalf("the disabled user's sign-in answered %v, want 403", err)
	}
	if d, err := s.C.UserData(t.Context(), m.ID, item); err != nil || !d.Played {
		t.Fatalf("the disabled user's data on %s is %+v (err %v), want it played", item, d, err)
	}
	s.Restrict(t, m, s.folder(t))
	if err := s.SignInAs(t, m.Name, m.Password); err != nil {
		t.Fatalf("signing in after the restrict enabled the user: %v", err)
	}
}

// UserDeletedTwice deletes a user, which leaves the listing, and deletes
// it again, which answers 404.
func (s *Server) UserDeletedTwice(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	if err := s.C.DeleteUser(t.Context(), m.ID); err != nil {
		t.Fatal(err)
	}
	users, err := s.C.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(users, func(u jellyfin.User) bool { return u.ID == m.ID }) {
		t.Fatalf("the listing still has the deleted %s", m.Name)
	}
	if err := s.C.DeleteUser(t.Context(), m.ID); !jellyfin.NotFound(err) {
		t.Fatalf("deleting the deleted user answered %v, want a 404", err)
	}
}

// lockedOut is whether p names when Emby locked the user out.
func lockedOut(t *testing.T, p jellyfin.Policy) bool {
	t.Helper()
	var at *time.Time
	if raw, ok := p["LockedOutDate"]; ok {
		if err := json.Unmarshal(raw, &at); err != nil {
			t.Fatal(err)
		}
	}
	return at != nil && !at.IsZero()
}

// FailedSignInsLock signs in with a wrong password until the user is
// locked out, at most 10 times, then with the right one. Emby 4.10 sets
// Policy.LockedOutDate and refuses the right password with 403, its
// IsDisabled still false. Jellyfin's new user has
// LoginAttemptsBeforeLockout -1 and never locks (online run 2026-11-26).
func (s *Server) FailedSignInsLock(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	s.Restrict(t, m, s.folder(t))
	const tries = 10
	locked := 0
	for n := 1; n <= tries && locked == 0; n++ {
		if err := s.SignInAs(t, m.Name, "wrong-"+m.Password); !status(err, http.StatusUnauthorized) {
			t.Fatalf("wrong password %d answered %v, want 401", n, err)
		}
		p, err := s.C.UserPolicy(t.Context(), m.ID)
		if err != nil {
			t.Fatal(err)
		}
		if lockedOut(t, p) {
			locked = n
		}
	}
	if s.flags(t, m).IsDisabled {
		t.Fatalf("%d wrong passwords disabled the user", tries)
	}
	err := s.SignInAs(t, m.Name, m.Password)
	if s.f == jellyfin.Emby {
		if locked == 0 || !status(err, http.StatusForbidden) {
			t.Fatalf("after %d wrong passwords Emby locked the user at %d and the right one answered %v, want a lock and 403", tries, locked, err)
		}
		t.Logf("%d wrong passwords locked the user out", locked)
		return
	}
	if locked != 0 || err != nil {
		t.Fatalf("after %d wrong passwords Jellyfin locked the user at %d and the right one answered %v, want no lock", tries, locked, err)
	}
}
