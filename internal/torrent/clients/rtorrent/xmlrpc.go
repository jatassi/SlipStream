package rtorrent

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// fault is an XML-RPC fault rTorrent answered.
type fault struct {
	Code    int64
	Message string
}

func (f *fault) Error() string { return fmt.Sprintf("rTorrent fault %d: %s", f.Code, f.Message) }

// call is one method call of a system.multicall.
type call struct {
	method string
	params []any
}

// encode is a methodCall of method with params, each a string, []byte as
// base64, []any or call.
func encode(method string, params []any) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0"?><methodCall><methodName>`)
	_ = xml.EscapeText(&b, []byte(method))
	b.WriteString("</methodName><params>")
	for _, p := range params {
		b.WriteString("<param>")
		writeValue(&b, p)
		b.WriteString("</param>")
	}
	b.WriteString("</params></methodCall>")
	return b.Bytes()
}

func writeValue(b *bytes.Buffer, v any) {
	b.WriteString("<value>")
	switch v := v.(type) {
	case string:
		b.WriteString("<string>")
		_ = xml.EscapeText(b, []byte(v))
		b.WriteString("</string>")
	case []byte:
		b.WriteString("<base64>" + base64.StdEncoding.EncodeToString(v) + "</base64>")
	case []any:
		b.WriteString("<array><data>")
		for _, x := range v {
			writeValue(b, x)
		}
		b.WriteString("</data></array>")
	case call:
		b.WriteString("<struct><member><name>methodName</name>")
		writeValue(b, v.method)
		b.WriteString("</member><member><name>params</name>")
		writeValue(b, v.params)
		b.WriteString("</member></struct>")
	default:
		panic(fmt.Sprintf("rtorrent: no XML-RPC value for %T", v))
	}
	b.WriteString("</value>")
}

var errShape = errors.New("the response isn't an XML-RPC methodResponse")

// decode reads a methodResponse into its one value: a string, an int64,
// a []any or a map[string]any. A fault is a *fault.
func decode(body []byte) (any, error) {
	d := xml.NewDecoder(bytes.NewReader(body))
	if err := enter(d, "methodResponse"); err != nil {
		return nil, err
	}
	el, _, err := child(d)
	if err != nil {
		return nil, err
	}
	switch el {
	case "params":
		if err := enter(d, "param"); err != nil {
			return nil, err
		}
		if err := enter(d, "value"); err != nil {
			return nil, err
		}
		return value(d)
	case "fault":
		if err := enter(d, "value"); err != nil {
			return nil, err
		}
		v, err := value(d)
		if err != nil {
			return nil, err
		}
		if f, ok := asFault(v); ok {
			return nil, f
		}
	}
	return nil, errShape
}

// asFault is v as a fault struct, which system.multicall answers in place
// of a one-value array for a call that failed.
func asFault(v any) (*fault, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	code, coded := m["faultCode"].(int64)
	message, worded := m["faultString"].(string)
	if !coded || !worded {
		return nil, false
	}
	return &fault{Code: code, Message: message}, true
}

// child is the name of the next element inside the current one, false at
// the current one's end.
func child(d *xml.Decoder) (name string, ok bool, err error) {
	for {
		tok, tokenErr := d.Token()
		if tokenErr != nil {
			return "", false, errShape
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return t.Name.Local, true, nil
		case xml.EndElement:
			return "", false, nil
		}
	}
}

func enter(d *xml.Decoder, name string) error {
	el, ok, err := child(d)
	if err != nil || !ok || el != name {
		return errShape
	}
	return nil
}

// skip reads to the end of the current element.
func skip(d *xml.Decoder) error {
	if err := d.Skip(); err != nil {
		return errShape
	}
	return nil
}

func text(d *xml.Decoder) (string, error) {
	var b strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return "", errShape
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.EndElement:
			return b.String(), nil
		case xml.StartElement:
			return "", errShape
		}
	}
}

// value reads the rest of a <value>. A value with no type is a string.
func value(d *xml.Decoder) (any, error) {
	var bare strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, errShape
		}
		switch t := tok.(type) {
		case xml.CharData:
			bare.Write(t)
		case xml.EndElement:
			return bare.String(), nil
		case xml.StartElement:
			v, err := typed(d, t.Name.Local)
			if err != nil {
				return nil, err
			}
			return v, skip(d)
		}
	}
}

func typed(d *xml.Decoder, kind string) (any, error) {
	switch kind {
	case "array":
		return array(d)
	case "struct":
		return members(d)
	}
	s, err := text(d)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "string":
		return s, nil
	case "i4", "i8", "int", "boolean":
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("read the client's %s %q: %w", kind, s, err)
		}
		return n, nil
	case "base64":
		b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
		if err != nil {
			return nil, fmt.Errorf("read the client's base64: %w", err)
		}
		return string(b), nil
	}
	return nil, fmt.Errorf("the client's response holds an XML-RPC %s, which SlipStream doesn't read", kind)
}

func array(d *xml.Decoder) ([]any, error) {
	if err := enter(d, "data"); err != nil {
		return nil, err
	}
	out := []any{}
	for {
		el, ok, err := child(d)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return out, skip(d)
		case el != "value":
			return nil, errShape
		}
		v, err := value(d)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

func members(d *xml.Decoder) (map[string]any, error) {
	out := map[string]any{}
	for {
		_, ok, err := child(d)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return out, nil
		}
		if err := enter(d, "name"); err != nil {
			return nil, err
		}
		name, err := text(d)
		if err != nil {
			return nil, err
		}
		if err := enter(d, "value"); err != nil {
			return nil, err
		}
		if out[name], err = value(d); err != nil {
			return nil, err
		}
		if err := skip(d); err != nil {
			return nil, err
		}
	}
}
