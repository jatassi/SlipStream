package cardigann

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
)

// Got is what a download link leads to: a Torrent or a Magnet.
type Got interface{ got() }

// Torrent is a .torrent file's bytes as the site sent them. They decode as
// a v1 or hybrid torrent.
type Torrent struct{ Raw []byte }

// Magnet is a magnet link: one the page links to or the download redirects
// to, or a trackerless one built from the page's info hash.
type Magnet struct{ Link magnet.Link }

func (Torrent) got() {}
func (Magnet) got()  {}

// Download is a download's Exchange: Prowlarr's DownloadRequest
// (CardigannRequestGenerator.cs:827-990), then HttpIndexerBase.Download's
// request for the file. Every request asks the caller to follow redirects
// except a before request's.
type Download struct {
	flow
	d      *Definition
	env    env
	site   *url.URL
	link   *url.URL
	header http.Header
	method string
	// last is the response DownloadRequest read last, which a usebeforeresponse
	// block reads in place of fetching the link again. A test link's response
	// counts too, as it does in Prowlarr.
	last *Response
	got  Got
}

// Download starts the download of link, a Release's download link.
func (d *Definition) Download(s Settings, link *url.URL, c Clock) *Download {
	x := &Download{d: d, link: link, method: http.MethodGet}
	x.flow = startFlow(x.start(s, c))
	return x
}

// Result is the downloaded .torrent file or magnet link.
func (x *Download) Result() (Got, error) {
	if err := x.ended(); err != nil {
		return nil, err
	}
	return x.got, nil
}

func (x *Download) start(s Settings, c Clock) (*step, error) {
	d := x.d
	sc := d.baseScope(s, c)
	addURIVariables(sc, ".DownloadUri", x.link)
	x.env = env{Scope: sc, Clock: c, charset: d.charset}
	var err error
	if x.site, err = url.Parse(d.siteLink(s)); err != nil {
		return nil, fmt.Errorf("read the site link: %w", err)
	}
	dl := d.def.Download
	headers := d.def.Search.Headers
	if dl != nil && len(dl.Headers) > 0 {
		headers = dl.Headers
	}
	if x.header, err = templateHeaders(headers, sc); err != nil {
		return nil, err
	}
	if dl == nil {
		return x.fetch(x.link), nil
	}
	if dl.Method == methodPost {
		x.method = http.MethodPost
	}
	switch b := dl.Before; {
	case b == nil:
		return x.pick()
	case b.PathSelector != nil:
		return x.send(x.linkRequest(), func(r Response) (*step, error) {
			path, ok, err := x.match(r, b.PathSelector)
			if err != nil {
				return nil, fmt.Errorf("download before pathselector: %w", err)
			}
			if !ok {
				return nil, errors.New("the download before pathselector matched nothing on the download page")
			}
			t, err := parseTemplate(path)
			if err != nil {
				return nil, fmt.Errorf("download before path %q: %w", path, err)
			}
			return x.before(t)
		}), nil
	default:
		return x.before(b.Path)
	}
}

// before is HandleRequest (CardigannRequestGenerator.cs:754-825) for the
// before block, its path resolved against the site link.
func (x *Download) before(path *template) (*step, error) {
	b := x.d.def.Download.Before
	sc := x.env.Scope
	p, err := path.Eval(sc, nil)
	if err != nil {
		return nil, fmt.Errorf("download before path: %w", err)
	}
	u, err := resolveURL(x.site, p)
	if err != nil {
		return nil, err
	}
	var pairs []pair
	for _, in := range b.Inputs {
		v, err := in.Value.Eval(sc, nil)
		if err != nil {
			return nil, fmt.Errorf("download before input %s: %w", in.Key, err)
		}
		pairs = append(pairs, pair{Key: in.Key, Value: v})
	}
	req := Request{Method: http.MethodGet, Header: x.header.Clone(), Purpose: ForDownload}
	req.Header.Set("Referer", x.link.String())
	target := u.String()
	switch {
	case strings.EqualFold(b.Method, methodPost):
		req.Method = http.MethodPost
		req.Body = []byte(x.d.charset.QueryString(pairs, "&"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case len(pairs) > 0:
		if !strings.Contains(target, "?") {
			target += "?"
		}
		target += x.d.charset.QueryString(pairs, b.QuerySeparator)
	}
	if req.URL, err = url.Parse(target); err != nil {
		return nil, fmt.Errorf("read the address %q: %w", target, err)
	}
	return x.send(req, func(r Response) (*step, error) {
		if r.Status >= http.StatusBadRequest {
			return nil, fmt.Errorf("the download before request responded with HTTP %d", r.Status)
		}
		return x.pick()
	}), nil
}

// pick reads the link through the infohash block, else the selectors,
// else downloads it as it is.
func (x *Download) pick() (*step, error) {
	dl := x.d.def.Download
	switch {
	case dl.Infohash != nil:
		return x.infohash()
	case len(dl.Selectors) > 0:
		return x.selector(0)
	}
	return x.fetch(x.link), nil
}

// infohash builds a trackerless magnet link from the page's hash and
// title. A page that doesn't give both is downloaded as it is, as Prowlarr
// falls through on any exception.
func (x *Download) infohash() (*step, error) {
	ih := x.d.def.Download.Infohash
	return x.read(ih.UseBeforeResponse, func(r Response) (*step, error) {
		hash, ok := x.found(r, ih.Hash)
		title, titled := x.found(r, ih.Title)
		if !ok || !titled {
			return x.fetch(x.link), nil
		}
		l, err := magnet.Parse("magnet:?xt=urn:btih:" + url.QueryEscape(hash))
		if err != nil {
			return nil, fmt.Errorf("read the download page's info hash: %w", err)
		}
		l.Name = title
		x.got = Magnet{Link: l}
		return nil, nil
	})
}

// selector tries the download selectors from i on. A selector that fails,
// or whose link fails the testlinktorrent check, passes to the next.
func (x *Download) selector(i int) (*step, error) {
	sels := x.d.def.Download.Selectors
	if i == len(sels) {
		return nil, errors.New("no download selector matched the download page")
	}
	sel := &sels[i]
	next := func() (*step, error) { return x.selector(i + 1) }
	return x.read(sel.UseBeforeResponse, func(r Response) (*step, error) {
		href, ok := x.found(r, sel)
		if !ok {
			return next()
		}
		if magnet.Is(href) {
			l, err := magnet.Parse(href)
			if err != nil {
				return nil, fmt.Errorf("read the download page's magnet link: %w", err)
			}
			x.got = Magnet{Link: l}
			return nil, nil
		}
		to, err := resolveURL(x.link, href)
		if err != nil {
			return next()
		}
		if !x.d.def.TestLinkTorrent {
			return x.fetch(to), nil
		}
		test := Request{Method: http.MethodGet, URL: to, Header: x.header.Clone(), FollowRedirects: true, Purpose: ForDownload}
		return x.send(test, func(r Response) (*step, error) {
			got, err := downloaded(r)
			switch {
			case err != nil:
				return next()
			case x.method == http.MethodGet:
				x.got = got
				return nil, nil
			}
			return x.fetch(to), nil
		}), nil
	})
}

// read hands then the last response when the block uses the before
// response and there is one, else fetches the link first.
func (x *Download) read(useBefore bool, then func(Response) (*step, error)) (*step, error) {
	if useBefore && x.d.def.Download.Before != nil && x.last != nil {
		return then(*x.last)
	}
	return x.send(x.linkRequest(), then), nil
}

func (x *Download) send(req Request, then func(Response) (*step, error)) *step {
	return &step{req: req, then: func(r Response) (*step, error) {
		x.last = &r
		return then(r)
	}}
}

func (x *Download) linkRequest() Request {
	return Request{Method: http.MethodGet, URL: x.link, Header: x.header.Clone(), FollowRedirects: true, Purpose: ForDownload}
}

// fetch is the last request: the file at to, with the download method.
func (x *Download) fetch(to *url.URL) *step {
	req := Request{Method: x.method, URL: to, Header: x.header.Clone(), FollowRedirects: true, Purpose: ForDownload}
	return &step{req: req, then: func(r Response) (*step, error) {
		got, err := downloaded(r)
		x.got = got
		return nil, err
	}}
}

// match is MatchSelector over a download page's response. An HTTP error
// fails as Prowlarr's client throws on one.
func (x *Download) match(r Response, f *selectorField) (text string, ok bool, err error) {
	if r.Status >= http.StatusBadRequest {
		return "", false, fmt.Errorf("the download page responded with HTTP %d", r.Status)
	}
	doc, err := parseHTML(x.d.charset.Decode(r.Body))
	if err != nil {
		return "", false, err
	}
	v, ok, err := f.Match(doc, x.env)
	return v.String(), ok, err
}

// found is match for a block that passes over a failure, as Prowlarr's
// try blocks do.
func (x *Download) found(r Response, f *selectorField) (string, bool) {
	text, ok, err := x.match(r, f)
	return text, ok && err == nil
}

// downloaded reads the response to a download: a redirect to a magnet
// link, or a .torrent file. The caller follows every other redirect.
func downloaded(r Response) (Got, error) {
	if to := r.Header.Get("Location"); r.Status/100 == 3 && magnet.Is(to) {
		l, err := magnet.Parse(to)
		if err != nil {
			return nil, fmt.Errorf("read the magnet link the download redirected to: %w", err)
		}
		return Magnet{Link: l}, nil
	}
	if r.Status < 200 || r.Status >= 300 {
		return nil, fmt.Errorf("the download link responded with HTTP %d", r.Status)
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "text/html") {
		return nil, errors.New("the download link responded with a web page")
	}
	if _, err := bencode.DecodeMetainfo(r.Body); err != nil {
		return nil, fmt.Errorf("read the .torrent file: %w", err)
	}
	return Torrent{Raw: r.Body}, nil
}

// addURIVariables is AddTemplateVariablesFromUri (CardigannBase.cs:797-815).
// A query key given more than once takes its first value.
func addURIVariables(sc scope, prefix string, u *url.URL) {
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := ""
	if u.RawQuery != "" {
		query = "?" + u.RawQuery
	}
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[u.Scheme]
	}
	sc[prefix+".AbsoluteUri"] = textValue(u.String())
	sc[prefix+".AbsolutePath"] = textValue(path)
	sc[prefix+".Scheme"] = textValue(u.Scheme)
	sc[prefix+".Host"] = textValue(u.Hostname())
	sc[prefix+".Port"] = textValue(port)
	sc[prefix+".PathAndQuery"] = textValue(path + query)
	sc[prefix+".Query"] = textValue(query)
	// ParseQuery keeps every pair it reads past a malformed one, as
	// QueryHelpers.ParseQuery reads leniently.
	values, _ := url.ParseQuery(u.RawQuery)
	for k, vs := range values {
		sc[prefix+".Query."+k] = textValue(vs[0])
	}
}

// templateHeaders is ParseCustomHeaders: each header's first value,
// templated.
func templateHeaders(hs keyedList[[]*template], sc scope) (http.Header, error) {
	out := http.Header{}
	for _, h := range hs {
		if len(h.Value) == 0 {
			continue
		}
		v, err := h.Value[0].Eval(sc, nil)
		if err != nil {
			return nil, fmt.Errorf("header %s: %w", h.Key, err)
		}
		out.Set(h.Key, v)
	}
	return out, nil
}
