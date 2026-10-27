package cardigann

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// variantLimit caps how many if-branch combinations of one template are
// compiled at load.
const variantLimit = 64

// placeholders stand in for substituted values when a selector template
// is compiled at load. A variant passes if either compiles, as a value may
// sit where a number or a word belongs.
var placeholders = []string{"1", "x"}

// checkCompiles compiles every selector in the definition, each branch of
// a templated one included, and checks that each template reads only the
// settings the definition declares. A templated selector fails only when no
// branch compiles. broken lists the branches that don't, which Prowlarr
// fails on only when a request takes them.
func checkCompiles(def *cardigannDefinition) (u *Unrunnable, broken []string) {
	c := checker{}
	c.configRefs(def)
	for _, kind := range responseKinds(def) {
		c.rows(def.Search.Rows, kind)
		for _, f := range def.Search.Fields {
			c.block("search.fields."+f.Key, &f.Value, kind)
		}
	}
	c.errorBlocks("search.error", def.Search.Error)
	if def.Login != nil {
		c.login(def.Login)
	}
	if def.Download != nil {
		c.download(def.Download)
	}
	return c.unrunnable, c.broken
}

type responseKind int

const (
	htmlResponse responseKind = iota
	jsonResponse
)

func responseKinds(def *cardigannDefinition) []responseKind {
	var kinds []responseKind
	for _, p := range def.Search.Paths {
		kind := htmlResponse
		if p.Response != nil && p.Response.Type == "json" {
			kind = jsonResponse
		}
		if !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

type checker struct {
	unrunnable *Unrunnable
	broken     []string
}

func (c *checker) fail(u *Unrunnable) {
	if c.unrunnable == nil {
		c.unrunnable = u
	}
}

func (c *checker) css(path, css string) {
	if css == "" {
		return
	}
	if _, err := compileSelector(css); err != nil {
		c.fail(unrunnable(err, "its selector %q at %s can't be read: %s", css, path, err))
	}
}

// templated compiles each variant of t with check. A variant compiles when
// it does with one of the placeholders, and t fails when none does.
func (c *checker) templated(path string, t *template, check func(string) error) {
	if t == nil {
		return
	}
	variants := make([][]string, len(placeholders))
	for i, p := range placeholders {
		variants[i] = t.variants(p, variantLimit)
	}
	var firstErr error
	compiled := false
	for i := range variants[0] {
		var err error
		for _, v := range variants {
			if err = check(v[i]); err == nil {
				break
			}
		}
		if err == nil {
			compiled = true
			continue
		}
		firstErr = cmp.Or(firstErr, err)
		c.broken = append(c.broken, fmt.Sprintf("%s %q: %s", path, variants[0][i], err))
	}
	if !compiled {
		c.fail(unrunnable(firstErr, "its selector %q at %s can't be read: %s", t.Source(), path, firstErr))
	}
}

func compileCSS(css string) error {
	_, err := compileSelector(css)
	return err
}

func (c *checker) block(path string, b *selectorBlock, kind responseKind) {
	if b == nil {
		return
	}
	if kind == jsonResponse {
		c.templated(path+".selector", b.Selector, func(s string) error {
			return checkJSONSelector(strings.TrimLeft(s, "."))
		})
		return
	}
	c.templated(path+".selector", b.Selector, compileCSS)
	c.css(path+".remove", b.Remove)
	for _, e := range b.Case {
		c.css(path+".case", e.Key)
	}
}

func (c *checker) rows(r rowsBlock, kind responseKind) {
	if kind == jsonResponse {
		c.templated("search.rows.selector", r.Selector, checkJSONSelector)
		if r.Attribute != "" {
			c.jsonPath("search.rows.attribute", r.Attribute)
		}
		c.block("search.rows.count", r.Count, kind)
		return
	}
	c.templated("search.rows.selector", r.Selector, compileCSS)
	c.block("search.rows.dateheaders", r.DateHeaders, kind)
}

func (c *checker) jsonPath(path, p string) {
	if _, err := compileJSONPath(p); err != nil {
		c.fail(unrunnable(err, "its JSON path %q at %s can't be read: %s", p, path, err))
	}
}

// checkJSONSelector compiles a JSON field or rows selector the way
// JsonParseFieldSelector reads it: the path before the first ':', then
// each :has, :not and :contains.
func checkJSONSelector(sel string) error {
	base, _, _ := strings.Cut(sel, ":")
	if _, err := compileJSONPath(base); err != nil {
		return err
	}
	m, err := jsonFilterRe.FindStringMatch(sel)
	for ; m != nil && err == nil; m, err = jsonFilterRe.FindNextMatch(m) {
		filter, key := group(m, "filter"), group(m, "key")
		if filter != "has" && filter != "not" {
			continue
		}
		if nested, _ := jsonFilterRe.MatchString(key); nested {
			err = checkJSONSelector(key)
		} else {
			_, err = compileJSONPath(key)
		}
		if err != nil {
			return err
		}
	}
	if err != nil {
		return fmt.Errorf("read the JSON selector %q: %w", sel, err)
	}
	return nil
}

func (c *checker) errorBlocks(path string, blocks []errorBlock) {
	for i, e := range blocks {
		p := fmt.Sprintf("%s[%d]", path, i)
		c.css(p+".selector", e.Selector)
		c.block(p+".message", e.Message, htmlResponse)
	}
}

func (c *checker) login(l *loginBlock) {
	form := l.Form
	if form == "" {
		form = "form"
	}
	c.css("login.form", form)
	c.errorBlocks("login.error", l.Error)
	if l.Test != nil {
		c.css("login.test.selector", l.Test.Selector)
	}
	if l.Captcha != nil {
		c.css("login.captcha.selector", l.Captcha.Selector)
		if l.Selectors {
			c.css("login.captcha.input", l.Captcha.Input)
		}
	}
	for _, e := range l.SelectorInputs {
		c.block("login.selectorinputs."+e.Key, &e.Value, htmlResponse)
	}
	for _, e := range l.GetSelectorInputs {
		c.block("login.getselectorinputs."+e.Key, &e.Value, htmlResponse)
	}
	if l.Selectors {
		for _, e := range l.Inputs {
			c.css("login.inputs", e.Key)
		}
	}
}

func (c *checker) download(d *downloadBlock) {
	field := func(path string, f *selectorField) {
		if f != nil {
			c.templated(path, f.Selector, compileCSS)
		}
	}
	for i := range d.Selectors {
		field(fmt.Sprintf("download.selectors[%d].selector", i), &d.Selectors[i])
	}
	if d.Infohash != nil {
		field("download.infohash.hash.selector", d.Infohash.Hash)
		field("download.infohash.title.selector", d.Infohash.Title)
	}
	if d.Before != nil {
		field("download.before.pathselector.selector", d.Before.PathSelector)
	}
}

// configVariables are the setting types that set a .Config variable.
var configVariables = []string{"text", "password", "checkbox", "select"}

// configRefs refuses a template that reads a .Config name no setting
// sets, which Prowlarr fails on at the first request.
func (c *checker) configRefs(def *cardigannDefinition) {
	declared := map[string]bool{"sitelink": true}
	for _, s := range def.Settings {
		if slices.Contains(configVariables, s.Type) {
			declared[s.Name] = true
		}
	}
	for _, t := range templatesIn(reflect.ValueOf(def).Elem()) {
		for _, name := range t.variables() {
			key, ok := strings.CutPrefix(name, ".Config.")
			if ok && !declared[key] {
				c.fail(unrunnable(fmt.Errorf("undeclared setting %s", key),
					"a template reads the setting %q, which it doesn't declare", key))
				return
			}
		}
	}
}

var filterType = reflect.TypeFor[filterBlock]()

// templatesIn finds every template in v, those inside filter arguments
// included.
func templatesIn(v reflect.Value) []*template {
	switch {
	case v.Type() == templateType:
		if t, _ := v.Interface().(*template); t != nil {
			return []*template{t}
		}
		return nil
	case v.Type() == filterType:
		f, _ := v.Interface().(filterBlock)
		return filterTemplates(f)
	}
	var out []*template
	switch v.Kind() { //nolint:exhaustive // no other kind can hold a template
	case reflect.Pointer:
		if !v.IsNil() {
			out = templatesIn(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				out = append(out, templatesIn(v.Field(i))...)
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			out = append(out, templatesIn(v.Index(i))...)
		}
	}
	return out
}

// templatedArgs are the filter arguments Prowlarr renders as templates.
var templatedArgs = map[string]int{"re_replace": 1, "replace": 1, "prepend": 0, "append": 0}

func filterTemplates(f filterBlock) []*template {
	i, ok := templatedArgs[f.Name]
	if !ok || i >= len(f.Args.Items) {
		return nil
	}
	t, err := parseTemplate(f.Args.Items[i])
	if err != nil {
		return nil
	}
	return []*template{t}
}
