package cardigann

import (
	"strings"

	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
)

// pair is one key and value of a form or query string, kept in order.
type pair struct {
	Key, Value string
}

// queryString is Prowlarr's GetQueryString: key=value pairs in order joined
// by sep, each value URL-encoded in the charset and each key as written.
func queryString(c encoding.Charset, pairs []pair, sep string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p.Key + "=" + c.URLEncode(p.Value)
	}
	return strings.Join(parts, sep)
}
