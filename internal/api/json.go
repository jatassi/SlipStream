package api

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// inUTC writes every time.Time as its UTC instant. A Node keeps its
// process's zone for the words people read, so a time.Now() carries the
// Node's offset until it reaches the wire.
var inUTC = jsonv2.WithMarshalers(jsonv2.MarshalFunc(func(t time.Time) ([]byte, error) {
	return t.UTC().MarshalJSON()
}))

// marshal is json.Marshal with every time.Time in UTC.
func marshal(v any) ([]byte, error) {
	return jsonv2.Marshal(v, json.DefaultOptionsV1(), inUTC) //nolint:wrapcheck // every caller wraps it with what it encoded
}

// jsonFormat is Huma's JSON format with every time.Time in UTC.
var jsonFormat = huma.Format{
	Marshal: func(w io.Writer, v any) error {
		b, err := jsonv2.Marshal(v, json.DefaultOptionsV1(), jsontext.EscapeForHTML(false), inUTC)
		if err != nil {
			return fmt.Errorf("api: encode %T: %w", v, err)
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return fmt.Errorf("api: write a response: %w", err)
		}
		return nil
	},
	Unmarshal: json.Unmarshal,
}
