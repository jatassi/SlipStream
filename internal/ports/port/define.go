package port

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/indexerfields"
)

// Floor is the least spacing a port may declare, the Node's own.
const Floor = 2 * time.Second

var encodings = []string{"", "iso-8859-1", "windows-1251"}

// Define is the one way to make a Port. It reflects S into Fields,
// derives Auth, Key and SignInWords, and panics when the parts disagree,
// so a misdeclared port fails at package init, in every test that loads
// the registry.
func Define[S any](e Entry, impl Impl[S]) Port {
	settings, err := settingsOf(reflect.TypeFor[S]())
	if err == nil {
		err = check(e, impl, settings)
	}
	if err != nil {
		panic(fmt.Sprintf("port %q: %v", e.ID, err))
	}
	e.Auth = impl.Login.auth()
	e.Fields = make([]Field, 0, len(settings)+1)
	for _, s := range settings {
		e.Fields = append(e.Fields, s.Field)
	}
	if e.Auth == FormWithCode {
		key := authenticatorKey
		key.Help = fmt.Sprintf(key.Help, e.Name)
		e.Fields = append(e.Fields, key)
	}
	e.Key = words(settings, func(s setting) bool { return s.key })
	e.SignInWords = words(settings, func(s setting) bool { return s.signin })
	return Port{entry: e, run: &typed[S]{impl: impl, settings: settings, login: familyOf[S](impl.Login)}}
}

func check[S any](e Entry, impl Impl[S], settings []setting) error {
	return errors.Join(checkEntry(e), checkModes(e.Capabilities, impl), checkLogin(impl, settings), checkNames(settings))
}

func checkEntry(e Entry) error {
	var errs []error
	if e.ID == "" || e.Name == "" || e.Description == "" {
		errs = append(errs, errors.New("an entry needs an ID, a Name and a Description"))
	}
	if !slices.Contains([]Privacy{Private, SemiPrivate, Public}, e.Privacy) {
		errs = append(errs, fmt.Errorf("privacy %q is none of private, semi-private and public", e.Privacy))
	}
	if len(e.IndexerURLs) == 0 {
		errs = append(errs, errors.New("an entry needs at least one address"))
	}
	for _, raw := range slices.Concat(e.IndexerURLs, e.LegacyURLs) {
		if u, err := url.Parse(raw); err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
			errs = append(errs, fmt.Errorf("address %q isn't an http or https URL", raw))
		}
	}
	if e.RateLimit != 0 && e.RateLimit < Floor {
		errs = append(errs, fmt.Errorf("the rate limit %s is below the %s floor", e.RateLimit, Floor))
	}
	if e.PageSize < 0 || e.QueryLimit.Count < 0 || e.GrabLimit.Count < 0 {
		errs = append(errs, errors.New("a page size or limit is negative"))
	}
	if !slices.Contains(encodings, e.Encoding) {
		errs = append(errs, fmt.Errorf("encoding %q is none of %q", e.Encoding, encodings))
	}
	if e.Capabilities.Categories == nil {
		errs = append(errs, errors.New("the capabilities need a category mapping"))
	}
	if e.Fields != nil || e.Auth != 0 || e.Key != "" || e.SignInWords != "" {
		errs = append(errs, errors.New("the entry sets Fields, Auth, Key or SignInWords, which Define derives"))
	}
	return errors.Join(errs...)
}

// checkModes refuses a mode with no generator and a generator with no
// mode, and an Impl with no parser.
func checkModes[S any](c Capabilities, impl Impl[S]) error {
	var errs []error
	for _, m := range []struct {
		name     string
		params   []Param
		declared bool
		gen      bool
	}{
		{"movie", c.MovieSearchParams, c.MovieSearchParams != nil, impl.Movie != nil},
		{"tv", c.TvSearchParams, c.TvSearchParams != nil, impl.TV != nil},
		{"search", c.SearchParams, c.SearchParams != nil || impl.Basic != nil, impl.Basic != nil},
	} {
		if m.declared != m.gen {
			errs = append(errs, fmt.Errorf("the %s mode needs both its params and its generator", m.name))
		}
		if m.params != nil && len(m.params) == 0 {
			errs = append(errs, fmt.Errorf("the %s mode declares no params", m.name))
		}
	}
	if impl.Basic == nil {
		errs = append(errs, errors.New("a port needs a Basic generator, which Recent and Test run"))
	}
	if impl.Parse == nil {
		errs = append(errs, errors.New("a port needs a parser"))
	}
	return errors.Join(errs...)
}

func checkLogin[S any](impl Impl[S], settings []setting) error {
	if impl.Login == nil {
		return errors.New("a port needs a Login family")
	}
	if t := impl.Login.settings(); t != nil && t != reflect.TypeFor[S]() {
		return fmt.Errorf("the Login family reads %s, the port's settings are %s", t, reflect.TypeFor[S]())
	}
	signin := slices.ContainsFunc(settings, func(s setting) bool { return s.signin })
	key := slices.ContainsFunc(settings, func(s setting) bool { return s.key })
	sealed := slices.ContainsFunc(settings, func(s setting) bool { return indexerfields.Sealed(s.Field) })
	switch l := impl.Login.(type) {
	case NoLogin:
		if sealed && !key {
			return errors.New("a NoLogin port with a secret marks the secret it sends as key")
		}
	case FormLogin[S]:
		return signedIn(l.DoLogin != nil, signin)
	case TokenLogin[S]:
		return signedIn(l.DoLogin != nil, signin)
	case CodeLogin[S]:
		return signedIn(l.DoLogin != nil, signin)
	case CookieLogin:
	}
	return nil
}

func signedIn(doLogin, signin bool) error {
	if !doLogin || !signin {
		return errors.New("a sign-in family needs DoLogin and the fields it posts tagged signin")
	}
	return nil
}

func checkNames(settings []setting) error {
	seen := map[string]bool{authenticatorKey.Name: true}
	for _, s := range settings {
		if seen[s.Name] {
			return fmt.Errorf("setting %q is declared twice, or is the authenticator key's", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}
