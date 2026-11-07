// Ported from Prowlarr's src/NzbDrone.Core/Parser/StringUtil.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package stringutil is Prowlarr's StringUtil.GetQueryString over .NET's
// NameValueCollection: the query string most HTML sites build.
package stringutil

import (
	"strings"

	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
)

// NameValueCollection is .NET's: names in the order first added, matched
// without regard to case, and a name's values read joined by commas. The
// zero value is empty.
type NameValueCollection struct {
	names  []string
	values [][]string
}

func (c *NameValueCollection) index(name string) int {
	for i, n := range c.names {
		if strings.EqualFold(n, name) {
			return i
		}
	}
	return -1
}

func (c *NameValueCollection) Add(name, value string) {
	if i := c.index(name); i >= 0 {
		c.values[i] = append(c.values[i], value)
		return
	}
	c.names = append(c.names, name)
	c.values = append(c.values, []string{value})
}

// Set makes value name's only value.
func (c *NameValueCollection) Set(name, value string) {
	if i := c.index(name); i >= 0 {
		c.values[i] = []string{value}
		return
	}
	c.Add(name, value)
}

// Get is name's values joined by commas, "" when name isn't there.
func (c *NameValueCollection) Get(name string) string {
	if i := c.index(name); i >= 0 {
		return strings.Join(c.values[i], ",")
	}
	return ""
}

func (c *NameValueCollection) Count() int { return len(c.names) }

// QueryString are GetQueryString's optional parameters. The zero value
// is C#'s defaults: UTF-8, one pair per name, joined by "&".
type QueryString struct {
	Encoding encoding.Charset
	// DuplicateKeysIfMulti writes a pair for each comma-separated part of
	// a name's values.
	DuplicateKeysIfMulti bool
	Separator            string
}

// GetQueryString is GetQueryString() with C#'s defaults: name=value
// pairs joined by "&", a repeated name's values joined by commas, each
// value URL-encoded as UTF-8.
func (c *NameValueCollection) GetQueryString() string { return c.GetQueryStringWith(QueryString{}) }

// GetQueryStringWith is GetQueryString with its optional parameters.
func (c *NameValueCollection) GetQueryStringWith(o QueryString) string {
	separator := o.Separator
	if separator == "" {
		separator = "&"
	}
	var pairs []string
	for i, name := range c.names {
		value := strings.Join(c.values[i], ",")
		parts := []string{value}
		if o.DuplicateKeysIfMulti {
			parts = strings.Split(value, ",")
		}
		for _, v := range parts {
			pairs = append(pairs, name+"="+o.Encoding.URLEncode(v))
		}
	}
	return strings.Join(pairs, separator)
}
