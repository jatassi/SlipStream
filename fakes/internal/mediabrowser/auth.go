package mediabrowser

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

var authField = regexp.MustCompile(`(\w+)="([^"]*)"`)

// authHeader parses an `<scheme> Name="value", ...` header, false when
// its scheme isn't scheme. The scheme is case-insensitive.
func authHeader(value, scheme string) (map[string]string, bool) {
	head, rest, ok := strings.Cut(strings.TrimSpace(value), " ")
	if !ok || !strings.EqualFold(head, scheme) {
		return nil, false
	}
	out := map[string]string{}
	for _, m := range authField.FindAllStringSubmatch(rest, -1) {
		out[m[1]] = m[2]
	}
	return out, true
}

// client is what a request says about its app: the Authorization fields
// a product reads. Jellyfin 12 reads only Authorization with the
// MediaBrowser scheme, since legacy authorization is off by default and
// forced off by a 12.x migration; Emby reads X-Emby-Authorization or
// Authorization with its own scheme.
func (s *server) client(r *http.Request) map[string]string {
	if s.p.Flavor == Jellyfin {
		f, _ := authHeader(r.Header.Get("Authorization"), "MediaBrowser")
		return f
	}
	for _, h := range []string{"X-Emby-Authorization", "Authorization"} {
		if f, ok := authHeader(r.Header.Get(h), "Emby"); ok {
			return f
		}
	}
	return nil
}

// token is the credential a request carries, each product's way.
// Jellyfin: the Authorization Token field or ?ApiKey=. Emby: X-Emby-Token,
// ?api_key= or the authorization header's Token.
func (s *server) token(r *http.Request) string {
	q := r.URL.Query()
	if s.p.Flavor == Jellyfin {
		if t := s.client(r)["Token"]; t != "" {
			return t
		}
		return q.Get("ApiKey")
	}
	for _, t := range []string{r.Header.Get("X-Emby-Token"), q.Get("api_key"), q.Get("X-Emby-Token"), s.client(r)["Token"]} {
		if t != "" {
			return t
		}
	}
	return ""
}

// caller is who a request is: an API key, which stands for the server and
// passes every elevation check, or a user's token.
type caller struct {
	key   bool
	user  string
	admin bool
}

func (s *server) caller(r *http.Request) (caller, bool) {
	t := s.token(r)
	if t == "" {
		return caller{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.keys, t) {
		return caller{key: true, admin: true}, true
	}
	sess, ok := s.tokens[t]
	if !ok {
		return caller{}, false
	}
	u := slices.IndexFunc(s.srv.Users, func(u User) bool { return u.Name == sess.user })
	return caller{user: sess.user, admin: u >= 0 && s.srv.Users[u].Admin}, true
}

// level is what a route needs.
type level int

const (
	public level = iota
	signedIn
	elevated
)

// guard answers 401 without a credential the server issued, and 403 to a
// user who isn't an administrator on an elevated route. A world with no
// server answers 404 everywhere.
func (s *server) guard(need level, h func(http.ResponseWriter, *http.Request, caller)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.srv == nil {
			http.NotFound(w, r)
			return
		}
		if need == public {
			h(w, r, caller{})
			return
		}
		c, ok := s.caller(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if need == elevated && !c.admin {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h(w, r, c)
	}
}

// authenticate is POST /Users/AuthenticateByName. A second sign-in with
// one DeviceId revokes the token the first got (#71 delta 13).
func (s *server) authenticate(w http.ResponseWriter, r *http.Request, _ caller) {
	app := s.client(r)
	if app["DeviceId"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var in struct{ Username, Pw string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	i := slices.IndexFunc(s.srv.Users, func(u User) bool { return strings.EqualFold(u.Name, in.Username) && u.Password == in.Pw })
	if i < 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	u := s.srv.Users[i]
	s.mu.Lock()
	for t, sess := range s.tokens {
		if sess.device == app["DeviceId"] {
			delete(s.tokens, t)
		}
	}
	token := s.mint()
	s.tokens[token] = session{user: u.Name, device: app["DeviceId"]}
	s.mu.Unlock()
	writeJSON(w, map[string]any{
		"User": s.renderUser(u), "AccessToken": token, "ServerId": s.srv.ID,
		"SessionInfo": map[string]any{
			"Id": id("session " + token), "UserId": userID(u.Name), "UserName": u.Name, "Client": app["Client"],
			"DeviceName": app["Device"], "DeviceId": app["DeviceId"], "ApplicationVersion": app["Version"], "IsActive": true,
		},
	})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request, _ caller) {
	s.mu.Lock()
	delete(s.tokens, s.token(r))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) renderUser(u User) map[string]any {
	return map[string]any{
		"Name": u.Name, "ServerId": s.srv.ID, "Id": userID(u.Name), "HasPassword": u.Password != "",
		"HasConfiguredPassword": u.Password != "", "EnableAutoLogin": false,
		"Policy": map[string]any{"IsAdministrator": u.Admin, "IsHidden": u.Admin, "IsDisabled": false, "EnableAllFolders": true},
	}
}

func (s *server) users(w http.ResponseWriter, _ *http.Request, _ caller) {
	out := make([]map[string]any, len(s.srv.Users))
	for i, u := range s.srv.Users {
		out[i] = s.renderUser(u)
	}
	writeJSON(w, out)
}

// createKey is POST /Auth/Keys, answered 204 with no body: the key is
// read back from the list.
func (s *server) createKey(w http.ResponseWriter, r *http.Request, _ caller) {
	app := r.URL.Query().Get("app")
	if s.p.Flavor == Emby {
		app = r.URL.Query().Get("App")
	}
	if app == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	key := s.mint()
	s.keys = append(s.keys, key)
	s.keyApps = append(s.keyApps, keyApp{key: key, app: app, created: time.Now()})
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

type keyApp struct {
	key, app string
	created  time.Time
}

func (s *server) listKeys(w http.ResponseWriter, _ *http.Request, _ caller) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []map[string]any{}
	for _, k := range s.keys {
		app, created := "", time.Time{}
		if i := slices.IndexFunc(s.keyApps, func(a keyApp) bool { return a.key == k }); i >= 0 {
			app, created = s.keyApps[i].app, s.keyApps[i].created
		}
		items = append(items, map[string]any{"AccessToken": k, "AppName": app, "DateCreated": created, "IsActive": true})
	}
	writeJSON(w, map[string]any{"Items": items, "TotalRecordCount": len(items), "StartIndex": 0})
}

func (s *server) revokeKey(w http.ResponseWriter, r *http.Request, _ caller) {
	key := r.PathValue("key")
	s.mu.Lock()
	s.keys = slices.DeleteFunc(s.keys, func(k string) bool { return k == key })
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
