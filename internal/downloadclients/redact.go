package downloadclients

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const hidden = "•••"

var (
	hex32       = regexp.MustCompile(`^[0-9A-Fa-f]{32}$`)
	secretQuery = []string{"passkey", "pk", "authkey", "torrent_pass"}
)

// Redact hides a tracker URL's credentials and keeps its host readable:
// every path segment after announce, every 32-hex path segment, the
// password of its userinfo and the values of its passkey query keys. Text
// that is not a URL is hidden whole.
func Redact(tracker string) string {
	u, err := url.Parse(tracker)
	if err != nil || u.Host == "" {
		return hidden
	}
	var b strings.Builder
	b.WriteString(u.Scheme + "://")
	if u.User != nil {
		b.WriteString(u.User.Username())
		if _, ok := u.User.Password(); ok {
			b.WriteString(":" + hidden)
		}
		b.WriteString("@")
	}
	b.WriteString(u.Host)
	b.WriteString(redactPath(u.EscapedPath()))
	if u.RawQuery != "" {
		b.WriteString("?" + redactQuery(u.RawQuery))
	}
	return b.String()
}

func redactPath(path string) string {
	segments := strings.Split(path, "/")
	afterAnnounce := false
	for i, s := range segments {
		if afterAnnounce && s != "" || hex32.MatchString(s) {
			segments[i] = hidden
		}
		afterAnnounce = afterAnnounce || s == "announce"
	}
	return strings.Join(segments, "/")
}

func redactQuery(query string) string {
	pairs := strings.Split(query, "&")
	for i, p := range pairs {
		key, _, _ := strings.Cut(p, "=")
		if k, err := url.QueryUnescape(key); err == nil && slices.Contains(secretQuery, strings.ToLower(k)) {
			pairs[i] = key + "=" + hidden
		}
	}
	return strings.Join(pairs, "&")
}
