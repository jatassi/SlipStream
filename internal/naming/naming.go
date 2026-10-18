// Package naming turns a Content Type's tokens into the folder and file
// names a file is placed under. One sanitizer serves every OS, so a library
// made on Linux opens on Windows.
package naming

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is how a Content Type names its files. A type that names nothing
// answers no tokens and empty Templates.
type Facet interface {
	NamingTokens(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (map[string]string, error)
	DefaultTemplates() Templates
}

// Templates hold {Token}s: the type's own, and the core's {Target},
// {Edition Tag} and {Edition}.
type Templates struct {
	Folder string
	File   string
}

// Core is the tokens the core supplies: Target is the first Target in
// Profile order the file fills; Edition is the file's cut by name and
// EditionTag is it as {edition-<name>}, both empty for Standard.
type Core struct {
	Target     string
	EditionTag string
	Edition    string
}

var token = regexp.MustCompile(`\{([^{}]*)\}`)

// Render fills template's tokens and sanitizes the result into one path
// segment. A token neither tokens nor core holds is an error, as is a
// name that sanitizes to nothing.
func Render(template string, tokens map[string]string, core Core) (string, error) {
	values := map[string]string{"Target": core.Target, "Edition Tag": core.EditionTag, "Edition": core.Edition}
	for k, v := range tokens {
		values[k] = v
	}
	var missing []string
	filled := token.ReplaceAllStringFunc(template, func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := values[name]
		if !ok {
			missing = append(missing, m)
		}
		return v
	})
	if missing != nil {
		return "", fmt.Errorf("template %q names %s, which no token fills", template, strings.Join(missing, ", "))
	}
	name := Sanitize(filled)
	if name == "" {
		return "", fmt.Errorf("template %q renders to an empty name", template)
	}
	return name, nil
}

// Sanitize makes s one path segment valid on every OS: < > : " / \ | ? *
// and control characters become spaces, runs of spaces collapse to one,
// leading spaces and trailing dots and spaces go, and a Windows reserved
// device name gains a leading _.
func Sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimRight(s, ". ")
	if reserved(s) {
		s = "_" + s
	}
	return s
}

var devices = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])$`)

// reserved holds whatever follows the device name, as Windows does:
// NUL.txt and NUL .txt open the device too.
func reserved(s string) bool {
	base, _, _ := strings.Cut(s, ".")
	return devices.MatchString(strings.TrimRight(base, " "))
}
