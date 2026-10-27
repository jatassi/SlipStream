package cardigann

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
)

const fullDefinition = "\xef\xbb\xbf---\n" + `id: testsite
name: Test Site
description: "A site	for tests"
language: en-US
type: private
encoding: UTF-8
requestDelay: 2.5
followredirect: true
testlinktorrent: false
unknownkey: ignored
links:
  - https://test.example/
legacylinks:
  - https://old.example/
certificates:
  - aa:bb
caps:
  categorymappings:
    - {id: 1, cat: Movies, desc: "Movies"}
    - {id: 2, cat: TV/HD, desc: "TV HD", default: true}
    - {id: 3, cat: Not/AThing, desc: "skipped"}
  modes:
    search: [q]
    tv-search: [q, season, ep, IMDBID]
    movie-search: [q]
settings:
  - name: username
    type: text
    label: Username
  - name: password
    type: password
    label: Password
  - name: sort
    type: select
    label: Sort
    default: 2
    options:
      2: seeders
      1: added
      True: "yes"
      1: newest
  - name: info
    type: info
    label: About
    default: Read this.
  - name: freeleech
    type: checkbox
    label: Freeleech only
    default: false
login:
  path: login.php
  method: post
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  error:
    - selector: div.error
  test:
    path: index.php
    selector: a[href*="logout"]
  captcha:
    type: image
    selector: img#captcha
    input: captcha
search:
  paths:
    - path: browse.php
    - path: api/search
      response:
        type: json
  keywordsfilters:
    - name: re_replace
      args: ["[^a-z]", " "]
  inputs:
    q: "{{ .Keywords }}"
  rows:
    selector: tr.row{{ if .Config.freeleech }}.free{{ end }}
  fields:
    title:
      selector: a.title
    title:
      text: "{{ .Result.title }} extra"
    category:
      selector: a.cat
      case:
        'a[href*="1"]': Movies
        "*": TV
    size:
      selector: td.size
      filters:
        - name: nosuchfilter
        - name: append
          args: " GB"
download:
  selectors:
    - selector: a.dl
      attribute: href
`

// everyFeaturePorted lifts the unported gate for a test of what Load reads
// from a definition that uses a feature SlipStream can't run yet.
func everyFeaturePorted(t *testing.T) {
	t.Helper()
	saved := unported
	unported = nil
	t.Cleanup(func() { unported = saved })
}

func mustLoad(t *testing.T, src string) *Definition {
	t.Helper()
	d, err := Load([]byte(src))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return d
}

func TestLoadModel(t *testing.T) {
	everyFeaturePorted(t)
	def := mustLoad(t, fullDefinition).def
	if def.Description != "A site\tfor tests" {
		t.Errorf("description = %q", def.Description)
	}
	if def.TestLinkTorrent || !def.FollowRedirect {
		t.Errorf("testlinktorrent %v, followredirect %v", def.TestLinkTorrent, def.FollowRedirect)
	}
	sort := def.Settings[2]
	var keys []string
	for _, o := range sort.Options {
		keys = append(keys, o.Key+"="+o.Value)
	}
	if got := strings.Join(keys, ","); got != "2=seeders,1=newest,True=yes" {
		t.Errorf("options = %s", got)
	}
	var fields []string
	for _, f := range def.Search.Fields {
		fields = append(fields, f.Key)
	}
	if got := strings.Join(fields, ","); got != "title,title,category,size" {
		t.Errorf("fields = %s", got)
	}
	category := def.Search.Fields[2].Value
	if category.Case[0].Key != `a[href*="1"]` || category.Case[1].Key != "*" {
		t.Errorf("case keys = %q, %q", category.Case[0].Key, category.Case[1].Key)
	}
	if len(def.Search.Paths) != 2 || def.Search.Paths[0].QuerySeparator != "&" || !def.Search.Paths[0].InheritInputs {
		t.Errorf("paths = %+v", def.Search.Paths)
	}
	size, err := def.Search.Fields[3].Value.Filters.Apply(textValue("12"), testEnv(t))
	if err != nil || size.String() != "12 GB" {
		t.Errorf("size filters = %q, %v", size.String(), err)
	}
}

func TestLoadDefaults(t *testing.T) {
	everyFeaturePorted(t)
	def := mustLoad(t, `id: min
links: [https://min.example/]
caps:
  modes:
    search: [q]
login:
  path: login
  inputs:
    user: "{{ .Config.username }}"
search:
  path: browse
  rows:
    selector: tr
  fields:
    title:
      selector: a
`).def
	if def.Encoding != "UTF-8" || def.Login.Method != "form" || !def.TestLinkTorrent {
		t.Errorf("encoding %q, login method %q, testlinktorrent %v", def.Encoding, def.Login.Method, def.TestLinkTorrent)
	}
	if len(def.Settings) != 2 || def.Settings[0].Name != "username" || def.Settings[1].Type != "password" {
		t.Errorf("settings = %+v", def.Settings)
	}
	if len(def.Search.Paths) != 1 || def.Search.Paths[0].Path.Source() != "browse" {
		t.Errorf("paths = %+v", def.Search.Paths)
	}
}

func TestSummary(t *testing.T) {
	everyFeaturePorted(t)
	s := mustLoad(t, fullDefinition).Summary()
	want := Summary{
		ID: "testsite", Name: "Test Site", Description: "A site\tfor tests", Language: "en-US",
		Privacy:      Private,
		Links:        []string{"https://test.example/"},
		LegacyLinks:  []string{"https://old.example/"},
		Encoding:     "UTF-8",
		RequestDelay: 2500 * time.Millisecond,
		Fields: []Field{
			{Name: "username", Label: "Username", Type: TextField},
			{Name: "password", Label: "Password", Type: PasswordField},
			{Name: "sort", Label: "Sort", Type: SelectField, Default: "2", Options: []Option{{"1", "newest"}, {"2", "seeders"}, {"True", "yes"}}},
			{Name: "info", Label: "About", Type: InfoField, Help: "Read this."},
			{Name: "freeleech", Label: "Freeleech only", Type: CheckboxField, Default: "false"},
			{Name: "cardigannCaptcha", Label: "CAPTCHA", Type: CaptchaField},
		},
		Login:   SessionLogin,
		Captcha: true,
		Caps: Caps{
			Modes:      map[string][]string{"search": {"q"}, "tv-search": {"q", "season", "ep", "imdbid"}, "movie-search": {"q"}},
			Categories: []newznab.ID{5040, 100002, 2000, 100001},
		},
		Uses: Features{LoginPost, LoginError, LoginTest, LoginTestPath, LoginCaptcha, ResponseHTML, ResponseJSON,
			SearchPaths, KeywordsFilters, FollowRedirect, DownloadSelectors, NoTestLinkTorrent, Certificates},
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("Summary =\n%+v\nwant\n%+v", s, want)
	}
}

func TestLoginClass(t *testing.T) {
	everyFeaturePorted(t)
	base := "id: x\nlinks: [https://x/]\ncaps:\n  modes:\n    search: [q]\nsearch:\n  path: s\n"
	tests := []struct {
		name, extra string
		want        LoginClass
	}{
		{"no login", "", NoLogin},
		{"get without a password", "settings:\n  - {name: apikey, type: text}\nlogin:\n  method: get\n  path: api\n", StatelessLogin},
		{"get with a password", "settings:\n  - {name: password, type: password}\nlogin:\n  method: get\n  path: api\n", SessionLogin},
		{"cookie", "settings:\n  - {name: cookie, type: text}\nlogin:\n  method: cookie\n", SessionLogin},
		{"form by default", "login:\n  path: login\n", SessionLogin},
	}
	for _, tt := range tests {
		d, err := Load([]byte(base + tt.extra))
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := d.Summary().Login; got != tt.want {
			t.Errorf("%s: login class %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestPrivacy(t *testing.T) {
	for in, want := range map[string]Privacy{"private": Private, "public": Public, "semi-private": SemiPrivate, "": SemiPrivate, "other": SemiPrivate} {
		if got := privacy(in); got != want {
			t.Errorf("privacy(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadUnrunnable(t *testing.T) {
	base := "id: x\nlinks: [https://x/]\ncaps:\n  modes:\n    search: [q]\n"
	search := "search:\n  path: s\n  rows:\n    selector: tr\n"
	tests := []struct {
		name, src   string
		unsupported bool
		reason      string
	}{
		{"not YAML", "id: [", false, "it isn't valid YAML"},
		{"wrong shape", "id: x\nlinks: {a: b}\n", false, "its links can't be read"},
		{"no links", "id: x\n" + search, false, "it has no links"},
		{"no search path", base, false, "it has no search path"},
		{"unknown encoding", base + "encoding: klingon\n" + search, false, `it uses the encoding "klingon"`},
		{"search mode", "id: x\nlinks: [https://x/]\ncaps:\n  modes:\n    search: [q, imdbid]\n" + search, false, "its search modes can't be read"},
		{"login method", base + search + "login:\n  method: twofactor\n", true, `it logs in with "twofactor", which SlipStream doesn't run`},
		{"captcha type", base + search + "login:\n  path: l\n  captcha:\n    type: recaptcha\n", true, `it asks for a "recaptcha" captcha`},
		{"unported feature", base + search + "download:\n  selectors:\n    - selector: a\n", true, "it uses download selectors, which SlipStream can't run yet"},
		{"setting type", base + search + "settings:\n  - {name: a, type: multi-select}\n", true, `it has a setting of type "multi-select"`},
		{"undeclared setting", base + search + "  inputs:\n    q: \"{{ .Config.nope }}\"\n", false, `a template reads the setting "nope"`},
		{"bad selector", base + "search:\n  path: s\n  rows:\n    selector: \"tr[\"\n", false, "its selector \"tr[\" at search.rows.selector can't be read"},
		{"bad template", base + search + "  fields:\n    title:\n      selector: \"{{ if .True }}a\"\n", false, "its search.fields.title.selector can't be read"},
		{"bad filter", base + search + "  fields:\n    title:\n      selector: a\n      filters:\n        - name: regexp\n          args: \"(\"\n", false, "can't be read"},
		{"bad JSON path", base + "search:\n  paths:\n    - path: s\n      response: {type: json}\n  rows:\n    selector: \"a[\"\n", false, "can't be read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load([]byte(tt.src))
			var u *Unrunnable
			if !errors.As(err, &u) {
				t.Fatalf("Load: %v, want *Unrunnable", err)
			}
			if !strings.Contains(u.Reason, tt.reason) {
				t.Errorf("reason %q, want it to hold %q", u.Reason, tt.reason)
			}
			if errors.Is(err, errUnsupported) != tt.unsupported {
				t.Errorf("unsupported = %v, want %v", !tt.unsupported, tt.unsupported)
			}
			if u.Reason != strings.TrimRight(u.Reason, ".") || u.Reason[0] < 'a' || u.Reason[0] > 'z' {
				t.Errorf("reason %q isn't a Go error string", u.Reason)
			}
		})
	}
}

func TestLoadBranches(t *testing.T) {
	src := `id: x
links: [https://x/]
settings:
  - {name: free, type: checkbox}
caps:
  modes:
    search: [q]
search:
  path: s
  rows:
    selector: "tr{{ if .Config.free }}[src$=\\\"/x\\\"]{{ else }}.row{{ end }}"
`
	d, broken, err := load([]byte(src))
	if err != nil || d == nil {
		t.Fatalf("a definition with one good branch: %v", err)
	}
	if len(broken) != 1 || !strings.Contains(broken[0], "search.rows.selector") {
		t.Errorf("broken = %q", broken)
	}
}

func TestEscapeQuotedTabs(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a: \"x\ty\"", `a: "x\ty"`},
		{"a: 'x\ty'", "a: 'x\ty'"},
		{"a: \"x\\\"\ty\"", `a: "x\"\ty"`},
		{"a: x # \"c\td\"", "a: x # \"c\td\""},
		{"a:\tb", "a:\tb"},
	}
	for _, tt := range tests {
		if got := string(escapeQuotedTabs([]byte(tt.in))); got != tt.want {
			t.Errorf("escapeQuotedTabs(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
