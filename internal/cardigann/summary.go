package cardigann

import (
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
)

// Summary describes a definition for the Indexer form and the catalog.
type Summary struct {
	ID, Name, Description, Language string
	Privacy                         Privacy
	Links, LegacyLinks              []string
	Encoding                        string
	// RequestDelay is the definition's own, before the Node's floor.
	RequestDelay time.Duration
	// Fields are the settings in definition order, the captcha last.
	Fields  []Field
	Login   LoginClass
	Captcha bool
	Caps    Caps
	Uses    Features
}

// Privacy is a definition's type.
type Privacy string

// The privacy types. Prowlarr reads any other type as semi-private.
const (
	Private     Privacy = "private"
	SemiPrivate Privacy = "semi-private"
	Public      Privacy = "public"
)

// LoginClass is whether an Indexer for the definition holds a login
// session.
type LoginClass int

// The login classes, as the spec's Session() defines them.
const (
	// NoLogin is a definition without a login block.
	NoLogin LoginClass = iota
	// StatelessLogin is a get login without a password setting or a
	// captcha: an API key or passkey check.
	StatelessLogin
	// SessionLogin is every other login.
	SessionLogin
)

// Field is one setting of the Indexer form.
type Field struct {
	// Name may hold hyphens or start with a digit.
	Name    string
	Label   string
	Type    FieldType
	Default string
	// Options are a select's, in key order.
	Options []Option
	// Help is an info setting's text, third-party prose.
	Help string
}

// FieldType is a setting's type.
type FieldType string

// The setting types. CaptchaField is the one Prowlarr adds for a login
// captcha.
const (
	TextField             FieldType = "text"
	PasswordField         FieldType = "password"
	CheckboxField         FieldType = "checkbox"
	SelectField           FieldType = "select"
	InfoField             FieldType = "info"
	InfoCookieField       FieldType = "info_cookie"
	InfoFlareSolverrField FieldType = "info_flaresolverr"
	InfoUserAgentField    FieldType = "info_useragent"
	InfoCategory8000Field FieldType = "info_category_8000"
	CaptchaField          FieldType = "cardigannCaptcha"
)

// Option is one choice of a select.
type Option struct {
	Value, Label string
}

// Caps are a definition's search modes and categories.
type Caps struct {
	// Modes are the parameters each mode takes, such as "movie-search":
	// ["q", "imdbid"].
	Modes map[string][]string
	// Categories are the Newznab categories the definition maps to, the
	// defaults' first.
	Categories []newznab.ID
}

// Summary describes d.
func (d *Definition) Summary() Summary {
	def := d.def
	s := Summary{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Language:    def.Language,
		Privacy:     privacy(def.Type),
		Links:       slices.Clone(def.Links),
		LegacyLinks: slices.Clone(def.LegacyLinks),
		Encoding:    def.Encoding,
		Fields:      fields(def),
		Login:       loginClass(def),
		Captcha:     def.Login.captcha() != nil,
		Caps:        Caps{Modes: cloneModes(d.modes), Categories: d.categories.ids()},
		Uses:        slices.Clone(d.uses),
	}
	if def.RequestDelay != nil {
		s.RequestDelay = time.Duration(*def.RequestDelay * float64(time.Second))
	}
	return s
}

func privacy(t string) Privacy {
	if p := Privacy(t); p == Private || p == Public {
		return p
	}
	return SemiPrivate
}

func loginClass(def *cardigannDefinition) LoginClass {
	switch {
	case def.Login == nil:
		return NoLogin
	case def.Login.Method != methodGet || def.Login.Captcha != nil:
		return SessionLogin
	case slices.ContainsFunc(def.Settings, func(s settingsField) bool { return s.Type == "password" }):
		return SessionLogin
	}
	return StatelessLogin
}

func fields(def *cardigannDefinition) []Field {
	out := make([]Field, 0, len(def.Settings)+1)
	for _, s := range def.Settings {
		f := Field{Name: s.Name, Label: s.Label, Type: FieldType(s.Type), Default: s.Default}
		if strings.HasPrefix(s.Type, "info") {
			f.Help, f.Default = s.Default, ""
		}
		for _, o := range s.Options {
			f.Options = append(f.Options, Option{Value: o.Key, Label: o.Value})
		}
		slices.SortStableFunc(f.Options, func(a, b Option) int { return strings.Compare(a.Value, b.Value) })
		out = append(out, f)
	}
	if def.Login.captcha() != nil {
		out = append(out, Field{Name: "cardigannCaptcha", Label: "CAPTCHA", Type: CaptchaField})
	}
	return out
}

func cloneModes(m map[string][]string) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = slices.Clone(v)
	}
	return out
}
