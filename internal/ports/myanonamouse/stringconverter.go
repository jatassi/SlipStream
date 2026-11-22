// Ported from Prowlarr's src/NzbDrone.Core/Datastore/Converters/StringConverter.cs
// at 46c468c443ff3ebb86fc99347d3a15885d41d1e1. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package myanonamouse

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// stringConverter is a C# string STJson fills through Prowlarr's
// StringConverter: a JSON string, or a number's raw text. A null leaves
// it "", as STJson never hands a null to the converter.
type stringConverter string

func (s *stringConverter) UnmarshalJSON(raw []byte) error {
	switch {
	case bytes.Equal(raw, []byte("null")):
	case raw[0] == '"':
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("read a string: %w", err)
		}
		*s = stringConverter(v)
	case raw[0] == '-' || raw[0] >= '0' && raw[0] <= '9':
		*s = stringConverter(raw)
	default:
		return fmt.Errorf("%s is neither a string nor a number", raw)
	}
	return nil
}
