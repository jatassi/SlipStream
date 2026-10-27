package cardigann

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// Search is a search's Exchange: one request per search path whose
// categories match the query, each response parsed as it arrives. Prowlarr
// stops at the first short page. SlipStream requests every matching path,
// as the spec says.
type Search struct {
	flow
	d     *Definition
	query Query
	// env's scope is shared by every path and row of the search, so a
	// row's .Result values stay for the next row, as in Prowlarr.
	env env
	// base is the scope Prowlarr templates with when it passes no
	// variables: the JSON rows selector, date headers and search errors.
	base     env
	site     *url.URL
	mapped   []string
	gets     []string
	releases []*parsed
}

// Search starts a search for q.
func (d *Definition) Search(s Settings, q Query, c Clock) *Search {
	x := &Search{d: d, query: q}
	x.flow = startFlow(x.start(s, c))
	return x
}

// Recent starts a search for the newest items: a search with no text.
func (d *Definition) Recent(s Settings, c Clock) *Search {
	return d.Search(s, Query{Mode: "search"}, c)
}

func (x *Search) start(s Settings, c Clock) (*step, error) {
	d := x.d
	sc, err := d.searchScope(s, x.query, c)
	if err != nil {
		return nil, err
	}
	x.env = env{Scope: sc, Clock: c, charset: d.charset}
	x.base = env{Scope: d.baseScope(s, c), Clock: c, charset: d.charset}
	if x.site, err = url.Parse(d.siteLink(s)); err != nil {
		return nil, fmt.Errorf("read the site link: %w", err)
	}
	kw, err := d.def.Search.KeywordsFilters.Apply(sc[".Query.Keywords"], x.env)
	if err != nil {
		return nil, fmt.Errorf("apply keywordsfilters: %w", err)
	}
	sc[".Keywords"] = kw
	x.mapped = d.categories.mapping.TrackerIDs(x.query.Categories)
	if len(x.mapped) == 0 {
		x.mapped = d.categories.defaults
	}
	return x.stepFrom(0)
}

// stepFrom is the request for the first path from i on that the search
// takes, nil when none is left. Each is built after the previous
// response is parsed, as Prowlarr's lazy request list is.
func (x *Search) stepFrom(i int) (*step, error) {
	paths := x.d.def.Search.Paths
	for ; i < len(paths); i++ {
		p := &paths[i]
		req, ok, err := x.request(p)
		if err != nil {
			return nil, fmt.Errorf("build the request for search path %d: %w", i+1, err)
		}
		if !ok {
			continue
		}
		next := i + 1
		return &step{req: req, then: func(r Response) (*step, error) {
			if err := x.parse(p, req.URL, r); err != nil {
				return nil, err
			}
			return x.stepFrom(next)
		}}, nil
	}
	return nil, nil
}

// request is GetRequest's body for one path (CardigannRequestGenerator.cs:
// 1097-1230). It reports false for a path whose categories don't match
// and for a GET already requested.
func (x *Search) request(p *searchPathBlock) (Request, bool, error) {
	sc := x.env.Scope
	sc[".Categories"] = listValue(x.mapped)
	if len(p.Categories) > 0 && len(x.mapped) > 0 {
		both := intersect(x.mapped, p.Categories)
		if (len(both) > 0) == (p.Categories[0] == "!") {
			return Request{}, false, nil
		}
		sc[".Categories"] = listValue(both)
	}
	path, err := p.Path.Eval(sc, urlEncodeUTF8)
	if err != nil {
		return Request{}, false, err
	}
	u, err := resolveURL(x.site, strings.ReplaceAll(path, "+", "%20"))
	if err != nil {
		return Request{}, false, err
	}
	pairs, err := x.inputs(p)
	if err != nil {
		return Request{}, false, err
	}
	req := Request{Method: http.MethodGet, Header: http.Header{}, FollowRedirects: p.FollowRedirect, Purpose: ForSearch}
	if strings.EqualFold(p.Method, methodPost) {
		req.Method = http.MethodPost
		req.Body = []byte(x.d.charset.QueryString(pairs, "&"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	target := u.String()
	if req.Method == http.MethodGet && len(pairs) > 0 {
		target += "?" + x.d.charset.QueryString(pairs, "&")
	}
	if req.Method == http.MethodGet && slices.Contains(x.gets, target) {
		return Request{}, false, nil
	}
	x.gets = append(x.gets, target)
	if req.URL, err = url.Parse(target); err != nil {
		return Request{}, false, fmt.Errorf("read the address %q: %w", target, err)
	}
	for _, h := range x.d.def.Search.Headers {
		if len(h.Value) == 0 {
			continue
		}
		v, err := h.Value[0].Eval(sc, nil)
		if err != nil {
			return Request{}, false, fmt.Errorf("header %s: %w", h.Key, err)
		}
		req.Header.Set(h.Key, v)
	}
	return req, true, nil
}

// inputs are the search's inputs, when the path inherits them, then the
// path's. A $raw input is templated URL-encoded and split into pairs that
// QueryString encodes again, as Prowlarr does. Any other input is dropped
// when blank unless allowEmptyInputs is set.
func (x *Search) inputs(p *searchPathBlock) ([]pair, error) {
	var lists []keyedList[*template]
	if p.InheritInputs {
		lists = append(lists, x.d.def.Search.Inputs)
	}
	lists = append(lists, p.Inputs)
	var pairs []pair
	for _, inputs := range lists {
		for _, in := range inputs {
			if in.Key == "$raw" {
				raw, err := in.Value.Eval(x.env.Scope, urlEncodeUTF8)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, rawPairs(raw)...)
				continue
			}
			v, err := in.Value.Eval(x.env.Scope, nil)
			if err != nil {
				return nil, fmt.Errorf("input %s: %w", in.Key, err)
			}
			if strings.TrimSpace(v) != "" || x.d.def.Search.AllowEmptyInputs {
				pairs = append(pairs, pair{Key: in.Key, Value: v})
			}
		}
	}
	return pairs, nil
}

func rawPairs(raw string) []pair {
	var out []pair
	for part := range strings.SplitSeq(raw, "&") {
		k, v, _ := strings.Cut(part, "=")
		if k != "" {
			out = append(out, pair{Key: k, Value: v})
		}
	}
	return out
}

// intersect is a's items that b holds, in a's order, without repeats.
func intersect(a, b []string) []string {
	var out []string
	for _, s := range a {
		if slices.Contains(b, s) && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// resolveURL is ResolvePath: ref read against base.
func resolveURL(base *url.URL, ref string) (*url.URL, error) {
	r, err := url.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("read the address %q: %w", ref, err)
	}
	return base.ResolveReference(r), nil
}
