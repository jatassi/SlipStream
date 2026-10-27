package cardigann

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Rejected is a login.error or search.error match, with the site's own
// words.
type Rejected struct {
	Message string
	Search  bool
}

func (r *Rejected) Error() string {
	if r.Search {
		return fmt.Sprintf("the search page shows the error %q", r.Message)
	}
	return fmt.Sprintf("the sign-in page shows the error %q", r.Message)
}

// optionalFields never fail a row (CardigannBase.cs:33). They are matched
// by the whole field key, so description|append isn't one.
var optionalFields = []string{"imdb", "imdbid", "tmdbid", "rageid", "tvdbid", "tvmazeid", "traktid", "doubanid", "poster", "banner", "description", "genre"}

var redirectStatuses = []int{
	http.StatusMultipleChoices, http.StatusMovedPermanently, http.StatusFound,
	http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect,
}

var refreshURL = regexp.MustCompile(`^(.*?url)=(.*?)(?:;|$)`)

// parse is CardigannParser.ParseResponse for one path's response. A
// relative link resolves against the request's address, as Prowlarr's
// does.
func (x *Search) parse(p *searchPathBlock, reqURL *url.URL, r Response) error {
	if to := redirectTarget(reqURL, r); to != "" {
		if strings.Contains(strings.ToLower(to), "/login.php") {
			return errors.New("the site redirected the search to its login page")
		}
		return fmt.Errorf("the site redirected the search to %s", to)
	}
	if r.Status != http.StatusOK {
		return fmt.Errorf("the search page responded with HTTP %d", r.Status)
	}
	body := x.d.charset.Decode(r.Body)
	if p.Response != nil && p.Response.Type == "json" {
		return x.parseJSONResponse(p.Response, body, reqURL)
	}
	return x.parseDocument(p.Response != nil && p.Response.Type == "xml", body, reqURL)
}

// redirectTarget is RedirectUrl when HasHttpRedirect holds: the Location,
// or the url in a Refresh header, read against the request's address.
func redirectTarget(reqURL *url.URL, r Response) string {
	refresh := r.Header.Get("Refresh")
	if !slices.Contains(redirectStatuses, r.Status) && refresh == "" {
		return ""
	}
	to := r.Header.Get("Location")
	if to == "" {
		m := refreshURL.FindStringSubmatch(refresh)
		if m == nil {
			return ""
		}
		to = m[2]
	}
	u, err := resolveURL(reqURL, to)
	if err != nil {
		return to
	}
	return u.String()
}

// parseDocument parses an HTML or XML response's rows. A row that fails
// is skipped. A required field that fails leaves its release field unset
// and the row kept.
func (x *Search) parseDocument(isXML bool, body string, reqURL *url.URL) error {
	search := &x.d.def.Search
	parseDoc := parseHTML
	if isXML {
		parseDoc = parseXML
	}
	if len(search.PreprocessingFilters) > 0 {
		v, err := search.PreprocessingFilters.Apply(textValue(body), x.env)
		if err != nil {
			return fmt.Errorf("apply preprocessingfilters: %w", err)
		}
		body = v.text
	}
	doc, err := parseDoc(body)
	if err != nil {
		return err
	}
	if err := x.searchError(doc); err != nil {
		return err
	}
	if search.Rows.Selector == nil {
		return errors.New("the definition has no rows selector")
	}
	css, err := search.Rows.Selector.Eval(x.env.Scope, nil)
	if err != nil {
		return fmt.Errorf("rows selector: %w", err)
	}
	sel, err := compileSelector(css)
	if err != nil {
		return err
	}
	rows, err := mergeAfter(sel.All(doc), search.Rows.After)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if r, err := x.parseHTMLRow(row, reqURL); err == nil {
			x.releases = append(x.releases, r)
		}
	}
	return nil
}

// mergeAfter moves the child nodes of the after rows that follow each row
// into it. A row short of after followers fails the response, as
// Prowlarr's index past the end does.
func mergeAfter(rows []*html.Node, after int) ([]*html.Node, error) {
	if after <= 0 {
		return rows, nil
	}
	var out []*html.Node
	for i := 0; i < len(rows); i += after + 1 {
		if i+after >= len(rows) {
			return nil, fmt.Errorf("rows after %d runs past the last row", after)
		}
		row := rows[i]
		for _, m := range rows[i+1 : i+1+after] {
			for c := m.FirstChild; c != nil; c = m.FirstChild {
				m.RemoveChild(c)
				row.AppendChild(c)
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// searchError is the first search.error selector that matches, with real
// CSS :root, as a *Rejected carrying the element's text or the message
// selector's value.
func (x *Search) searchError(doc *html.Node) error {
	for _, e := range x.d.def.Search.Error {
		sel, err := compileSelector(e.Selector)
		if err != nil {
			return err
		}
		el := sel.First(doc)
		if el == nil {
			continue
		}
		msg := textContent(el)
		if e.Message != nil {
			v, err := e.Message.Select(firstElement(doc), x.base, true)
			if err != nil {
				return fmt.Errorf("search error message: %w", err)
			}
			msg = v.text
		}
		return &Rejected{Message: strings.TrimSpace(msg), Search: true}
	}
	return nil
}

func (x *Search) parseHTMLRow(row *html.Node, reqURL *url.URL) (*parsed, error) {
	r := &parsed{}
	for _, f := range x.d.def.Search.Fields {
		_ = x.field(r, f, reqURL, func(b *selectorBlock, required bool) (value, error) {
			return b.Select(row, x.env, required)
		})
	}
	if !r.Published.IsZero() || x.d.def.Search.Rows.DateHeaders == nil {
		return r, nil
	}
	return r, x.dateHeader(r, row)
}

// dateHeader dates r from the nearest earlier row that the dateheaders
// selector matches, looking at the row's earlier siblings, then at its
// parent's.
func (x *Search) dateHeader(r *parsed, row *html.Node) error {
	dh := x.d.def.Search.Rows.DateHeaders
	prev := earlierRow(row)
	for prev != nil {
		if v, err := dh.Select(prev, x.base, true); err == nil {
			if v.IsNull() {
				break
			}
			t, err := parseDate(v.text, x.env.Clock)
			if err != nil {
				return err
			}
			r.Published = t
			return nil
		}
		prev = earlierRow(prev)
	}
	if !dh.Optional {
		return errors.New("no date header row precedes the row")
	}
	return nil
}

// earlierRow is n's previous element sibling, or else its parent's.
func earlierRow(n *html.Node) *html.Node {
	if p := previousElement(n); p != nil {
		return p
	}
	if n.Parent != nil && n.Parent.Type == html.ElementNode {
		return previousElement(n.Parent)
	}
	return nil
}

func previousElement(n *html.Node) *html.Node {
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

func firstElement(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return c
		}
	}
	return n
}

// parseJSONResponse parses a JSON response's rows. Unlike HTML, a required
// field that fails fails the whole response.
func (x *Search) parseJSONResponse(resp *responseBlock, body string, reqURL *url.URL) error {
	if m := resp.NoResultsMessage; m != nil {
		if strings.TrimSpace(*m) != "" && strings.Contains(body, *m) || strings.TrimSpace(*m) == "" && strings.TrimSpace(body) == "" {
			return nil
		}
	}
	root, err := parseJSON(body)
	if err != nil {
		return err
	}
	rows := &x.d.def.Search.Rows
	if rows.Count != nil && x.countBelowOne(root) {
		return nil
	}
	objs, err := x.jsonRows(root)
	if err != nil {
		return err
	}
	for _, o := range objs {
		r := &parsed{}
		for _, f := range x.d.def.Search.Fields {
			obj := o.obj
			if strings.HasPrefix(f.Value.Selector.Source(), "..") {
				obj = o.row
			}
			err := x.field(r, f, reqURL, func(b *selectorBlock, required bool) (value, error) {
				return b.SelectJSON(obj, x.env, required)
			})
			if err != nil {
				return err
			}
		}
		x.releases = append(x.releases, r)
	}
	return nil
}

// countBelowOne reports whether rows.count reads as a whole number below
// 1. A count that fails to read is ignored.
func (x *Search) countBelowOne(root *jsonToken) bool {
	v, err := x.d.def.Search.Rows.Count.SelectJSON(root, x.env, true)
	if err != nil || v.IsNull() {
		return false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v.text), 10, 32)
	return err == nil && n < 1
}

// jsonRow is one release's object and the row of the rows array it came
// from, which a selector starting with ".." reads.
type jsonRow struct {
	row, obj *jsonToken
}

// jsonRows are the release objects: each row of the rows array, or the
// object its attribute names, or with multiple each object in that.
func (x *Search) jsonRows(root *jsonToken) ([]jsonRow, error) {
	rows := &x.d.def.Search.Rows
	var arr []*jsonToken
	css, err := rows.Selector.Eval(x.base.Scope, nil)
	ok := err == nil && rows.Selector != nil
	if ok {
		arr, ok = root.Rows(css)
	}
	if !ok {
		if rows.MissingAttributeEqualsNoResults {
			return nil, nil
		}
		return nil, fmt.Errorf("the rows selector %q finds no array of objects", rows.Selector.Source())
	}
	var out []jsonRow
	for _, row := range arr {
		sel := row
		if rows.Attribute != "" {
			t, err := row.SelectToken(rows.Attribute)
			if err != nil {
				return nil, err
			}
			if t == nil && rows.MissingAttributeEqualsNoResults {
				continue
			}
			sel = t
		}
		objs, err := rowObjects(sel, rows.Multiple)
		if err != nil {
			return nil, fmt.Errorf("rows attribute %q: %w", rows.Attribute, err)
		}
		for _, o := range objs {
			out = append(out, jsonRow{row: row, obj: o})
		}
	}
	return out, nil
}

// rowObjects is Value<JObject>() over t, or with multiple Values<JObject>():
// the items of an array or the values of an object, each an object.
func rowObjects(t *jsonToken, multiple bool) ([]*jsonToken, error) {
	if t == nil {
		return nil, errors.New("a row doesn't have it")
	}
	if !multiple {
		if t.kind != jsonObject {
			return nil, errors.New("a row's value isn't an object")
		}
		return []*jsonToken{t}, nil
	}
	if t.kind != jsonObject && t.kind != jsonArray {
		return nil, nil
	}
	for _, item := range t.items {
		if item.kind != jsonObject {
			return nil, errors.New("a row holds a value that isn't an object")
		}
	}
	return t.items, nil
}

// field is one field of a row, as the parser's try and catch blocks run
// it. An optional field that fails or reads blank takes its default, and
// its .Result value is null without one. A required field that fails
// keeps the earlier row's .Result value, as Prowlarr's does.
func (x *Search) field(r *parsed, f entry[selectorBlock], reqURL *url.URL, sel func(*selectorBlock, bool) (value, error)) error {
	name, mods, _ := strings.Cut(f.Key, "|")
	modifiers := strings.Split(mods, "|")
	key := ".Result." + name
	optional := slices.Contains(optionalFields, f.Key) || slices.Contains(modifiers, "optional") || f.Value.Optional
	err := func() error {
		v, err := sel(&f.Value, !optional)
		if err != nil {
			return err
		}
		if optional && v.blank() {
			d, err := f.Value.Default.Eval(x.env.Scope, nil)
			if err != nil {
				return err
			}
			if strings.TrimSpace(d) == "" {
				x.env.Scope[key] = nullValue
				return nil
			}
			v = textValue(d)
		}
		out, err := x.setField(r, name, modifiers, v, reqURL)
		if err != nil {
			return err
		}
		x.env.Scope[key] = out
		return nil
	}()
	if err == nil {
		return nil
	}
	if _, ok := x.env.Scope[key]; !ok || optional {
		x.env.Scope[key] = nullValue
	}
	if optional {
		return nil
	}
	return fmt.Errorf("field %s: %w", f.Key, err)
}
