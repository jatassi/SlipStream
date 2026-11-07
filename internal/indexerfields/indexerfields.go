// Package indexerfields is the settings form of an Indexer, whether a
// catalog definition or a ported site declares it: the field model, which
// values an Indexer stores and seals, and whether a value fits its field.
package indexerfields

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Field is one setting of the Indexer form.
type Field struct {
	// Name may hold hyphens or start with a digit.
	Name    string
	Label   string
	Type    Type
	Default string
	// Options are a select's or a multiselect's, in order.
	Options []Option
	// Help is an info setting's text, or a ported site's help sentence.
	Help string
	// Min and Max bound a number, nil for no bound.
	Min *int `json:",omitempty"`
	Max *int `json:",omitempty"`
	// Required is a ported site's credential, which an Indexer can't save
	// empty.
	Required bool `json:",omitempty"`
}

// Type is a setting's type.
type Type string

// The setting types. Captcha is the one Prowlarr adds for a login captcha.
// Number and Multiselect are a ported site's only.
const (
	Text             Type = "text"
	Password         Type = "password"
	Checkbox         Type = "checkbox"
	Select           Type = "select"
	Number           Type = "number"
	Multiselect      Type = "multiselect"
	Info             Type = "info"
	InfoCookie       Type = "info_cookie"
	InfoFlareSolverr Type = "info_flaresolverr"
	InfoUserAgent    Type = "info_useragent"
	InfoCategory8000 Type = "info_category_8000"
	Captcha          Type = "cardigannCaptcha"
)

// Option is one choice of a select or a multiselect.
type Option struct {
	Value, Label string
}

// Stored reports whether an Indexer stores a value for f. Info fields are
// help text, and a captcha is answered at each sign-in.
func Stored(f Field) bool {
	return slices.Contains([]Type{Text, Password, Checkbox, Select, Number, Multiselect}, f.Type)
}

// Sealed reports whether an Indexer seals f's value: every text and
// password field but the username, since most API key fields are text.
func Sealed(f Field) bool {
	return (f.Type == Text || f.Type == Password) && f.Name != "username"
}

// OptionLabel is the label of f's option value, "" when f doesn't offer it.
func OptionLabel(f Field, value string) string {
	for _, o := range f.Options {
		if o.Value == value {
			return o.Label
		}
	}
	return ""
}

func offers(f Field, value string) bool {
	return slices.ContainsFunc(f.Options, func(o Option) bool { return o.Value == value })
}

// ValueError is a setting value its field doesn't take.
type ValueError struct {
	Field string
	// Sentence says what the field takes, for the form.
	Sentence string
}

func (e *ValueError) Error() string { return "setting " + e.Field + " doesn't fit its field" }

// Fits refuses an empty v for a required field, v for a select or
// multiselect that doesn't offer it, a checkbox that isn't true or false,
// and a number that isn't a whole number between f's bounds. A
// multiselect's value is its option values separated by commas.
func Fits(f Field, v string) error {
	if f.Required && v == "" {
		return empty(f)
	}
	switch f.Type {
	case Select:
		if !offers(f, v) {
			return &ValueError{Field: f.Name, Sentence: "Choose one of the options for " + f.Label + "."}
		}
	case Multiselect:
		if v == "" {
			return nil
		}
		for value := range strings.SplitSeq(v, ",") {
			if !offers(f, value) {
				return &ValueError{Field: f.Name, Sentence: "Choose from the options for " + f.Label + "."}
			}
		}
	case Checkbox:
		if v != "true" && v != "false" {
			return &ValueError{Field: f.Name, Sentence: f.Label + " is true or false."}
		}
	case Number:
		n, err := strconv.Atoi(v)
		if err != nil || f.Min != nil && n < *f.Min || f.Max != nil && n > *f.Max {
			return &ValueError{Field: f.Name, Sentence: numberSentence(f)}
		}
	case Text, Password, Info, InfoCookie, InfoFlareSolverr, InfoUserAgent, InfoCategory8000, Captcha:
	}
	return nil
}

func empty(f Field) *ValueError {
	return &ValueError{Field: f.Name, Sentence: f.Label + " is required."}
}

func numberSentence(f Field) string {
	switch {
	case f.Min != nil && f.Max != nil:
		return fmt.Sprintf("%s is a whole number from %d to %d.", f.Label, *f.Min, *f.Max)
	case f.Min != nil:
		return fmt.Sprintf("%s is a whole number of %d or more.", f.Label, *f.Min)
	case f.Max != nil:
		return fmt.Sprintf("%s is a whole number of %d or less.", f.Label, *f.Max)
	}
	return f.Label + " is a whole number."
}

// Split splits values over the stored fields into the plain ones and the
// sealed ones, on top of the plain and sealed values saved before. A
// setting values leaves out keeps its saved value, as does a sealed one
// sent empty. A setting with no value takes its default when the Node
// runs it, and a required one with no value fails.
func Split(fields []Field, values, plainBefore, sealedBefore map[string]string) (plain, sealed map[string]string, err error) {
	plain, sealed = map[string]string{}, map[string]string{}
	byName := map[string]Field{}
	for _, f := range fields {
		if !Stored(f) {
			continue
		}
		byName[f.Name] = f
		if v, ok := plainBefore[f.Name]; ok && !Sealed(f) {
			plain[f.Name] = v
		}
		if v, ok := sealedBefore[f.Name]; ok && Sealed(f) {
			sealed[f.Name] = v
		}
	}
	for name, v := range values {
		f, ok := byName[name]
		if !ok {
			return nil, nil, &ValueError{Field: name, Sentence: "This Indexer has no setting named " + name + "."}
		}
		if Sealed(f) && v == "" {
			continue
		}
		if err := Fits(f, v); err != nil {
			return nil, nil, err
		}
		if Sealed(f) {
			sealed[name] = v
		} else {
			plain[name] = v
		}
	}
	for _, f := range fields {
		if f.Required && plain[f.Name] == "" && sealed[f.Name] == "" {
			return nil, nil, empty(f)
		}
	}
	return plain, sealed, nil
}
