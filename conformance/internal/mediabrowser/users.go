package mediabrowser

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

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
		{Name: "BlockedBeatsEnabled", Run: s.BlockedBeatsEnabled},
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

// Drop deletes m. A user already deleted is no error.
func (s *Server) Drop(t *testing.T, m Member) {
	t.Helper()
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

func refused(err error) bool {
	e, ok := errors.AsType[*jellyfin.Error](err)
	return ok && (e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden)
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
	if err := s.SignInAs(t, m.Name, m.Password); !refused(err) {
		t.Fatalf("signing in with the replaced password answered %v, want a refusal", err)
	}
}

// UserNameTaken makes a user under a name another user has in other
// case. The server answers 400, and Taken reads it.
func (s *Server) UserNameTaken(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	u, err := s.C.CreateUser(t.Context(), strings.ToUpper(m.Name), secret())
	if err == nil {
		_ = s.C.DeleteUser(t.Context(), u.ID)
		t.Fatalf("a second user named %s in other case was made", m.Name)
	}
	if !jellyfin.Taken(err) {
		t.Fatalf("the duplicate name answered %v, want Taken's 400", err)
	}
	t.Logf("the duplicate name answered %v", err)
}

// UserRenamed renames a user to another user's name in other case, which
// the server refuses with Taken's 400, then to a free name, which signs
// in.
func (s *Server) UserRenamed(t *testing.T) {
	a, b := s.NewMember(t), s.NewMember(t)
	defer s.Drop(t, a)
	defer s.Drop(t, b)
	if err := s.C.UpdateUser(t.Context(), a.ID, strings.ToUpper(b.Name)); !jellyfin.Taken(err) {
		t.Fatalf("renaming to a taken name answered %v, want Taken's 400", err)
	}
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

// BlockedBeatsEnabled restricts a user to the target's library, which it
// then sees, and blocks the same library, which hides it.
func (s *Server) BlockedBeatsEnabled(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	f := s.folder(t)
	s.Restrict(t, m, f)
	if s.Sees(t, m) == 0 {
		t.Fatalf("a user restricted to %s sees no video", s.tg.Library)
	}
	p, err := s.C.UserPolicy(t.Context(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := json.Marshal([]string{f})
	if err != nil {
		t.Fatal(err)
	}
	p["BlockedMediaFolders"] = blocked
	if err := s.C.SetPolicy(t.Context(), m.ID, p); err != nil {
		t.Fatal(err)
	}
	if n := s.Sees(t, m); n != 0 {
		t.Fatalf("a user with %s enabled and blocked sees %d videos", s.tg.Library, n)
	}
}

// UserDisabled marks a video played for a restricted user, then disables
// the user, which can't sign in and keeps its watch data, and restricts
// it again, which lets it sign in.
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
	if err := s.SignInAs(t, m.Name, m.Password); !refused(err) {
		t.Fatalf("the disabled user's sign-in answered %v, want a refusal", err)
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

// FailedSignInsLock records how many wrong passwords disable a user. It
// asserts nothing.
func (s *Server) FailedSignInsLock(t *testing.T) {
	m := s.NewMember(t)
	defer s.Drop(t, m)
	s.Restrict(t, m, s.folder(t))
	const tries = 10
	for n := range tries {
		if err := s.SignInAs(t, m.Name, "wrong-"+m.Password); err == nil {
			t.Fatal("a wrong password signed in")
		}
		if s.flags(t, m).IsDisabled {
			t.Logf("%d wrong passwords disabled the user", n+1)
			return
		}
	}
	t.Logf("%d wrong passwords left the user enabled", tries)
}
