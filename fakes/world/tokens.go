package world

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// tokens are what Expand replaces: {{, a literal brace, and a {path}.
var tokens = regexp.MustCompile(`\{\{|\{[A-Za-z][\w.-]*\}`)

// Expand replaces each {path} in args with the manifest value at path,
// and each {{ with a literal {. A path walks the manifest's JSON by its
// keys and a list by its entries' names, as in {apiKey},
// {nodes.home.url} and {fakes.tmdb.url}. A path that names nothing, or
// names an object or a list, is refused.
func Expand(m Manifest, args []string) ([]string, error) {
	var tree any
	if err := json.Unmarshal(marshal(m), &tree); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	out := make([]string, len(args))
	var err error
	for i, a := range args {
		out[i] = tokens.ReplaceAllStringFunc(a, func(tok string) string {
			if tok == "{{" {
				return "{"
			}
			v, verr := lookup(tree, tok[1:len(tok)-1])
			if verr != nil && err == nil {
				err = verr
			}
			return v
		})
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func lookup(tree any, path string) (string, error) {
	at := tree
	for _, key := range strings.Split(path, ".") {
		switch node := at.(type) {
		case map[string]any:
			at = node[key]
		case []any:
			at = named(node, key)
		default:
			at = nil
		}
		if at == nil {
			return "", fmt.Errorf("{%s} names nothing in the manifest", path)
		}
	}
	switch v := at.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("{%s} names a list or an object, not a value", path)
	}
}

func named(list []any, name string) any {
	for _, e := range list {
		if entry, ok := e.(map[string]any); ok && entry["name"] == name {
			return entry
		}
	}
	return nil
}

// braces are what Literal escapes: a run of braces, which Expand halves,
// and a single brace that starts a {path}.
var braces = regexp.MustCompile(`\{\{+|\{[A-Za-z][\w.-]*\}`)

// Literal escapes s so that Expand gives it back unchanged.
func Literal(s string) string {
	return braces.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasPrefix(m, "{{") {
			return m + m
		}
		return "{" + m
	})
}

// StepOutputs are where a step's stdout and stderr are kept, relative to
// the run dir: journal/<n>.out and .err, or the same names under
// proof/<feature>/<sub> for a step with a proof dir.
func StepOutputs(n int, proof string) map[string]string {
	dir := "journal"
	if proof != "" {
		dir = "proof/" + proof
	}
	return map[string]string{
		"stdout": fmt.Sprintf("%s/%d.out", dir, n),
		"stderr": fmt.Sprintf("%s/%d.err", dir, n),
	}
}
