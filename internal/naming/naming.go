// Package naming turns a Content Type's tokens into the folder and file
// names a file is placed under. One sanitizer serves every OS, so a library
// made on Linux opens on Windows.
package naming

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
// {Edition Tag} and {Edition}. {Token:00} pads a whole number to the
// pattern's width.
type Templates struct {
	Folder string
	// Levels is a folder template per level, outermost first, each its
	// own path segment between the Work's folder and the file. Empty puts
	// files straight in the Work's folder.
	Levels []string
	// Aside replaces the outermost level's template for an Item in an
	// aside level.
	Aside string
	File  string
}

// Core is the tokens the core supplies: Target is the first Target in
// Profile order the file fills; Edition is the file's cut by name and
// EditionTag is it as {edition-<name>}, both empty for Standard.
type Core struct {
	Target     string
	EditionTag string
	Edition    string
}

var (
	token = regexp.MustCompile(`\{([^{}]*)\}`)
	pad   = regexp.MustCompile(`^0+$`)
)

// Render fills template's tokens and sanitizes the result into one path
// segment. A token neither tokens nor core holds is an error, as is a
// padding that isn't all zeros or pads a token that isn't a whole number,
// and a name that sanitizes to nothing.
func Render(template string, tokens map[string]string, core Core) (string, error) {
	values := map[string]string{"Target": core.Target, "Edition Tag": core.EditionTag, "Edition": core.Edition}
	for k, v := range tokens {
		values[k] = v
	}
	var missing, unpadded []string
	filled := token.ReplaceAllStringFunc(template, func(m string) string {
		name, width, padded := strings.Cut(m[1:len(m)-1], ":")
		v, ok := values[name]
		switch {
		case !ok:
			missing = append(missing, m)
		case padded:
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil || !pad.MatchString(width) {
				unpadded = append(unpadded, m)
				return v
			}
			return fmt.Sprintf("%0*d", len(width), n)
		}
		return v
	})
	if missing != nil {
		return "", fmt.Errorf("template %q names %s, which no token fills", template, strings.Join(missing, ", "))
	}
	if unpadded != nil {
		return "", fmt.Errorf("template %q pads %s, which needs a whole number and a pattern of zeros", template, strings.Join(unpadded, ", "))
	}
	name := Sanitize(filled)
	if name == "" {
		return "", fmt.Errorf("template %q renders to an empty name", template)
	}
	return name, nil
}

// Folders renders the level folders of an Item with tokens, outermost
// first: none when on is false or t names no level, and t.Aside in place
// of the outermost level's template when aside and t has one.
func Folders(t Templates, tokens map[string]string, aside, on bool) ([]string, error) {
	if !on || len(t.Levels) == 0 {
		return nil, nil
	}
	out := make([]string, len(t.Levels))
	for i, template := range t.Levels {
		if i == 0 && aside && t.Aside != "" {
			template = t.Aside
		}
		var err error
		if out[i], err = Render(template, tokens, Core{}); err != nil {
			return nil, err
		}
	}
	return out, nil
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
