package mediabrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// user is one server user. Its id is minted, so a user made again gets a
// new one.
type user struct {
	id, name, password string
	policy             policy
}

// policy is every policy field the fake renders, so posting back what a
// GET read changes nothing. EnabledFolders and BlockedMediaFolders hold a
// library's ItemId on Jellyfin and its Guid on Emby.
type policy struct {
	IsAdministrator            bool     `json:"IsAdministrator"`
	IsHidden                   bool     `json:"IsHidden"`
	IsDisabled                 bool     `json:"IsDisabled"`
	EnableAllFolders           bool     `json:"EnableAllFolders"`
	EnabledFolders             []string `json:"EnabledFolders"`
	BlockedMediaFolders        []string `json:"BlockedMediaFolders"`
	EnableMediaPlayback        bool     `json:"EnableMediaPlayback"`
	EnableRemoteAccess         bool     `json:"EnableRemoteAccess"`
	LoginAttemptsBeforeLockout int      `json:"LoginAttemptsBeforeLockout"`
	AuthenticationProviderID   string   `json:"AuthenticationProviderId,omitempty"`
	PasswordResetProviderID    string   `json:"PasswordResetProviderId,omitempty"`
}

// newPolicy is a new user's policy. Jellyfin 10.9 and later name the
// providers and refuse a policy without them; Emby has no such fields.
func (s *server) newPolicy(admin bool) policy {
	p := policy{
		IsAdministrator: admin, IsHidden: admin, EnableAllFolders: true, EnableMediaPlayback: true, EnableRemoteAccess: true,
		EnabledFolders: []string{}, BlockedMediaFolders: []string{}, LoginAttemptsBeforeLockout: -1,
	}
	if s.p.Flavor == Jellyfin {
		p.AuthenticationProviderID = "Jellyfin.Server.Implementations.Users.DefaultAuthenticationProvider"
		p.PasswordResetProviderID = "Jellyfin.Server.Implementations.Users.DefaultPasswordResetProvider"
	}
	return p
}

// addUser makes a user with a minted id. The caller holds s.mu.
func (s *server) addUser(name, password string, admin bool) *user {
	u := &user{id: s.mint("user"), name: name, password: password, policy: s.newPolicy(admin)}
	s.users = append(s.users, u)
	return u
}

// userAt is the user with id. The caller holds s.mu.
func (s *server) userAt(uid string) (*user, bool) {
	i := slices.IndexFunc(s.users, func(u *user) bool { return u.id == uid })
	if i < 0 {
		return nil, false
	}
	return s.users[i], true
}

// userNamed is the user with name in any case, other than except. The
// caller holds s.mu.
func (s *server) userNamed(name string, except *user) (*user, bool) {
	i := slices.IndexFunc(s.users, func(u *user) bool { return u != except && strings.EqualFold(u.name, name) })
	if i < 0 {
		return nil, false
	}
	return s.users[i], true
}

// owner is the first administrator, whose data a key's calls read. The
// caller holds s.mu.
func (s *server) owner() user {
	i := slices.IndexFunc(s.users, func(u *user) bool { return u.policy.IsAdministrator })
	if i < 0 {
		return user{}
	}
	return *s.users[i]
}

// folderID is how a policy names the library key: its ItemId on
// Jellyfin, its Guid on Emby.
func (s *server) folderID(library string) string {
	if s.p.Flavor == Emby {
		return libraryGUID(library)
	}
	return library
}

// sees is whether u may see the library key. BlockedMediaFolders wins
// over EnabledFolders and EnableAllFolders.
func (s *server) sees(u user, library string) bool {
	f := s.folderID(library)
	if slices.Contains(u.policy.BlockedMediaFolders, f) {
		return false
	}
	return u.policy.EnableAllFolders || slices.Contains(u.policy.EnabledFolders, f)
}

// taken is each product's 400 for a name another user has.
func taken(name string) string {
	return fmt.Sprintf("A user with the name '%s' already exists.", name)
}

func (s *server) renderUser(u user) map[string]any {
	p := u.policy
	p.EnabledFolders, p.BlockedMediaFolders = nonNilStrings(p.EnabledFolders), nonNilStrings(p.BlockedMediaFolders)
	return map[string]any{
		"Name": u.name, "ServerId": s.srv.ID, "Id": u.id, "HasPassword": u.password != "",
		"HasConfiguredPassword": u.password != "", "EnableAutoLogin": false, "Policy": p,
	}
}

func (s *server) listUsers(w http.ResponseWriter, _ *http.Request, _ caller) {
	s.mu.Lock()
	out := make([]map[string]any, len(s.users))
	for i, u := range s.users {
		out[i] = s.renderUser(*u)
	}
	s.mu.Unlock()
	writeJSON(w, out)
}

// usersChanged records the users and their policies after a change. The
// caller holds s.mu.
func (s *server) usersChanged(ctx context.Context) {
	rec := UserList{Users: make([]UserRecord, len(s.users))}
	for i, u := range s.users {
		rec.Users[i] = UserRecord{
			ID: u.id, Name: u.name, Administrator: u.policy.IsAdministrator, Disabled: u.policy.IsDisabled,
			AllFolders: u.policy.EnableAllFolders, Folders: nonNilStrings(u.policy.EnabledFolders),
			Blocked: nonNilStrings(u.policy.BlockedMediaFolders),
		}
	}
	s.userLog.Record(ctx, rec)
}

// UserList is a users record's data: every user after a change.
type UserList struct {
	Users []UserRecord `json:"users"`
}

type UserRecord struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Administrator bool     `json:"administrator"`
	Disabled      bool     `json:"disabled"`
	AllFolders    bool     `json:"allFolders"`
	Folders       []string `json:"folders"`
	Blocked       []string `json:"blocked"`
}

// createUser is POST /Users/New. Jellyfin sets the Password it is given,
// and Emby ignores it.
func (s *server) createUser(w http.ResponseWriter, r *http.Request, _ caller) {
	var in struct{ Name, Password string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if s.p.Flavor == Emby {
		in.Password = ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.userNamed(in.Name, nil); ok {
		http.Error(w, taken(in.Name), http.StatusBadRequest)
		return
	}
	u := s.addUser(in.Name, in.Password, false)
	s.usersChanged(r.Context())
	writeJSON(w, s.renderUser(*u))
}

// withUser runs h on the user the path's id names under s.mu, and
// answers 404 when there is none.
func (s *server) withUser(h func(http.ResponseWriter, *http.Request, *user)) func(http.ResponseWriter, *http.Request, caller) {
	return func(w http.ResponseWriter, r *http.Request, _ caller) {
		s.mu.Lock()
		defer s.mu.Unlock()
		u, ok := s.userAt(r.PathValue("id"))
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h(w, r, u)
	}
}

func (s *server) getUser(w http.ResponseWriter, _ *http.Request, u *user) {
	writeJSON(w, s.renderUser(*u))
}

// updateUser is POST /Users/{id}, which renames the user to the body's
// Name.
func (s *server) updateUser(w http.ResponseWriter, r *http.Request, u *user) {
	var in struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := s.userNamed(in.Name, u); ok {
		http.Error(w, taken(in.Name), http.StatusBadRequest)
		return
	}
	u.name = in.Name
	s.usersChanged(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// setPolicy is POST /Users/{id}/Policy, which replaces the whole policy.
// Jellyfin refuses one without its two provider ids.
func (s *server) setPolicy(w http.ResponseWriter, r *http.Request, u *user) {
	var p policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if s.p.Flavor == Jellyfin && (p.AuthenticationProviderID == "" || p.PasswordResetProviderID == "") {
		http.Error(w, "The AuthenticationProviderId and PasswordResetProviderId fields are required.", http.StatusBadRequest)
		return
	}
	u.policy = p
	s.usersChanged(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

type passwordChange struct {
	CurrentPw     string
	NewPw         string
	ResetPassword bool
}

// setPassword is Jellyfin's POST /Users/Password?userId=, which takes the
// new password from a key.
func (s *server) setPassword(w http.ResponseWriter, r *http.Request, _ caller) {
	var in passwordChange
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.userAt(r.URL.Query().Get("userId"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	u.password = in.NewPw
	if in.ResetPassword {
		u.password = ""
	}
	w.WriteHeader(http.StatusNoContent)
}

// setEmbyPassword is Emby's POST /Users/{id}/Password: ResetPassword
// empties the password, and a change needs the current one.
func (s *server) setEmbyPassword(w http.ResponseWriter, r *http.Request, u *user) {
	var in passwordChange
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch {
	case in.ResetPassword:
		u.password = ""
	case in.CurrentPw != u.password:
		http.Error(w, "Invalid user or password entered.", http.StatusBadRequest)
		return
	default:
		u.password = in.NewPw
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteUser is DELETE /Users/{id}, which ends the user's sessions.
func (s *server) deleteUser(w http.ResponseWriter, r *http.Request, u *user) {
	s.users = slices.DeleteFunc(s.users, func(x *user) bool { return x == u })
	for t, sess := range s.tokens {
		if sess.user == u.id {
			delete(s.tokens, t)
		}
	}
	s.usersChanged(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// FailNext is a fail-next cue's arguments: the route, by its name in the
// fake's routes, whose next call answers 500.
type FailNext struct {
	Route string `json:"route"`
}

// failing answers 500 to the one call after a fail-next cue named route.
func (s *server) failing(route string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		fail := s.failNext[route]
		delete(s.failNext, route)
		s.mu.Unlock()
		if fail {
			http.Error(w, "fail-next cue", http.StatusInternalServerError)
			return
		}
		h(w, r)
	}
}
