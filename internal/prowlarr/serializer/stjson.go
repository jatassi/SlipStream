// Ported from Prowlarr's src/NzbDrone.Common/Serializer/System.Text.Json/BooleanConverter.cs, src/NzbDrone.Common/Serializer/System.Text.Json/STJson.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package serializer

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// STJInt is a C# int or long STJson fills, whose NumberHandling is
// AllowReadingFromString: a JSON number or a string of one. A null is an
// error, as for C#'s non-nullable int.
type STJInt int64

func (n *STJInt) UnmarshalJSON(raw []byte) error {
	s, err := stjNumber(raw)
	if err != nil {
		return err
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%s isn't a whole number", raw)
	}
	*n = STJInt(v)
	return nil
}

// STJFloat is a C# double STJson fills: a JSON number or a string of one.
// A null is an error.
type STJFloat float64

func (n *STJFloat) UnmarshalJSON(raw []byte) error {
	s, err := stjNumber(raw)
	if err != nil {
		return err
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("%s isn't a number", raw)
	}
	*n = STJFloat(v)
	return nil
}

// STJBool is a C# bool STJson fills through Prowlarr's BooleanConverter:
// true, false, 1 or 0. Anything else, a null or a string included, is an
// error.
type STJBool bool

func (b *STJBool) UnmarshalJSON(raw []byte) error {
	switch string(raw) {
	case "true", "1":
		*b = true
	case "false", "0":
		*b = false
	default:
		return fmt.Errorf("%s isn't true, false, 1 or 0", raw)
	}
	return nil
}

// stjNumber is the text of a JSON number, or of a string that holds only
// one.
func stjNumber(raw []byte) (string, error) {
	s, err := unquoted(raw)
	if err != nil {
		return "", err
	}
	if !json.Valid([]byte(s)) {
		return "", fmt.Errorf("%s isn't a number", raw)
	}
	return s, nil
}
