// Ported from Prowlarr's src/NzbDrone.Common/Serializer/Newtonsoft.Json/Json.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package serializer is the leniency of Prowlarr's two JSON readers, as
// value types a port's response struct declares in place of string, int,
// float and bool: String, Int and Bool read as Newtonsoft's Json does,
// STJInt, STJFloat and STJBool as STJson does. A C# nullable is a pointer
// to one, which a JSON null leaves nil.
package serializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var null = []byte("null")

// String is a C# string Newtonsoft fills: from a JSON string, a number's
// text, or "True" or "False". A null leaves it as it was, as
// NullValueHandling.Ignore does.
type String string

func (s *String) UnmarshalJSON(raw []byte) error {
	switch {
	case bytes.Equal(raw, null):
	case raw[0] == '"':
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("read a string: %w", err)
		}
		*s = String(v)
	case bytes.Equal(raw, []byte("true")):
		*s = "True"
	case bytes.Equal(raw, []byte("false")):
		*s = "False"
	default:
		*s = String(raw)
	}
	return nil
}

// Int is a C# int or long Newtonsoft fills: from a JSON number or a
// string of one. A null or an empty string leaves it as it was.
type Int int64

func (n *Int) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, null) {
		return nil
	}
	s, err := unquoted(raw)
	if err != nil || s == "" && raw[0] == '"' {
		return err
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%s isn't a whole number", raw)
	}
	*n = Int(v)
	return nil
}

// Bool is a C# bool Newtonsoft fills: from true or false, a number,
// nonzero for true, or "true" or "false" in any case. A null leaves it
// as it was.
type Bool bool

func (b *Bool) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, null) {
		return nil
	}
	s, err := unquoted(raw)
	if err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "true":
		*b = true
		return nil
	case "false":
		*b = false
		return nil
	}
	if raw[0] != '"' {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			*b = v != 0
			return nil
		}
	}
	return fmt.Errorf("%s isn't a boolean", raw)
}

// unquoted is a JSON string's value, or any other token's text.
func unquoted(raw []byte) (string, error) {
	if raw[0] != '"' {
		return string(raw), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("read a string: %w", err)
	}
	return s, nil
}
