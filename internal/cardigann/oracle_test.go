package cardigann

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/dlclark/regexp2"
)

// prowlarrTemplate is a line-for-line port of Prowlarr's multi-pass
// ApplyGoTemplateText (CardigannBase.cs:363-602), kept as an oracle for the
// parse-once evaluator. A nil value in vars is C#'s null and a []string is a
// collection.
func prowlarrTemplate(template string, vars map[string]any) (string, error) {
	if strings.TrimSpace(template) == "" || !strings.Contains(template, "{{") {
		return template, nil
	}
	lookup := func(name string) (any, error) {
		v, ok := vars[name]
		if !ok {
			return nil, fmt.Errorf("the key %s isn't present", name)
		}
		return v, nil
	}
	var err error
	for _, pass := range []func(string, func(string) (any, error)) (string, error){oracleReReplace, oracleJoin, oracleLogic, oracleIfElse, oracleRange, oracleVariables} {
		if template, err = pass(template, lookup); err != nil {
			return "", err
		}
	}
	return template, nil
}

func oracleMatches(re *regexp2.Regexp, s string) []*regexp2.Match {
	var out []*regexp2.Match
	m, _ := re.FindStringMatch(s)
	for m != nil {
		out = append(out, m)
		m, _ = re.FindNextMatch(m)
	}
	return out
}

var (
	oracleReReplaceRe = regexp2.MustCompile(`{{\s*re_replace\s+(\..+?)\s+"(.*?)"\s+"(.*?)"\s*}}`, regexp2.None)
	oracleJoinRe      = regexp2.MustCompile(`{{\s*join\s+(\..+?)\s+"(.*?)"\s*}}`, regexp2.None)
	oracleLogicRe     = regexp2.MustCompile(`\b(and|or|eq|ne)(?:\s+(\(?\.[^\)\s]+\)?|"[^"]+")){2,}`, regexp2.None)
	oracleIfElseRe    = regexp2.MustCompile(`{{\s*if\s*(.+?)\s*}}(.*?){{\s*else\s*}}(.*?){{\s*end\s*}}`, regexp2.None)
	oracleRangeRe     = regexp2.MustCompile(`{{\s*range\s*(((?<index>\$.+?),)((\s*(?<element>.+?)\s*(:=)\s*)))?(?<variable>.+?)\s*}}(?<prefix>.*?){{\.}}(?<postfix>.*?){{end}}`, regexp2.None)
	oracleVariableRe  = regexp2.MustCompile(`{{\s*(\..+?)\s*}}`, regexp2.None)
)

func oracleString(v any) (string, error) {
	switch v := v.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	}
	return "", errors.New("a collection where a string belongs")
}

func oracleReReplace(template string, lookup func(string) (any, error)) (string, error) {
	for _, m := range oracleMatches(oracleReReplaceRe, template) {
		v, err := lookup(m.GroupByNumber(1).String())
		if err != nil {
			return "", err
		}
		input, err := oracleString(v)
		if err != nil {
			return "", err
		}
		re, err := compileRegex(m.GroupByNumber(2).String())
		if err != nil {
			return "", err
		}
		out, err := re.Replace(input, m.GroupByNumber(3).String())
		if err != nil {
			return "", err
		}
		template = strings.ReplaceAll(template, m.String(), out)
	}
	return template, nil
}

func oracleJoin(template string, lookup func(string) (any, error)) (string, error) {
	for _, m := range oracleMatches(oracleJoinRe, template) {
		v, err := lookup(m.GroupByNumber(1).String())
		if err != nil {
			return "", err
		}
		list, ok := v.([]string)
		if !ok {
			return "", errors.New("join over a value that isn't a collection")
		}
		template = strings.ReplaceAll(template, m.String(), strings.Join(list, m.GroupByNumber(2).String()))
	}
	return template, nil
}

func oracleLogic(template string, lookup func(string) (any, error)) (string, error) {
	for {
		m, _ := oracleLogicRe.FindStringMatch(template)
		if m == nil {
			return template, nil
		}
		start, length := m.Index, m.Length
		name := m.GroupByNumber(1).String()
		var params []string
		for _, c := range m.GroupByNumber(2).Captures {
			params = append(params, strings.Trim(c.String(), "()"))
		}
		var result string
		var err error
		if name == "and" || name == "or" {
			result, err = oracleAndOr(name == "and", params, lookup)
		} else {
			if len(params) > 2 {
				length = m.GroupByNumber(2).Captures[2].Index - start
			}
			result, err = oracleEq(name == "eq", params[:2], lookup)
		}
		if err != nil {
			return "", err
		}
		runes := []rune(template)
		template = string(runes[:start]) + result + string(runes[start+length:])
	}
}

func oracleAndOr(isAnd bool, params []string, lookup func(string) (any, error)) (string, error) {
	result := ""
	for _, p := range params {
		if strings.HasPrefix(p, `"`) {
			continue
		}
		result = p
		v, err := lookup(p)
		if err != nil {
			return "", err
		}
		s, _ := v.(string)
		if (strings.TrimSpace(s) == "") == isAnd {
			break
		}
	}
	return result, nil
}

func oracleEq(wantEqual bool, params []string, lookup func(string) (any, error)) (string, error) {
	values := make([]any, 2)
	for i, p := range params {
		if strings.HasPrefix(p, `"`) {
			values[i] = strings.Trim(p, `"`)
			continue
		}
		v, err := lookup(p)
		if err != nil {
			return "", err
		}
		if s, ok := v.(string); ok {
			values[i] = s
		}
	}
	if (values[0] == values[1]) == wantEqual {
		return ".True", nil
	}
	return ".False", nil
}

func oracleIfElse(template string, lookup func(string) (any, error)) (string, error) {
	for _, m := range oracleMatches(oracleIfElseRe, template) {
		cond := m.GroupByNumber(1).String()
		if !strings.HasPrefix(cond, ".") {
			return "", fmt.Errorf("condition %q not implemented", cond)
		}
		v, err := lookup(cond)
		if err != nil {
			return "", err
		}
		state := false
		switch v := v.(type) {
		case string:
			state = strings.TrimSpace(v) != ""
		case []string:
			state = len(v) > 0
		}
		result := m.GroupByNumber(3).String()
		if state {
			result = m.GroupByNumber(2).String()
		}
		template = strings.ReplaceAll(template, m.String(), result)
	}
	return template, nil
}

func oracleRange(template string, lookup func(string) (any, error)) (string, error) {
	for _, m := range oracleMatches(oracleRangeRe, template) {
		index := m.GroupByName("index").String()
		v, err := lookup(m.GroupByName("variable").String())
		if err != nil {
			return "", err
		}
		list, ok := v.([]string)
		if !ok {
			return "", errors.New("range over a value that isn't a collection")
		}
		prefix, postfix := m.GroupByName("prefix").String(), m.GroupByName("postfix").String()
		var b strings.Builder
		for i, item := range list {
			if strings.TrimSpace(index) != "" {
				ir := "{{" + index + "}}"
				b.WriteString(strings.ReplaceAll(prefix, ir, strconv.Itoa(i)) + item + strings.ReplaceAll(postfix, ir, strconv.Itoa(i)))
			} else {
				b.WriteString(prefix + item + postfix)
			}
		}
		template = strings.ReplaceAll(template, m.String(), b.String())
	}
	return template, nil
}

func oracleVariables(template string, lookup func(string) (any, error)) (string, error) {
	for _, m := range oracleMatches(oracleVariableRe, template) {
		v, err := lookup(m.GroupByNumber(1).String())
		if err != nil {
			return "", err
		}
		s := "System.String[]"
		if str, ok := v.(string); ok || v == nil {
			s = str
		}
		template = strings.ReplaceAll(template, m.String(), s)
	}
	return template, nil
}

func TestTemplateOracleFixture(t *testing.T) {
	vars := map[string]any{
		".Config.sitelink": "https://somesite.com/", ".True": "True", ".False": nil,
		".Categories": []string{"tv", "movies"}, ".Query.Keywords": "abc def", ".Config.x": "1",
	}
	sc := scope{}
	for k, v := range vars {
		switch v := v.(type) {
		case string:
			sc[k] = textValue(v)
		case []string:
			sc[k] = listValue(v)
		default:
			sc[k] = nullValue
		}
	}
	for _, src := range []string{
		"{{ range .Categories}}&categories[]={{.}}{{end}}",
		"{{ range $i, $e := .Categories}}&categories[{{$i}}]={{.}}{{end}}",
		`{{ re_replace .Query.Keywords "[^a-zA-Z0-9]+" "%" }}`,
		`{{ join .Categories ", " }}`,
		"{{if .False }}0{{else}}1{{end}}",
		`{{ if and .True .Config.x }}y{{ else }}n{{ end }}`,
		`{{ if or .False .True }}y{{ else }}n{{ end }}`,
		`{{ if eq .Config.x "1" }}y{{ else }}n{{ end }}`,
		`{{ if ne .Config.x "1" }}y{{ else }}n{{ end }}`,
		`{{ or .False .Config.sitelink }}`,
	} {
		want, err := prowlarrTemplate(src, vars)
		if err != nil {
			t.Fatalf("oracle %q: %v", src, err)
		}
		tmpl, err := parseTemplate(src)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := tmpl.Eval(sc, nil); err != nil || got != want {
			t.Errorf("%q = %q, %v; Prowlarr gives %q", src, got, err, want)
		}
	}
}

// TestTemplateOracleCatalog renders every template in the catalog with both
// evaluators under a few variable assignments, and fails where they differ.
func TestTemplateOracleCatalog(t *testing.T) {
	dir := os.Getenv("SLIPSTREAM_CARDIGANN_CATALOG")
	if dir == "" {
		t.Skip("SLIPSTREAM_CARDIGANN_CATALOG names no catalog directory")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	seen := map[string]bool{}
	compared, differ := 0, 0
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Load(src)
		if err != nil {
			continue
		}
		for _, tmpl := range templatesIn(reflect.ValueOf(d.def).Elem()) {
			if seen[tmpl.Source()] {
				continue
			}
			seen[tmpl.Source()] = true
			for _, assign := range oracleAssignments(tmpl) {
				compared++
				want, wantErr := prowlarrTemplate(tmpl.Source(), assign.vars)
				got, gotErr := tmpl.Eval(assign.scope, nil)
				if (wantErr != nil) != (gotErr != nil) || (wantErr == nil && got != want) {
					differ++
					t.Errorf("%s %s: %q\n  ours:     %q %v\n  Prowlarr: %q %v", filepath.Base(f), assign.name, tmpl.Source(), got, gotErr, want, wantErr)
				}
			}
		}
	}
	t.Logf("%d renders of %d templates, %d differ", compared, len(seen), differ)
}

type oracleAssignment struct {
	name  string
	vars  map[string]any
	scope scope
}

// oracleAssignments set every variable the template reads to the same
// text, to null, to blank text and to its own name, with ranged and joined
// names as lists.
func oracleAssignments(tmpl *template) []oracleAssignment {
	lists := listNames(tmpl.nodes)
	var out []oracleAssignment
	for _, a := range []struct {
		name, text string
		null       bool
	}{{"text", "v1 x", false}, {"null", "", true}, {"blank", " ", false}, {"distinct", "", false}} {
		o := oracleAssignment{name: a.name, vars: map[string]any{".True": "True", ".False": nil}, scope: scope{".True": textValue("True"), ".False": nullValue}}
		for _, name := range tmpl.variables() {
			switch {
			case name == ".True" || name == ".False":
			case lists[name]:
				o.vars[name], o.scope[name] = []string{"1", "2"}, listValue([]string{"1", "2"})
			case a.name == "distinct":
				o.vars[name], o.scope[name] = name, textValue(name)
			case a.null:
				o.vars[name], o.scope[name] = nil, nullValue
			default:
				o.vars[name], o.scope[name] = a.text, textValue(a.text)
			}
		}
		out = append(out, o)
	}
	return out
}

// listNames are the variables a range or join reads.
func listNames(nodes []tnode) map[string]bool {
	lists := map[string]bool{}
	for _, n := range nodes {
		switch n := n.(type) {
		case rangeNode:
			lists[n.variable] = true
			maps.Copy(lists, listNames(n.body))
		case joinNode:
			lists[n.variable] = true
		case ifNode:
			maps.Copy(lists, listNames(n.then))
			maps.Copy(lists, listNames(n.els))
		}
	}
	return lists
}
