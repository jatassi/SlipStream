package cardigann

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Unrunnable says why SlipStream can't run a definition, as a Go error
// string such as `it logs in with "twofactor", which SlipStream doesn't
// run`.
type Unrunnable struct {
	Reason string
	err    error
}

func (u *Unrunnable) Error() string { return u.Reason }
func (u *Unrunnable) Unwrap() error { return u.err }

// errUnsupported marks what SlipStream doesn't run, as against a definition
// that fails to read or compile.
var errUnsupported = errors.New("unsupported")

func unrunnable(err error, format string, args ...any) *Unrunnable {
	return &Unrunnable{Reason: fmt.Sprintf(format, args...), err: err}
}

func unsupported(format string, args ...any) *Unrunnable {
	return unrunnable(errUnsupported, format, args...)
}

// Definition is a Cardigann definition, read and compiled.
type Definition struct {
	def        *cardigannDefinition
	charset    charset
	categories categories
	modes      map[string][]string
	uses       Features
}

// Load reads a definition, applies Prowlarr's load defaults and compiles
// every template, regex and selector in it. It fails with *Unrunnable when
// SlipStream can't run the definition.
func Load(src []byte) (*Definition, error) {
	d, _, err := load(src)
	return d, err
}

// load is Load that also lists the templated selector branches that don't
// compile.
func load(src []byte) (*Definition, []string, error) {
	def, err := decodeDefinition(src)
	if err != nil {
		return nil, nil, decodeUnrunnable(err)
	}
	applyDefaults(def)
	d, u := prepare(def)
	if u == nil {
		u = checkSupport(d)
	}
	if u != nil {
		return nil, nil, u
	}
	u, broken := checkCompiles(def)
	if u != nil {
		return nil, broken, u
	}
	return d, broken, nil
}

func decodeUnrunnable(err error) *Unrunnable {
	var fe *fieldError
	if errors.As(err, &fe) {
		return unrunnable(err, "its %s can't be read: %s", fe.path, fe.err)
	}
	var ye *yamlError
	if errors.As(err, &ye) {
		err = ye.err
	}
	return unrunnable(err, "it isn't valid YAML: %s", firstLine(err.Error()))
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

// applyDefaults is CleanIndexerDefinition
// (IndexerDefinitionUpdateService.cs:246-283).
func applyDefaults(def *cardigannDefinition) {
	if def.Settings == nil {
		def.Settings = []settingsField{
			{Name: "username", Label: "Username", Type: "text"},
			{Name: "password", Label: "Password", Type: "password"},
		}
	}
	if def.Encoding == "" {
		def.Encoding = "UTF-8"
	}
	if def.Login != nil && def.Login.Method == "" {
		def.Login.Method = "form"
	}
	if def.Search.Path != nil {
		p := searchPathBlock{Path: def.Search.Path}
		p.setDefaults()
		def.Search.Paths = append(def.Search.Paths, p)
	}
}

// prepare resolves what the CardigannBase constructor and the caps parser
// read: the encoding, the categories and the search modes.
func prepare(def *cardigannDefinition) (*Definition, *Unrunnable) {
	if len(def.Links) == 0 {
		return nil, unrunnable(errors.New("no links"), "it has no links")
	}
	if len(def.Search.Paths) == 0 {
		return nil, unrunnable(errors.New("no search paths"), "it has no search path")
	}
	cs, err := lookupCharset(def.Encoding)
	if err != nil {
		return nil, unrunnable(err, "it uses the encoding %q, which SlipStream doesn't support", def.Encoding)
	}
	d := &Definition{def: def, charset: cs, uses: usedFeatures(def)}
	if d.categories, err = buildCategories(def.Caps); err != nil {
		return nil, unrunnable(err, "its categories can't be read: %s", err)
	}
	if d.modes, err = parseSearchModes(def.Caps.Modes); err != nil {
		return nil, unrunnable(err, "its search modes can't be read: %s", err)
	}
	return d, nil
}

var (
	loginMethods = []string{"form", "post", "cookie", "get", "oneurl"}
	settingTypes = []string{"text", "password", "checkbox", "select", "info", "info_cookie", "info_flaresolverr", "info_useragent", "info_category_8000"}
)

// checkSupport refuses a definition that uses a login method, captcha or
// setting type SlipStream doesn't run.
func checkSupport(d *Definition) *Unrunnable {
	def := d.def
	if def.Login != nil && !slices.Contains(loginMethods, def.Login.Method) {
		return unsupported("it logs in with %q, which SlipStream doesn't run", def.Login.Method)
	}
	if c := def.Login.captcha(); c != nil && c.Type != "image" {
		return unsupported("it asks for a %q captcha, and SlipStream shows only image captchas", c.Type)
	}
	for _, s := range def.Settings {
		if !slices.Contains(settingTypes, s.Type) {
			return unsupported("it has a setting of type %q, which SlipStream can't show", s.Type)
		}
	}
	return nil
}

func (l *loginBlock) captcha() *captchaBlock {
	if l == nil {
		return nil
	}
	return l.Captcha
}
