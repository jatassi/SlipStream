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
	variants = make([]any, len(setting.Kinds))
	for i, k := range setting.Kinds {
		variants[i] = k
	}
	return "kind", variants
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
