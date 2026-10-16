// Package settingapi is the wire side of setting descriptors: the field
// union SettingField renders and the code that refuses a value outside
// its setting's range. Each owner's operations serve and save its own
// settings with them.
package settingapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/setting"
)

// Field names the union of every kind of setting field, discriminated on
// kind: api.OneOf[Field] holds any one.
type Field struct{}

func (Field) Variants() (discriminator string, variants []any) {
	return "kind", []any{setting.DurationField{}, setting.CountField{}, setting.ChoiceField{}, setting.WindowField{}}
}

func FieldOf(f any) api.OneOf[Field] { return api.NewOneOf[Field](f) }

// ErrOutOfRange answers a value its setting refuses, with the setting
// control's sentence as the detail.
var ErrOutOfRange = api.NewCode(http.StatusUnprocessableEntity, "setting_out_of_range", "The value is outside the setting's range")

// Refused turns a setting's RangeError into ErrOutOfRange and passes any
// other error through.
func Refused(err error) error {
	if r, ok := errors.AsType[*setting.RangeError](err); ok {
		return ErrOutOfRange.Because(r.Sentence)
	}
	return err
}
