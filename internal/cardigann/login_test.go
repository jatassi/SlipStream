package cardigann

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func with(r Response, headers ...string) Response {
	r.Header = http.Header{}
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Add(headers[i], headers[i+1])
	}
	return r
}

func status(code int, headers ...string) Response {
	return with(Response{Status: code}, headers...)
}

var signInSettings = Settings{Values: map[string]string{"username": "jo", "password": "p@ss word!", "apikey": "K&Y", "passkey": "PK"}}

func login(t *testing.T, d *Definition, c *Captcha, s site) ([]Request, error) {
	t.Helper()
	x := d.Login(signInSettings, c, testClock)
	sent := s.drive(t, x)
	return sent, x.Result()
}

func wantHeaders(t *testing.T, r Request, want map[string]string) {
	t.Helper()
	for k, v := range want {
		if got := r.Header.Get(k); got != v {
			t.Errorf("%s %s: %s = %q, want %q", r.Method, r.URL, k, got, v)
		}
	}
}

const signInBase = `links: [https://site.example/]
settings:
  - {name: username, type: text}
  - {name: password, type: password}
  - {name: apikey, type: text}
  - {name: passkey, type: text}
caps:
  modes:
    search: [q]
search:
  path: browse.php
  headers:
    X-Search: ["s"]
  rows:
    selector: tr
  fields:
    title:
      selector: a
`

const postDefinition = "id: poster\n" + signInBase + `login:
  method: post
  path: takelogin.php
  cookies: ["lang=en; path=/", "theme=dark"]
  headers:
    X-Login: ["{{ .Config.username }}"]
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
    keeplogged: "1"
  error:
    - selector: div.error
  test:
    path: my.php
    selector: a[href*="logout"]
`

func TestLoginPost(t *testing.T) {
	d := mustLoad(t, postDefinition)
	const signIn, test = "POST https://site.example/takelogin.php", "GET https://site.example/my.php"
	sent, err := login(t, d, nil, site{signIn: page("<p>Welcome</p>"), test: page(`<a href="logout.php">Log out</a>`)})
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, signIn, test)
	post := sent[0]
	if got := string(post.Body); got != "username=jo&password=p%40ss+word!&keeplogged=1" {
		t.Errorf("body %q", got)
	}
	wantHeaders(t, post, map[string]string{
		"Content-Type": formContentType, "Referer": "https://site.example/", "Cookie": "lang=en; theme=dark", "X-Login": "jo", "X-Search": "",
	})
	if !post.ReplaceCookies || !post.FollowRedirects || post.Purpose != ForLogin {
		t.Errorf("sign-in request: replace cookies %v, follow redirects %v, purpose %d", post.ReplaceCookies, post.FollowRedirects, post.Purpose)
	}
	if check := sent[1]; check.ReplaceCookies || check.Header.Get("Cookie") != "" || check.Header.Get("X-Login") != "jo" {
		t.Errorf("test request: replace cookies %v, headers %v", check.ReplaceCookies, check.Header)
	}
}

func TestLoginRefused(t *testing.T) {
	d := mustLoad(t, postDefinition)
	const signIn, test = "POST https://site.example/takelogin.php", "GET https://site.example/my.php"
	tests := []struct {
		name   string
		s      site
		reject string
		err    error
	}{
		{"wrong password", site{signIn: page(`<div class="error"> Username or password incorrect </div>`)}, "Username or password incorrect", nil},
		{"HTTP 401", site{signIn: status(http.StatusUnauthorized)}, "", ErrSignInRefused},
		{"test page signed out", site{signIn: page("<p>Login</p>"), test: page(`<a href="login.php">Log in</a>`)}, "", ErrSignInRefused},
		{"test page redirects", site{signIn: page(""), test: status(http.StatusFound, "Location", "/login.php")}, "", ErrSignInRefused},
		{"test page HTTP 403", site{signIn: page(""), test: status(http.StatusForbidden)}, "", ErrSignInRefused},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := login(t, d, nil, tt.s)
			var rej *Rejected
			switch {
			case tt.err != nil && !errors.Is(err, tt.err):
				t.Errorf("Result: %v, want %v", err, tt.err)
			case tt.err == nil && (!errors.As(err, &rej) || rej.Message != tt.reject || rej.Search):
				t.Errorf("Result: %v, want a sign-in Rejected with %q", err, tt.reject)
			}
		})
	}
	t.Run("test page HTTP 500", func(t *testing.T) {
		_, err := login(t, d, nil, site{signIn: page(""), test: status(http.StatusInternalServerError)})
		if err == nil || errors.Is(err, ErrSignInRefused) {
			t.Errorf("Result: %v, want an error that doesn't refuse the sign-in", err)
		}
	})
}

func TestLoginGetIsUTF8(t *testing.T) {
	d := mustLoad(t, "id: getter\nencoding: windows-1251\n"+signInBase+`login:
  method: get
  path: api/check
  inputs:
    apikey: "{{ .Config.apikey }}"
    q: "привет"
  error:
    - selector: ':root:contains("Invalid API key")'
      message:
        text: That API key is wrong
`)
	const check = "GET https://site.example/api/check?apikey=K%26Y&q=%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82"
	sent, err := login(t, d, nil, site{check: page(`{"ok": true}`)})
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, check)
	wantHeaders(t, sent[0], map[string]string{"Cookie": "", "Referer": "https://site.example/", "X-Search": "s"})
	if sent[0].FollowRedirects || !sent[0].ReplaceCookies {
		t.Errorf("follow redirects %v, replace cookies %v", sent[0].FollowRedirects, sent[0].ReplaceCookies)
	}
	_, err = login(t, d, nil, site{check: page(`{"error": "Invalid API key"}`)})
	var rej *Rejected
	if !errors.As(err, &rej) || rej.Message != "That API key is wrong" {
		t.Errorf("Result: %v, want the error selector's message", err)
	}
}

func TestLoginOneURL(t *testing.T) {
	d := mustLoad(t, "id: one\n"+signInBase+`login:
  method: oneurl
  path: login/
  inputs:
    oneurl: "{{ .Config.passkey }}"
`)
	const one = "GET https://site.example/login/PK"
	sent, err := login(t, d, nil, site{one: page("")})
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, one)
}

func TestLoginCookie(t *testing.T) {
	d := mustLoad(t, "id: cookie\n"+signInBase+`login:
  method: cookie
  test:
    path: index.php
    selector: a.logout
`)
	const test = "GET https://site.example/index.php"
	sent, err := login(t, d, nil, site{test: page(`<a class="logout">Log out</a>`)})
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, test)
	if sent[0].ReplaceCookies {
		t.Error("the test request replaces the pasted cookie")
	}
	if _, err := login(t, d, nil, site{test: page("<form>Log in</form>")}); !errors.Is(err, ErrSignInRefused) {
		t.Errorf("Result with an expired cookie: %v, want ErrSignInRefused", err)
	}
	for _, src := range []string{"id: untested\n" + signInBase + "login:\n  method: cookie\n", "id: none\n" + signInBase} {
		if sent, err := login(t, mustLoad(t, src), nil, site{}); err != nil || len(sent) != 0 {
			t.Errorf("Login sent %v and ended with %v, want nothing", requestLines(sent), err)
		}
	}
}

const formDefinition = "id: former\n" + signInBase + `login:
  path: login.php
  form: form#login
  submitpath: takelogin.php
  cookies: ["seen=1"]
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  selectorinputs:
    token:
      selector: input[name="csrf"]
      attribute: value
    absent:
      selector: span.none
      optional: true
  getselectorinputs:
    sid:
      selector: form#login
      attribute: data-sid
  error:
    - selector: td.embedded:contains("failed")
      message:
        selector: td.embedded
        remove: b
  test:
    path: index.php
    selector: a[href="logout.php"]
`

const formLanding = `<html><body>
<form id="search" action="search.php"><input name="q" value="x"></form>
<form id="login" action="ignored.php" data-sid="S 1">
<input name="csrf" value="tok">
<input name="username" value="">
<input name="returnto" value="/">
<input type="checkbox" name="remember" value="yes">
<input type="CHECKBOX" name="secure" value="on" checked>
<input type="radio" name="mode" value="a">
<input type="radio" name="mode" value="b" checked>
<input name="off" value="1" disabled>
<input value="nameless">
</form></body></html>`

func TestLoginForm(t *testing.T) {
	d := mustLoad(t, formDefinition)
	const (
		landing = "GET https://site.example/login.php"
		submit  = "POST https://site.example/takelogin.php?sid=S+1"
		test    = "GET https://site.example/index.php"
	)
	s := site{
		landing: with(page(formLanding), "Set-Cookie", "PHPSESSID=abc; Path=/; HttpOnly", "Set-Cookie", "seen=; Max-Age=0"),
		submit:  page("<p>Welcome</p>"),
		test:    page(`<a href="logout.php">Log out</a>`),
	}
	sent, err := login(t, d, nil, s)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, landing, submit, test)
	wantHeaders(t, sent[0], map[string]string{"Cookie": "seen=1", "Referer": "https://site.example/"})
	if !sent[0].ReplaceCookies || sent[0].FollowRedirects {
		t.Errorf("landing: replace cookies %v, follow redirects %v", sent[0].ReplaceCookies, sent[0].FollowRedirects)
	}
	post := sent[1]
	if got := string(post.Body); got != "csrf=tok&username=jo&returnto=%2F&secure=on&mode=b&password=p%40ss+word!&token=tok" {
		t.Errorf("body %q", got)
	}
	wantHeaders(t, post, map[string]string{"Cookie": "PHPSESSID=abc", "Referer": "https://site.example/login.php", "Content-Type": formContentType})
	if !post.ReplaceCookies || !post.FollowRedirects {
		t.Errorf("submit: replace cookies %v, follow redirects %v", post.ReplaceCookies, post.FollowRedirects)
	}

	s[submit] = page(`<table><tr><td class="embedded"><b>Login failed!</b> Username or password incorrect</td></tr></table>`)
	sent, err = login(t, d, nil, s)
	var rej *Rejected
	if !errors.As(err, &rej) || rej.Message != "Username or password incorrect" {
		t.Errorf("Result: %v, want the message selector's text", err)
	}
	wantRequests(t, sent, landing, submit)
	if want := `the sign-in page shows the error "Username or password incorrect"`; err.Error() != want {
		t.Errorf("error %q, want %q", err, want)
	}

	s[landing] = page(`<form id="search"></form>`)
	if _, err := login(t, d, nil, s); err == nil || !strings.Contains(err.Error(), `no form matching "form#login"`) {
		t.Errorf("Result without the form: %v", err)
	}
	s[landing] = status(http.StatusServiceUnavailable)
	if _, err := login(t, d, nil, s); err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("Result on HTTP 503: %v", err)
	}
}

func TestLoginFormSimpleCaptchaAndMultipart(t *testing.T) {
	d := mustLoad(t, "id: simple\n"+signInBase+`login:
  path: login.php
  inputs:
    username: "{{ .Config.username }}"
`)
	const (
		landing = "GET https://site.example/login.php"
		simple  = "GET https://site.example/simpleCaptcha.php?numImages=1"
		submit  = "POST https://site.example/takelogin.php"
	)
	s := site{
		landing: with(page(`<script src="/js/simpleCaptcha.js"></script><form action="takelogin.php" enctype="multipart/form-data"><input name="username"></form>`), "Set-Cookie", "s=1"),
		simple:  page(`{"images": [{"hash": "h4sh"}, {"hash": "other"}]}`),
		submit:  page(""),
	}
	sent, err := login(t, d, nil, s)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	wantRequests(t, sent, landing, simple, submit)
	wantHeaders(t, sent[1], map[string]string{"Cookie": "s=1", "Referer": "https://site.example/login.php"})
	b := "---------------------------" + "1793113445000000"
	want := "--" + b + "\r\nContent-Disposition: form-data; name=\"username\"\r\n\r\njo\r\n" +
		"--" + b + "\r\nContent-Disposition: form-data; name=\"captchaSelection\"\r\n\r\nh4sh\r\n" +
		"--" + b + "\r\nContent-Disposition: form-data; name=\"submitme\"\r\n\r\nX\r\n" +
		"--" + b + "--"
	if got := string(sent[2].Body); got != want {
		t.Errorf("body\n%q\nwant\n%q", got, want)
	}
	wantHeaders(t, sent[2], map[string]string{"Content-Type": "multipart/form-data; boundary=" + b, "Referer": "https://site.example/", "Cookie": "s=1"})
}

const captchaDefinition = "id: captcha\n" + signInBase + `login:
  path: login
  selectors: true
  inputs:
    input[type="email"]: "{{ .Config.username }}"
    input[type="password"]: "{{ .Config.password }}"
  captcha:
    type: image
    selector: img#captcha
    input: input[placeholder="Code"]
  error:
    - selector: .alert-danger
`

const captchaLanding = `<form method="post" action="/login/submit">
<input type="email" name="u_1"><input type="password" name="p_2">
<input type="text" placeholder="Code" name="cap_3"><input type="hidden" name="nonce" value="n1">
</form><img id="captcha" src="/captcha.png?r=1">`

func TestCaptchaRoundTrip(t *testing.T) {
	d := mustLoad(t, captchaDefinition)
	const (
		landing = "GET https://site.example/login"
		image   = "GET https://site.example/captcha.png?r=1"
		submit  = "POST https://site.example/login/submit"
	)
	png := []byte("\x89PNG\r\n")
	s := site{
		landing: with(page(captchaLanding), "Set-Cookie", "sid=landing"),
		image:   with(Response{Status: http.StatusOK, Body: png}, "Content-Type", "image/png", "Set-Cookie", "cap=7"),
	}
	fetch := d.CaptchaFetch(signInSettings, testClock)
	sent := s.drive(t, fetch)
	c, err := fetch.Result()
	if err != nil {
		t.Fatalf("CaptchaFetch Result: %v", err)
	}
	wantRequests(t, sent, landing, image)
	wantHeaders(t, sent[1], map[string]string{"Cookie": "sid=landing", "Referer": "https://site.example/login"})
	for _, r := range sent {
		if r.Purpose != ForCaptcha {
			t.Errorf("%s: purpose %d", r.URL, r.Purpose)
		}
	}
	if !bytes.Equal(c.Image, png) || c.ContentType != "image/png" || len(c.PreLogin) == 0 {
		t.Fatalf("Captcha = %q, %q, %d bytes of pre-login state", c.Image, c.ContentType, len(c.PreLogin))
	}

	c.Answer = "x7k2"
	s = site{submit: page("<p>Welcome</p>")}
	sent, err = login(t, d, &c, s)
	if err != nil {
		t.Fatalf("Login Result: %v", err)
	}
	wantRequests(t, sent, submit)
	if got := string(sent[0].Body); got != "u_1=jo&p_2=p%40ss+word!&cap_3=x7k2&nonce=n1" {
		t.Errorf("body %q", got)
	}
	wantHeaders(t, sent[0], map[string]string{"Cookie": "sid=landing; cap=7", "Referer": "https://site.example/login"})

	s[submit] = page(`<div class="alert-danger">Wrong code</div>`)
	_, err = login(t, d, &c, s)
	var rej *Rejected
	if !errors.As(err, &rej) || rej.Message != "Wrong code" {
		t.Errorf("Login with a wrong answer: %v, want a Rejected", err)
	}
	if _, err := login(t, d, &Captcha{PreLogin: []byte("{"), Answer: "x"}, site{}); err == nil {
		t.Error("Login resumed from a pre-login state it can't read")
	}
}

func TestLoginMeetsCaptcha(t *testing.T) {
	d := mustLoad(t, captchaDefinition)
	const (
		landing = "GET https://site.example/login"
		submit  = "POST https://site.example/login/submit"
	)
	sent, err := login(t, d, nil, site{landing: page(captchaLanding)})
	if !errors.Is(err, ErrCaptchaNeeded) {
		t.Fatalf("Result: %v, want ErrCaptchaNeeded", err)
	}
	wantRequests(t, sent, landing)

	noImage := strings.Replace(captchaLanding, `<img id="captcha" src="/captcha.png?r=1">`, "", 1)
	sent, err = login(t, d, nil, site{landing: page(noImage), submit: page("")})
	if err != nil {
		t.Fatalf("Result without a captcha image: %v", err)
	}
	wantRequests(t, sent, landing, submit)
	if got := string(sent[1].Body); got != "u_1=jo&p_2=p%40ss+word!&cap_3=&nonce=n1" {
		t.Errorf("body %q", got)
	}

	fetch := d.CaptchaFetch(signInSettings, testClock)
	site{landing: page(noImage)}.drive(t, fetch)
	if c, err := fetch.Result(); err != nil || c.Image != nil || len(c.PreLogin) == 0 {
		t.Errorf("CaptchaFetch without an image = %q, %v", c.Image, err)
	}
	fetch = mustLoad(t, postDefinition).CaptchaFetch(signInSettings, testClock)
	if _, err := fetch.Result(); err == nil {
		t.Error("CaptchaFetch ran for a post sign-in")
	}
}

func TestLoggedOut(t *testing.T) {
	d := mustLoad(t, postDefinition)
	tests := []struct {
		name string
		r    Response
		want bool
	}{
		{"signed in", page(`<a href="/logout.php">Log out</a>`), false},
		{"no logout link", page(`<form action="login.php"></form>`), true},
		{"redirect", status(http.StatusFound, "Location", "/login.php"), true},
		{"refresh header", with(page(""), "Refresh", "0; url=/login.php"), true},
		{"HTTP 401", status(http.StatusUnauthorized), true},
		{"HTTP 403", status(http.StatusForbidden), true},
		{"HTTP 404 with the link", with(Response{Status: http.StatusNotFound, Body: []byte(`<a href="logout">x</a>`)}), false},
		{"JSON", with(page(`{}`), "Content-Type", "application/json"), false},
		{"HTML content type", with(page(`{}`), "Content-Type", "text/html; charset=utf-8"), true},
	}
	for _, tt := range tests {
		if got := d.LoggedOut(tt.r); got != tt.want {
			t.Errorf("%s: LoggedOut = %v, want %v", tt.name, got, tt.want)
		}
	}
	if mustLoad(t, "id: public\n"+signInBase).LoggedOut(status(http.StatusFound)) {
		t.Error("a definition without a login block reads as logged out")
	}
}

func TestParseCookieHeader(t *testing.T) {
	got := parseCookieHeader("a=1; Path=/; expires=Wed, 21 Oct 2026 07:28:00 GMT; b=x%3D; a=3; HttpOnly; bad")
	want := []pair{{"a", "3"}, {"b", "x%3D"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseCookieHeader = %v, want %v", got, want)
	}
}
