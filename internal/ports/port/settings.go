// Ported from Prowlarr's src/NzbDrone.Core/Annotations/FieldDefinitionAttribute.cs, src/NzbDrone.Core/Indexers/Settings/CookieTorrentBaseSettings.cs, src/NzbDrone.Core/Indexers/Settings/NoAuthTorrentBaseSettings.cs, src/NzbDrone.Core/Indexers/Settings/UserPassTorrentBaseSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package port

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/indexerfields"
)

// A port's settings are a struct, as its C# Settings class is, and the
// struct is the one list of its fields: Define reflects it into
// Entry.Fields, and each job decodes the opened values into a new one.
//
// The tag `port:"<name>[,key][,signin]"` names the stored value, and
// `label`, `help`, `default`, `min` and `max` fill the form. The Go type
// sets the field type:
//
//	string              text, sealed unless the name is "username"
//	Password            password, sealed
//	bool                checkbox
//	int                 number, bounded by min and max
//	T with Options()    select, T an int or string type
//	[]T with Options()  multiselect, stored as comma-separated option values
//
// key marks the credentials account_refused names, signin the ones
// credentials_rejected names, and either makes the value required. An
// embedded struct contributes its fields in place, as a C# base settings
// class does. A port embeds the credential structs below and never types
// a credential's label. It gives a credential the site's own help with a
// tag on the embedded struct: help on a struct of one field, or
// help.<name>, naming the stored value, on any:
//
//	port.PIDField      `help:"Find your PID on your My Account page."`
//	port.APIUserFields `help.apiUser:"Find it under Edit Profile."`
type (
	Field  = indexerfields.Field
	Option = indexerfields.Option
)

// Password is a password-typed value.
type Password string

// Optioned is a select's option type: C#'s enum with FieldOption labels.
type Optioned interface{ Options() []Option }

// UserPass is UserPassTorrentBaseSettings.
type UserPass struct {
	Username string   `port:"username,signin" label:"Username"`
	Password Password `port:"password,signin" label:"Password"`
}

// EmailPass is the SpeedApp family's sign-in.
type EmailPass struct {
	Email    string   `port:"email,signin" label:"Email"`
	Password Password `port:"password,signin" label:"Password"`
}

type APIKeyField struct {
	APIKey string `port:"apiKey,key" label:"API key"`
}

type PasskeyField struct {
	Passkey string `port:"passkey,key" label:"Passkey"`
}

type RSSKeyField struct {
	RSSKey string `port:"rssKey,key" label:"RSS key"`
}

// APIUserFields are PassThePopcorn's pair.
type APIUserFields struct {
	APIUser string `port:"apiUser,key" label:"API user"`
	APIKey  string `port:"apiKey,key" label:"API key"`
}

// PIDField is the Avistaz family's third sign-in field.
type PIDField struct {
	PID string `port:"pid,signin" label:"PID"`
}

// PINField is PreToMe's third sign-in field.
type PINField struct {
	PIN string `port:"pin,signin" label:"PIN"`
}

// CookieField is a cookie a site's own settings take in place of its
// sign-in form, as GreatPosterWall's does for an account with two-factor
// sign-in, so it isn't required.
type CookieField struct {
	Cookie string `port:"cookie" label:"Cookie"`
}

// authenticatorKey is the field Define adds for CodeLogin, its help
// formatted with the site's name.
var authenticatorKey = Field{
	Name: "authenticatorKey", Label: "Authenticator key", Type: indexerfields.Password,
	Help: "%s shows this key when you turn on two-factor sign-in, usually under the QR code. With it, SlipStream makes each code and signs in again automatically. Leave it empty to type a code whenever a sign-in needs one.",
}

// setting is one field of a settings struct and where its value goes.
type setting struct {
	Field
	index       []int
	key, signin bool
}

var (
	optionedType = reflect.TypeFor[Optioned]()
	passwordType = reflect.TypeFor[Password]()
)

// settingsOf reflects the struct t into its settings, in declaration
// order, embedded structs in place.
func settingsOf(t reflect.Type) ([]setting, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("settings type %s isn't a struct", t)
	}
	var out []setting
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct && sf.Tag.Get("port") == "" {
			inner, err := settingsOf(sf.Type)
			if err == nil {
				err = helpOf(sf.Tag, inner)
			}
			if err != nil {
				return nil, fmt.Errorf("settings field %s: %w", sf.Name, err)
			}
			for _, s := range inner {
				s.index = append([]int{i}, s.index...)
				out = append(out, s)
			}
			continue
		}
		s, err := settingOf(sf)
		if err != nil {
			return nil, fmt.Errorf("settings field %s: %w", sf.Name, err)
		}
		s.index = []int{i}
		out = append(out, s)
	}
	return out, nil
}

func settingOf(sf reflect.StructField) (setting, error) {
	tag, ok := sf.Tag.Lookup("port")
	if !ok || !sf.IsExported() {
		return setting{}, errors.New("an exported field with a port tag is a setting, and a settings struct holds nothing else")
	}
	name, flags, _ := strings.Cut(tag, ",")
	s := setting{Field: Field{Name: name, Label: sf.Tag.Get("label"), Help: sf.Tag.Get("help"), Default: sf.Tag.Get("default")}}
	for flag := range strings.SplitSeq(flags, ",") {
		switch flag {
		case "":
		case "key":
			s.key = true
		case "signin":
			s.signin = true
		default:
			return setting{}, fmt.Errorf("unknown port tag option %q", flag)
		}
	}
	if s.Name == "" || s.Label == "" {
		return setting{}, errors.New("a setting needs a name and a label")
	}
	s.Required = s.key || s.signin
	var err error
	if s.Type, s.Options, err = typeOf(sf.Type); err != nil {
		return setting{}, err
	}
	if s.Min, s.Max, err = bounds(sf); err != nil {
		return setting{}, err
	}
	if s.Type != indexerfields.Number && (s.Min != nil || s.Max != nil) {
		return setting{}, errors.New("only a number takes min and max")
	}
	if s.Default != "" {
		if err := indexerfields.Fits(s.Field, s.Default); err != nil {
			return setting{}, fmt.Errorf("default %q doesn't fit: %w", s.Default, err)
		}
	}
	return s, nil
}

// helpOf sets the help that the tag of an embedded struct gives its
// fields.
func helpOf(tag reflect.StructTag, inner []setting) error {
	pairs, err := tagPairs(tag)
	if err != nil {
		return err
	}
	for key, text := range pairs {
		name, named := strings.CutPrefix(key, "help.")
		i := 0
		switch {
		case key == "help" && len(inner) != 1:
			return errors.New("a struct of more than one field takes help.<name> tags, not help")
		case key != "help" && !named:
			return fmt.Errorf("an embedded struct takes only help tags, not %s", key)
		case text == "":
			return fmt.Errorf("the %s tag is empty", key)
		case named:
			i = slices.IndexFunc(inner, func(s setting) bool { return s.Name == name })
		}
		if i < 0 {
			return fmt.Errorf("the %s tag names no field of the struct", key)
		}
		inner[i].Help = text
	}
	return nil
}

// tagPairs is every key:"value" pair of tag. It fails where
// reflect.StructTag.Lookup would stop reading.
func tagPairs(tag reflect.StructTag) (map[string]string, error) {
	out := map[string]string{}
	malformed := fmt.Errorf("the tag %q isn't key:\"value\" pairs", tag)
	rest := strings.TrimLeft(string(tag), " ")
	for rest != "" {
		key, after, ok := strings.Cut(rest, ":")
		end := closingQuote(after)
		if !ok || key == "" || strings.ContainsAny(key, " \"") || end < 0 {
			return nil, malformed
		}
		v, err := strconv.Unquote(after[:end+1])
		if err != nil {
			return nil, malformed
		}
		out[key] = v
		rest = strings.TrimLeft(after[end+1:], " ")
	}
	return out, nil
}

// closingQuote is the index of the quote that ends the quoted string s
// starts with, -1 when s doesn't start with one or never ends it.
func closingQuote(s string) int {
	if !strings.HasPrefix(s, `"`) {
		return -1
	}
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

func typeOf(t reflect.Type) (indexerfields.Type, []Option, error) {
	switch {
	case t == passwordType:
		return indexerfields.Password, nil, nil
	case t.Implements(optionedType):
		opts, err := optionsOf(t)
		return indexerfields.Select, opts, err
	case t.Kind() == reflect.Slice && t.Elem().Implements(optionedType):
		opts, err := optionsOf(t.Elem())
		return indexerfields.Multiselect, opts, err
	case t.Kind() == reflect.String:
		return indexerfields.Text, nil, nil
	case t.Kind() == reflect.Bool:
		return indexerfields.Checkbox, nil, nil
	case t.Kind() == reflect.Int:
		return indexerfields.Number, nil, nil
	}
	return "", nil, fmt.Errorf("type %s is no setting type", t)
}

func optionsOf(t reflect.Type) ([]Option, error) {
	if t.Kind() != reflect.Int && t.Kind() != reflect.String {
		return nil, fmt.Errorf("option type %s isn't an int or string type", t)
	}
	o, ok := reflect.Zero(t).Interface().(Optioned)
	if !ok {
		return nil, fmt.Errorf("option type %s has Options on a pointer, not on the value", t)
	}
	opts := o.Options()
	if len(opts) == 0 {
		return nil, fmt.Errorf("option type %s offers no options", t)
	}
	for _, o := range opts {
		if _, err := scalar(t, o.Value); err != nil || o.Label == "" {
			return nil, fmt.Errorf("option %q of %s needs a label and a value of its type", o.Value, t)
		}
	}
	return opts, nil
}

func bounds(sf reflect.StructField) (lo, hi *int, err error) {
	for tag, at := range map[string]**int{"min": &lo, "max": &hi} {
		v, ok := sf.Tag.Lookup(tag)
		if !ok {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, nil, fmt.Errorf("%s %q isn't a whole number", tag, v)
		}
		*at = &n
	}
	if lo != nil && hi != nil && *lo > *hi {
		return nil, nil, fmt.Errorf("min %d is above max %d", *lo, *hi)
	}
	return lo, hi, nil
}

// decode fills a new S from values, one job's opened settings. A missing
// value takes its field's default. A value that doesn't fit is a bug in
// the Controller's checks, so the job fails.
func decode[S any](settings []setting, values map[string]string) (S, error) {
	var out S
	v := reflect.ValueOf(&out).Elem()
	for _, s := range settings {
		raw, ok := values[s.Name]
		if !ok {
			raw = s.Default
		}
		f := v.FieldByIndex(s.index)
		var err error
		switch s.Type {
		case indexerfields.Multiselect:
			err = decodeList(f, raw)
		case indexerfields.Checkbox:
			var b bool
			if raw != "" {
				b, err = strconv.ParseBool(raw)
			}
			f.SetBool(b)
		case indexerfields.Text, indexerfields.Password, indexerfields.Select, indexerfields.Number:
			var x reflect.Value
			if x, err = scalar(f.Type(), raw); err == nil {
				f.Set(x)
			}
		case indexerfields.Info, indexerfields.InfoCookie, indexerfields.InfoFlareSolverr, indexerfields.InfoUserAgent,
			indexerfields.InfoCategory8000, indexerfields.Captcha:
		}
		if err != nil {
			return out, fmt.Errorf("setting %s: %w", s.Name, err)
		}
	}
	return out, nil
}

func decodeList(f reflect.Value, raw string) error {
	if raw == "" {
		return nil
	}
	for item := range strings.SplitSeq(raw, ",") {
		x, err := scalar(f.Type().Elem(), item)
		if err != nil {
			return err
		}
		f.Set(reflect.Append(f, x))
	}
	return nil
}

// scalar is raw as a value of the string or int type t, zero for "".
func scalar(t reflect.Type, raw string) (reflect.Value, error) {
	x := reflect.New(t).Elem()
	switch {
	case raw == "":
	case t.Kind() == reflect.String:
		x.SetString(raw)
	case t.Kind() == reflect.Int:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return x, fmt.Errorf("%q isn't a whole number", raw)
		}
		x.SetInt(int64(n))
	default:
		return x, fmt.Errorf("type %s holds no setting", t)
	}
	return x, nil
}

// words names the fields of settings that want holds, as
// indexerfields.Words does.
func words(settings []setting, want func(setting) bool) string {
	var fields []Field
	for _, s := range settings {
		if want(s) {
			fields = append(fields, s.Field)
		}
	}
	return indexerfields.Words(fields)
}
