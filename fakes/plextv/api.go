package plextv

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
)

func (s *state) routes() fake.Routes {
	return fake.Routes{
		"pins":      {Pattern: "POST /api/v2/pins", Handler: s.createPin},
		"pin":       {Pattern: "GET /api/v2/pins/{id}", Handler: s.pollPin},
		"user":      {Pattern: "GET /api/v2/user", Handler: s.user},
		"resources": {Pattern: "GET /api/v2/resources", Handler: s.resources},
		"nonce":     {Pattern: "GET /api/v2/auth/nonce", Handler: s.nonce},
		"token":     {Pattern: "POST /api/v2/auth/token", Handler: s.renew},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(mustJSON(v))
}

// writeError is plex.tv's error body. The recorded run met 1001 on a 401.
// The other codes and messages follow the same shape unrecorded.
func writeError(w http.ResponseWriter, status, code int, message string) {
	writeJSON(w, status, map[string]any{"errors": []map[string]any{{"code": code, "message": message, "status": status}}})
}

func unauthenticated(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, 1001, "User could not be authenticated")
}

func (s *state) createPin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JWK    jwk  `json:"jwk"`
		Strong bool `json:"strong"`
	}
	clientID := r.Header.Get("X-Plex-Client-Identifier")
	if clientID == "" {
		writeError(w, http.StatusBadRequest, 1000, "X-Plex-Client-Identifier is missing")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.JWK.parse() != nil {
		writeError(w, http.StatusBadRequest, 1000, "jwk is not an Ed25519 key")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	length := 4
	if body.Strong {
		length = strongCode
	}
	p := &pin{
		id: 1_000_000_000 + s.draw.Int64N(1_000_000_000), code: s.word(length), clientID: clientID,
		product: r.Header.Get("X-Plex-Product"), jwk: body.JWK, created: now, expires: now.Add(pinLife),
	}
	s.pins[p.id] = p
	writeJSON(w, http.StatusCreated, p.wire(now))
}

// wire is p as the recorded run's PIN bodies carry it.
func (p *pin) wire(now time.Time) map[string]any {
	var token any
	if p.token != "" {
		token = p.token
	}
	return map[string]any{
		"id": p.id, "code": p.code, "product": p.product, "trusted": false, "qr": "https://plex.invalid/api/v2/pins/qr/" + p.code,
		"clientIdentifier": p.clientID, "expiresIn": int(p.expires.Sub(now).Seconds()),
		"createdAt": p.created.UTC().Format(time.RFC3339), "expiresAt": p.expires.UTC().Format(time.RFC3339),
		"authToken": token, "newRegistration": nil,
	}
}

// pollPin answers a PIN whose device JWT its own key signed. A gone PIN's
// 404 is plex.tv's as PLEXAPI meets it, unrecorded.
func (s *state) pollPin(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	p, ok := s.pins[id]
	if !ok || !now.Before(p.expires) {
		writeError(w, http.StatusNotFound, 1020, "Code not found or expired")
		return
	}
	if _, err := verifyDevice(r.URL.Query().Get("deviceJWT"), p.jwk.key, p.clientID, now); err != nil {
		unauthenticated(w)
		return
	}
	writeJSON(w, http.StatusOK, p.wire(now))
}

// signedIn is the device of the Plex JWT r carries, while it lives and
// the device isn't signed out. s.mu is held.
func (s *state) signedIn(r *http.Request) (string, *device, bool) {
	token := r.Header.Get("X-Plex-Token")
	if token == "" {
		token = r.URL.Query().Get("X-Plex-Token")
	}
	is, ok := s.issued[token]
	if !ok || is.device.revoked || !time.Now().Before(is.expires) {
		return "", nil, false
	}
	return token, is.device, true
}

// subscriptions are the recorded run's subscription with Plex Pass, and
// the shape PLEXAPI reads without it, unrecorded.
var subscriptions = map[bool]map[string]any{
	true: {
		"active": true, "subscribedAt": "2021-03-24T16:36:44Z", "status": "Active", "paymentService": "braintree", "plan": "yearly",
		"features": []string{"adaptive_bitrate", "camera_upload", "cloudsync", "webhooks"},
	},
	false: {"active": false, "subscribedAt": nil, "status": "Inactive", "paymentService": nil, "plan": nil, "features": []string{}},
}

func (s *state) user(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, _, ok := s.signedIn(r)
	if !ok {
		unauthenticated(w)
		return
	}
	a := s.account
	writeJSON(w, http.StatusOK, map[string]any{
		"id": 1, "uuid": "fake-plex-tv-account", "username": a.Username, "title": a.Username, "email": a.Username + "@plex.invalid",
		"friendlyName": a.Username, "locale": nil, "confirmed": true, "joinedAt": 1616603804, "emailOnlyAuth": false, "hasPassword": true,
		"protected": true, "thumb": "https://plex.invalid/users/fake/avatar", "authToken": token, "subscription": subscriptions[a.PlexPass],
		"restricted": false, "anonymous": false, "home": true, "guest": false, "homeAdmin": true, "twoFactorEnabled": true,
	})
}

// resources lists each fake-plex server the account owns, with one local
// connection per Node front in the fronts' order and the server token
// this device holds for it. Fronts answer every dialer, so a Node keeps
// the connection a Scenario pins for it.
func (s *state) resources(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.signedIn(r)
	if !ok {
		unauthenticated(w)
		return
	}
	relay := s.account.Relay && r.URL.Query().Get("includeRelay") == "1"
	out := []map[string]any{}
	for _, srv := range s.owned.Servers() {
		token, ok := d.tokens[srv.Identifier]
		if !ok {
			token = s.word(20)
			d.tokens[srv.Identifier] = token
			s.owned.Grant(srv.Identifier, token)
		}
		connections := []map[string]any{}
		for _, f := range s.fronts {
			if f.Target != srv.Target || f.Node == "world" {
				continue
			}
			connections = append(connections, map[string]any{
				"protocol": "http", "address": f.Address.AddrPort.Addr().String(), "port": f.Address.AddrPort.Port(),
				"uri": f.Address.URL(""), "local": true, "relay": false, "IPv6": false,
			})
		}
		if relay {
			connections = append(connections, map[string]any{
				"protocol": "https", "address": "203.0.113.7", "port": 8443, "uri": "https://203-0-113-7.relay.plex.invalid:8443",
				"local": false, "relay": true, "IPv6": false,
			})
		}
		out = append(out, map[string]any{
			"name": srv.Name, "product": "Plex Media Server", "productVersion": srv.Version, "clientIdentifier": srv.Identifier,
			"provides": "server", "ownerId": nil, "sourceTitle": nil, "publicAddress": "127.0.0.1", "accessToken": token,
			"searchEnabled": true, "createdAt": "2021-06-12T00:38:10Z", "lastSeenAt": time.Now().UTC().Format(time.RFC3339),
			"owned": true, "home": false, "synced": false, "relay": relay, "presence": false, "httpsRequired": false,
			"publicAddressMatches": true, "dnsRebindingProtection": false, "natLoopbackSupported": true, "connections": connections,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *state) nonce(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.word(8) + "-" + s.word(4) + "-" + s.word(4) + "-" + s.word(4) + "-" + s.word(12)
	s.nonces[n] = true
	writeJSON(w, http.StatusOK, map[string]string{"nonce": n})
}

// renew trades a device JWT with a fresh nonce for a new Plex JWT, the
// last one expired or not. A device signed out gets 422, as Plex's word
// on renewal has it, with an unrecorded body.
func (s *state) renew(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JWT string `json:"jwt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, 1000, "jwt is missing")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	d, ok := s.devices[kidOf(body.JWT)]
	if !ok {
		unauthenticated(w)
		return
	}
	c, err := verifyDevice(body.JWT, d.key, d.clientID, now)
	if err != nil || !s.nonces[c.Nonce] {
		unauthenticated(w)
		return
	}
	delete(s.nonces, c.Nonce)
	if d.revoked {
		writeError(w, http.StatusUnprocessableEntity, 1002, "Device has been removed from the account")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"auth_token": s.mint(d, now)})
}
