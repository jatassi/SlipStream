package cardigann

import (
	"cmp"
	"encoding/json"
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

var (
	// ErrCaptchaNeeded ends an automatic sign-in whose sign-in page shows
	// a captcha, which only an Admin can answer.
	ErrCaptchaNeeded = errors.New("the sign-in page shows a captcha")
	// ErrSignInRefused ends a sign-in that the site responded to with HTTP
	// 401, or after which the login test page still reads as signed out.
	ErrSignInRefused = errors.New("the site didn't accept the sign-in")
)

const formContentType = "application/x-www-form-urlencoded"

// Login is a sign-in's Exchange. Its Result is nil once signed in, a
// *Rejected with the site's own words, ErrSignInRefused, or
// ErrCaptchaNeeded when a sign-in without an Admin's answer meets a
// captcha.
type Login struct {
	flow
}

// Login starts a sign-in. A form sign-in resumes from c's pre-login state
// with c's answer when c is set, and fetches the sign-in page itself
// otherwise. A cookie sign-in sends only the login test, since the
// caller's jar already holds the pasted cookie.
func (d *Definition) Login(s Settings, c *Captcha, clock Clock) *Login {
	si, err := d.newSignIn(s, clock, ForLogin)
	if err != nil {
		return &Login{startFlow(nil, err)}
	}
	return &Login{startFlow(si.start(c))}
}

// Result is how the sign-in ended.
func (x *Login) Result() error { return x.ended() }

// Captcha is a captcha image and the pre-login state a Login resumes from.
type Captcha struct {
	// Image is empty when the sign-in page shows no captcha this time.
	Image       []byte
	ContentType string
	// PreLogin is opaque: the sign-in page's cookies, its form's action
	// and its inputs.
	PreLogin []byte
	// Answer is the Admin's, set by the caller before Login.
	Answer string
}

// CaptchaFetch is the Exchange that fetches a form sign-in's page and its
// captcha image, Prowlarr's GetConfigurationForSetup.
type CaptchaFetch struct {
	flow
	out Captcha
}

// CaptchaFetch starts fetching the captcha for a form sign-in.
func (d *Definition) CaptchaFetch(s Settings, clock Clock) *CaptchaFetch {
	x := &CaptchaFetch{}
	x.flow = startFlow(x.start(d, s, clock))
	return x
}

// Result is the captcha, once fetched.
func (x *CaptchaFetch) Result() (Captcha, error) {
	if err := x.ended(); err != nil {
		return Captcha{}, err
	}
	return x.out, nil
}

func (x *CaptchaFetch) start(d *Definition, s Settings, clock Clock) (*step, error) {
	si, err := d.newSignIn(s, clock, ForCaptcha)
	if err != nil {
		return nil, err
	}
	if si.login == nil || si.login.Method != "form" {
		return nil, errors.New("the definition doesn't sign in with a form")
	}
	return si.landing(func(doc *html.Node) (*step, error) {
		pre, err := si.readForm(doc)
		if err != nil {
			return nil, err
		}
		img, err := si.captchaImage(doc)
		if err != nil {
			return nil, err
		}
		if img == nil {
			return nil, x.finish(si, pre)
		}
		src, _ := attr(img, "src")
		u, err := resolveURL(si.loginURL, src)
		if err != nil {
			return nil, err
		}
		return &step{req: si.request(http.MethodGet, u, si.loginURL.String()), then: func(r Response) (*step, error) {
			if r.Status >= http.StatusBadRequest {
				return nil, fmt.Errorf("the captcha image responded with HTTP %d", r.Status)
			}
			si.keep(r)
			x.out.Image, x.out.ContentType = r.Body, r.Header.Get("Content-Type")
			return nil, x.finish(si, pre)
		}}, nil
	}), nil
}

func (x *CaptchaFetch) finish(si *signIn, pre *preLogin) error {
	pre.Cookies = si.cookies
	b, err := json.Marshal(pre)
	if err != nil {
		return fmt.Errorf("write the pre-login state: %w", err)
	}
	x.out.PreLogin = b
	return nil
}

// preLogin is what a form sign-in reads from its sign-in page, kept so
// that a Login can resume after the Admin answers the captcha.
type preLogin struct {
	// Action is the submit address, getselectorinputs included.
	Action    string
	Multipart bool
	// Form are the form's own inputs.
	Form []pair
	// Names map each login.inputs selector to the name of the input it
	// finds, in selectors mode.
	Names         []pair
	Selected      []pair
	CaptchaInput  string
	SimpleCaptcha bool
	Cookies       []pair
}

// signIn is what a sign-in's requests share. cookies are the ones it
// sends, Prowlarr's Cookies, which the sign-in page's Set-Cookie updates.
type signIn struct {
	d        *Definition
	login    *loginBlock
	env      env
	site     *url.URL
	path     string
	loginURL *url.URL
	headers  http.Header
	cookies  []pair
	purpose  Purpose
}

func (d *Definition) newSignIn(s Settings, clock Clock, p Purpose) (*signIn, error) {
	si := &signIn{d: d, login: d.def.Login, env: env{Scope: d.baseScope(s, clock), Clock: clock, charset: d.charset}, purpose: p}
	var err error
	if si.site, err = url.Parse(d.siteLink(s)); err != nil {
		return nil, fmt.Errorf("read the site link: %w", err)
	}
	if si.login == nil {
		return si, nil
	}
	if si.path, err = si.login.Path.Eval(si.env.Scope, nil); err != nil {
		return nil, fmt.Errorf("login path: %w", err)
	}
	if si.loginURL, err = resolveURL(si.site, si.path); err != nil {
		return nil, err
	}
	headers := si.login.Headers
	if headers == nil {
		headers = d.def.Search.Headers
	}
	if si.headers, err = customHeaders(headers, si.env.Scope); err != nil {
		return nil, err
	}
	return si, nil
}

// start is DoLogin's first request, nil for a cookie sign-in without a
// login test.
func (si *signIn) start(c *Captcha) (*step, error) {
	l := si.login
	switch {
	case l == nil:
		return nil, nil
	case l.Method == "cookie":
		return si.test()
	case l.Method == methodPost:
		return si.post()
	case l.Method == methodGet:
		pairs, err := si.inputs()
		if err != nil {
			return nil, err
		}
		return si.getAt(si.path + "?" + charset{}.QueryString(pairs, "&"))
	case l.Method == "oneurl":
		one, ok := l.Inputs.Get("oneurl")
		if !ok {
			return nil, errors.New("the oneurl sign-in has no oneurl input")
		}
		v, err := one.Eval(si.env.Scope, nil)
		if err != nil {
			return nil, fmt.Errorf("login input oneurl: %w", err)
		}
		return si.getAt(si.path + v)
	case c != nil:
		var pre preLogin
		if err := json.Unmarshal(c.PreLogin, &pre); err != nil {
			return nil, fmt.Errorf("read the pre-login state: %w", err)
		}
		si.cookies = pre.Cookies
		return si.submit(&pre, c.Answer)
	}
	return si.landing(func(doc *html.Node) (*step, error) {
		img, err := si.captchaImage(doc)
		if err != nil {
			return nil, err
		}
		if img != nil {
			return nil, ErrCaptchaNeeded
		}
		pre, err := si.readForm(doc)
		if err != nil {
			return nil, err
		}
		return si.submit(pre, "")
	}), nil
}

// request is a sign-in request with the sign-in's headers, the Referer
// and the sign-in's own cookies, which replace the site's jar.
func (si *signIn) request(method string, u *url.URL, referer string) Request {
	h := si.headers.Clone()
	h.Set("Referer", referer)
	if len(si.cookies) > 0 {
		parts := make([]string, len(si.cookies))
		for i, c := range si.cookies {
			parts[i] = c.Key + "=" + c.Value
		}
		h.Set("Cookie", strings.Join(parts, "; "))
	}
	return Request{Method: method, URL: u, Header: h, ReplaceCookies: true, Purpose: si.purpose}
}

func (si *signIn) inputs() ([]pair, error) {
	pairs := make([]pair, 0, len(si.login.Inputs))
	for _, in := range si.login.Inputs {
		v, err := in.Value.Eval(si.env.Scope, nil)
		if err != nil {
			return nil, fmt.Errorf("login input %s: %w", in.Key, err)
		}
		pairs = append(pairs, pair{Key: in.Key, Value: v})
	}
	return pairs, nil
}

// post is the post sign-in: login.inputs as a form in the definition's
// encoding, sent with login.cookies.
func (si *signIn) post() (*step, error) {
	pairs, err := si.inputs()
	if err != nil {
		return nil, err
	}
	si.cookies = parseCookieHeader(strings.Join(si.login.Cookies, "; "))
	req := si.request(http.MethodPost, si.loginURL, si.site.String())
	req.FollowRedirects = true
	req.Header.Set("Content-Type", formContentType)
	req.Body = []byte(si.d.charset.QueryString(pairs, "&"))
	return si.signedIn(req), nil
}

// getAt is the get and oneurl sign-in: a GET of target, read against the
// site link, without cookies and without following redirects.
func (si *signIn) getAt(target string) (*step, error) {
	u, err := resolveURL(si.site, target)
	if err != nil {
		return nil, err
	}
	return si.signedIn(si.request(http.MethodGet, u, si.site.String())), nil
}

// landing fetches the sign-in page with login.cookies and hands then its
// document.
func (si *signIn) landing(then func(*html.Node) (*step, error)) *step {
	si.cookies = parseCookieHeader(strings.Join(si.login.Cookies, "; "))
	req := si.request(http.MethodGet, si.loginURL, si.site.String())
	req.FollowRedirects = si.d.def.FollowRedirect
	return &step{req: req, then: func(r Response) (*step, error) {
		if r.Status >= http.StatusBadRequest {
			return nil, fmt.Errorf("the sign-in page responded with HTTP %d", r.Status)
		}
		si.keep(r)
		doc, err := parseHTML(si.d.charset.Decode(r.Body))
		if err != nil {
			return nil, err
		}
		return then(doc)
	}}
}

// keep merges r's Set-Cookie into the sign-in's cookies, dropping any the
// site expired.
func (si *signIn) keep(r Response) {
	for _, c := range (&http.Response{Header: r.Header}).Cookies() {
		if c.MaxAge < 0 || !c.Expires.IsZero() && !c.Expires.After(si.env.Clock.Now) {
			si.cookies = slices.DeleteFunc(si.cookies, func(p pair) bool { return p.Key == c.Name })
			continue
		}
		si.cookies = setPair(si.cookies, c.Name, c.Value)
	}
}

func (si *signIn) captchaImage(doc *html.Node) (*html.Node, error) {
	if si.login.Captcha == nil {
		return nil, nil
	}
	return firstMatch(doc, si.login.Captcha.Selector)
}

// readForm reads the sign-in form as DoLogin does
// (CardigannRequestGenerator.cs:254-383).
func (si *signIn) readForm(doc *html.Node) (*preLogin, error) {
	l := si.login
	formCSS := cmp.Or(l.Form, "form")
	form, err := firstMatch(doc, formCSS)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, fmt.Errorf("the sign-in page has no form matching %q", formCSS)
	}
	enctype, _ := attr(form, "enctype")
	pre := &preLogin{Multipart: enctype == "multipart/form-data"}
	if pre.Form, err = formInputs(form); err != nil {
		return nil, err
	}
	if pre.Names, pre.CaptchaInput, err = si.inputNames(doc); err != nil {
		return nil, err
	}
	if pre.Selected, err = si.selectorInputs(doc, l.SelectorInputs); err != nil {
		return nil, err
	}
	query, err := si.selectorInputs(doc, l.GetSelectorInputs)
	if err != nil {
		return nil, err
	}
	action, _ := attr(form, "action")
	if l.SubmitPath != "" {
		action = l.SubmitPath
	}
	if len(query) > 0 {
		action += "?" + charset{}.QueryString(query, "&")
	}
	u, err := resolveURL(si.loginURL, action)
	if err != nil {
		return nil, err
	}
	pre.Action = u.String()
	simple, err := firstMatch(doc, `script[src*="simpleCaptcha"]`)
	if err != nil {
		return nil, err
	}
	pre.SimpleCaptcha = simple != nil
	return pre, nil
}

// formInputs are the form's named, enabled inputs, a checkbox or radio
// only when checked, as a browser submits them.
func formInputs(form *html.Node) ([]pair, error) {
	sel, err := compileSelector("input")
	if err != nil {
		return nil, err
	}
	var out []pair
	for _, in := range sel.All(form) {
		name, named := attr(in, "name")
		if _, disabled := attr(in, "disabled"); !named || disabled {
			continue
		}
		typ, _ := attr(in, "type")
		if _, checked := attr(in, "checked"); !checked && slices.Contains([]string{"checkbox", "radio"}, strings.ToLower(typ)) {
			continue
		}
		v, _ := attr(in, "value")
		out = setPair(out, name, v)
	}
	return out, nil
}

// inputNames map each login.inputs key to the name of the input it
// selects in selectors mode, and name the captcha input.
func (si *signIn) inputNames(doc *html.Node) (names []pair, captcha string, err error) {
	l := si.login
	if l.Captcha != nil {
		captcha = l.Captcha.Input
	}
	if !l.Selectors {
		return nil, captcha, nil
	}
	for _, in := range l.Inputs {
		name, err := inputName(doc, in.Key)
		if err != nil {
			return nil, "", err
		}
		names = append(names, pair{Key: in.Key, Value: name})
	}
	if l.Captcha != nil {
		if captcha, err = inputName(doc, l.Captcha.Input); err != nil {
			return nil, "", err
		}
	}
	return names, captcha, nil
}

// selectorInputs are selectorinputs or getselectorinputs read from the
// sign-in page's top element. An optional one that finds nothing is left
// out.
func (si *signIn) selectorInputs(doc *html.Node, list keyedList[selectorBlock]) ([]pair, error) {
	var out []pair
	for _, e := range list {
		v, err := e.Value.Select(firstElement(doc), si.env, !e.Value.Optional)
		if err != nil {
			return nil, fmt.Errorf("login selector input %s: %w", e.Key, err)
		}
		if !v.IsNull() {
			out = setPair(out, e.Key, v.text)
		}
	}
	return out, nil
}

func inputName(doc *html.Node, css string) (string, error) {
	el, err := firstMatch(doc, css)
	if err != nil {
		return "", err
	}
	if el == nil {
		return "", fmt.Errorf("the sign-in page has no input matching %q", css)
	}
	name, _ := attr(el, "name")
	return name, nil
}

// submit posts the form: its inputs, then login.inputs, the
// selectorinputs, simpleCaptcha's answer and the captcha answer over them.
func (si *signIn) submit(pre *preLogin, answer string) (*step, error) {
	l := si.login
	pairs := slices.Clone(pre.Form)
	for _, in := range l.Inputs {
		v, err := in.Value.Eval(si.env.Scope, nil)
		if err != nil {
			return nil, fmt.Errorf("login input %s: %w", in.Key, err)
		}
		key := in.Key
		if l.Selectors {
			i := slices.IndexFunc(pre.Names, func(p pair) bool { return p.Key == in.Key })
			if i < 0 {
				return nil, fmt.Errorf("the pre-login state has no input for %q", in.Key)
			}
			key = pre.Names[i].Value
		}
		pairs = setPair(pairs, key, v)
	}
	for _, p := range pre.Selected {
		pairs = setPair(pairs, p.Key, p.Value)
	}
	finish := func(pairs []pair) (*step, error) {
		if l.Captcha != nil && strings.TrimSpace(answer) != "" {
			pairs = setPair(pairs, pre.CaptchaInput, answer)
		}
		return si.postForm(pre, pairs)
	}
	if !pre.SimpleCaptcha {
		return finish(pairs)
	}
	u, err := resolveURL(si.site, "simpleCaptcha.php?numImages=1")
	if err != nil {
		return nil, err
	}
	return &step{req: si.request(http.MethodGet, u, si.loginURL.String()), then: func(r Response) (*step, error) {
		hash, err := simpleCaptchaHash(si.d.charset.Decode(r.Body))
		if err != nil {
			return nil, err
		}
		pairs = setPair(pairs, "captchaSelection", hash)
		pairs = setPair(pairs, "submitme", "X")
		return finish(pairs)
	}}, nil
}

// simpleCaptchaHash is the hash of the first image simpleCaptcha.php
// offers, which solves it.
func simpleCaptchaHash(body string) (string, error) {
	root, err := parseJSON(body)
	if err != nil {
		return "", fmt.Errorf("read the simpleCaptcha response: %w", err)
	}
	hash, err := root.SelectToken("images[0].hash")
	if err != nil {
		return "", err
	}
	if hash == nil {
		return "", errors.New("the simpleCaptcha response has no image")
	}
	return hash.String(), nil
}

func (si *signIn) postForm(pre *preLogin, pairs []pair) (*step, error) {
	u, err := url.Parse(pre.Action)
	if err != nil {
		return nil, fmt.Errorf("read the address %q: %w", pre.Action, err)
	}
	req := si.request(http.MethodPost, u, si.loginURL.String())
	req.FollowRedirects = true
	if !pre.Multipart {
		req.Header.Set("Content-Type", formContentType)
		req.Body = []byte(si.d.charset.QueryString(pairs, "&"))
		return si.signedIn(req), nil
	}
	boundary := "---------------------------" + strconv.FormatInt(si.env.Clock.Now.UnixMicro(), 10)
	parts := make([]string, 0, len(pairs)+1)
	for _, p := range pairs {
		parts = append(parts, "--"+boundary+"\r\nContent-Disposition: form-data; name=\""+p.Key+"\"\r\n\r\n"+p.Value)
	}
	parts = append(parts, "--"+boundary+"--")
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Referer", si.site.String())
	req.Body = si.d.charset.Encode(strings.Join(parts, "\r\n"))
	return si.signedIn(req), nil
}

// signedIn is the step of a request that signs in: its response is
// checked for errors, then the login test runs.
func (si *signIn) signedIn(req Request) *step {
	return &step{req: req, then: func(r Response) (*step, error) {
		if err := si.checkError(r); err != nil {
			return nil, err
		}
		return si.test()
	}}
}

// checkError is CheckForError (CardigannRequestGenerator.cs:598-628): HTTP
// 401 refuses the sign-in, and a login.error selector that matches
// rejects it with the site's words.
func (si *signIn) checkError(r Response) error {
	if r.Status == http.StatusUnauthorized {
		return ErrSignInRefused
	}
	if len(si.login.Error) == 0 {
		return nil
	}
	doc, err := parseHTML(si.d.charset.Decode(r.Body))
	if err != nil {
		return err
	}
	msg, found, err := matchError(si.login.Error, doc, si.env)
	if err != nil || !found {
		return err
	}
	return &Rejected{Message: msg}
}

// test is the login test page's step, nil without a login.test path. A
// test page that reads as signed out refuses the sign-in, as Jackett's
// TestLogin does. Prowlarr never requests it.
func (si *signIn) test() (*step, error) {
	t := si.login.Test
	if t == nil || t.Path == "" {
		return nil, nil
	}
	u, err := resolveURL(si.site, t.Path)
	if err != nil {
		return nil, err
	}
	req := Request{Method: http.MethodGet, URL: u, Header: si.headers.Clone(), Purpose: si.purpose}
	return &step{req: req, then: func(r Response) (*step, error) {
		if r.Status >= http.StatusBadRequest && r.Status != http.StatusUnauthorized && r.Status != http.StatusForbidden {
			return nil, fmt.Errorf("the login test page responded with HTTP %d", r.Status)
		}
		if si.d.LoggedOut(r) {
			return nil, ErrSignInRefused
		}
		return nil, nil
	}}, nil
}

// LoggedOut is CheckIfLoginNeeded (CardigannRequestGenerator.cs:1029-1068)
// over a response, read before Feed: for a definition that signs in, a
// redirect, HTTP 401 or 403, or an HTML page that the login.test selector
// finds nothing on. Prowlarr signs in again on any status from 400. The
// spec reads a 429 and a 5xx first and counts only 401 and 403 here.
func (d *Definition) LoggedOut(r Response) bool {
	l := d.def.Login
	switch {
	case l == nil:
		return false
	case isRedirect(r) || r.Status == http.StatusUnauthorized || r.Status == http.StatusForbidden:
		return true
	case l.Test == nil || l.Test.Selector == "":
		return false
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "text/html") {
		return false
	}
	doc, err := parseHTML(d.charset.Decode(r.Body))
	if err != nil {
		return true
	}
	el, err := firstMatch(doc, l.Test.Selector)
	return err != nil || el == nil
}

func firstMatch(doc *html.Node, css string) (*html.Node, error) {
	sel, err := compileSelector(css)
	if err != nil {
		return nil, err
	}
	return sel.First(doc), nil
}

// customHeaders is ParseCustomHeaders: each header's first value,
// templated. A header without values is left out.
func customHeaders(list keyedList[[]*template], sc scope) (http.Header, error) {
	h := http.Header{}
	for _, e := range list {
		if len(e.Value) == 0 {
			continue
		}
		v, err := e.Value[0].Eval(sc, nil)
		if err != nil {
			return nil, fmt.Errorf("header %s: %w", e.Key, err)
		}
		h.Set(e.Key, v)
	}
	return h, nil
}

// setPair sets key's value in place, or appends it, as a .NET Dictionary's
// indexer does.
func setPair(pairs []pair, key, value string) []pair {
	for i := range pairs {
		if pairs[i].Key == key {
			pairs[i].Value = value
			return pairs
		}
	}
	return append(pairs, pair{Key: key, Value: value})
}

var (
	cookiePair = regexp.MustCompile(`([^()<>@,;:\\"/\[\]?={}\s]+)=([^,;\\"\s]+)`)
	// cookieAttributes are the Set-Cookie attribute names that a pasted
	// header may carry and that aren't cookies.
	cookieAttributes = []string{"COMMENT", "COMMENTURL", "DISCORD", "DOMAIN", "EXPIRES", "MAX-AGE", "PATH", "PORT", "SECURE", "VERSION", "HTTPONLY", "SAMESITE"}
)

// parseCookieHeader is CookieUtil.CookieHeaderToDictionary: each
// name=value pair in s, attribute names dropped, a repeated name taking
// the later value in the first one's place.
func parseCookieHeader(s string) []pair {
	var out []pair
	for _, m := range cookiePair.FindAllStringSubmatch(s, -1) {
		if !slices.Contains(cookieAttributes, strings.ToUpper(m[1])) {
			out = setPair(out, m[1], m[2])
		}
	}
	return out
}
