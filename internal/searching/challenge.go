package searching

import (
	"net/http"
	"slices"
	"strings"
)

// challengeServers are the Server headers of the services that put a
// browser challenge in front of a site.
var challengeServers = []string{"cloudflare", "cloudflare-nginx", "ddos-guard"}

// challengeMarkers are what a challenge page holds, as Prowlarr reads it.
var challengeMarkers = []string{"<title>Just a moment...</title>", "<title>Access denied</title>", "<title>Attention Required! | Cloudflare</title>"}

// challenged reports whether res is a browser challenge: a 403 or 503 from
// a challenge service whose page holds one of its markers.
func challenged(res *http.Response, body []byte) bool {
	if !slices.Contains(challengeServers, strings.ToLower(res.Header.Get("Server"))) {
		return false
	}
	if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusServiceUnavailable {
		return false
	}
	page := string(body)
	return slices.ContainsFunc(challengeMarkers, func(m string) bool { return strings.Contains(page, m) }) ||
		strings.TrimSpace(page) == "error code: 1020" || strings.Contains(strings.ToLower(page), "<title>ddos-guard</title>")
}
