package rtorrent

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"strconv"
	"strings"
)

// fault is an XML-RPC fault: xmlrpc-c's codes, which both versions use.
type fault struct {
	code    int
	message string
}

func (f *fault) Error() string { return f.message }

// xmlrpc-c's fault codes.
const (
	internalError = -500
	typeError     = -501
	parseError    = -503
	noSuchMethod  = -506
)

// request is a methodCall: its method and its params, each a string, an
// int64, a []any or a map[string]any, as rTorrent's torrent::Object holds
// them. base64 arrives as its decoded string.
type request struct {
	method string
	params []any
}

// parseCall reads a methodCall. untyped allows a <value> with bare text,
// which xmlrpc-c reads as a string and tinyxml2 refuses.
func parseCall(body []byte, untyped bool) (request, error) {
	d := xml.NewDecoder(bytes.NewReader(body))
	var req request
	if err := expect(d, "methodCall"); err != nil {
		return req, err
	}
	for {
		el, ok, err := child(d)
		switch {
		case err != nil:
			return req, err
		case !ok && req.method == "":
			return req, &fault{parseError, "methodName element not found"}
		case !ok:
			return req, nil
		case el == "methodName":
			name, err := text(d)
			if err != nil || strings.TrimSpace(name) == "" {
				return req, &fault{parseError, "methodName element is empty"}
			}
			req.method = strings.TrimSpace(name)
		case el == "params":
			if req.params, err = params(d, untyped); err != nil {
				return req, err
			}
		default:
			return req, &fault{parseError, "unexpected element " + el}
		}
	}
}

// child is the name of the next element inside the current one, false at
// the current one's end.
func child(d *xml.Decoder) (name string, ok bool, err error) {
	for {
		tok, tokenErr := d.Token()
		if tokenErr != nil {
			return "", false, &fault{parseError, "the document ends inside an element"}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return t.Name.Local, true, nil
		case xml.EndElement:
			return "", false, nil
		}
	}
}

func expect(d *xml.Decoder, name string) error {
	el, ok, err := child(d)
	if err != nil {
		return err
	}
	if !ok || el != name {
		return &fault{parseError, "could not find expected element " + name}
	}
	return nil
}

// skip reads to the end of the current element.
func skip(d *xml.Decoder) error {
	if err := d.Skip(); err != nil {
		return &fault{parseError, "the document ends inside an element"}
	}
	return nil
}

// text reads character data up to the end of the element it is in.
func text(d *xml.Decoder) (string, error) {
	var b strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return "", &fault{parseError, "the document ends inside an element"}
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.EndElement:
			return b.String(), nil
		case xml.StartElement:
			return "", &fault{parseError, "unexpected element " + t.Name.Local}
		}
	}
}

// inner reads the <value> inside the current element, then its end.
func inner(d *xml.Decoder, untyped bool) (any, error) {
	if err := expect(d, "value"); err != nil {
		return nil, err
	}
	v, err := value(d, untyped)
	if err != nil {
		return nil, err
	}
	return v, skip(d)
}

func params(d *xml.Decoder, untyped bool) ([]any, error) {
	out := []any{}
	for {
		el, ok, err := child(d)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return out, nil
		case el != "param":
			return nil, &fault{parseError, "unexpected element " + el}
		}
		v, err := inner(d, untyped)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

// value reads the rest of a <value> element.
func value(d *xml.Decoder, untyped bool) (any, error) {
	var bare strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, &fault{parseError, "the document ends inside an element"}
		}
		switch t := tok.(type) {
		case xml.CharData:
			bare.Write(t)
		case xml.EndElement:
			if !untyped {
				return nil, &fault{internalError, "received unsupported value type: " + strings.TrimSpace(bare.String())}
			}
			return bare.String(), nil
		case xml.StartElement:
			v, err := typed(d, t.Name.Local, untyped)
			if err != nil {
				return nil, err
			}
			return v, skip(d)
		}
	}
}

func typed(d *xml.Decoder, kind string, untyped bool) (any, error) {
	switch kind {
	case "array":
		return array(d, untyped)
	case "struct":
		return members(d, untyped)
	}
	s, err := text(d)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "string":
		return s, nil
	case "i4", "i8", "int":
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return nil, &fault{typeError, "unable to parse integer value"}
		}
		return n, nil
	case "boolean":
		switch strings.TrimSpace(s) {
		case "0":
			return int64(0), nil
		case "1":
			return int64(1), nil
		}
		return nil, &fault{typeError, "unknown boolean value: " + s}
	case "base64":
		b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
		if err != nil {
			return nil, &fault{parseError, "base64 value isn't base64"}
		}
		return string(b), nil
	}
	return nil, &fault{internalError, "received unsupported value type: " + kind}
}

func array(d *xml.Decoder, untyped bool) ([]any, error) {
	if err := expect(d, "data"); err != nil {
		return nil, &fault{parseError, "could not find expected data element in array"}
	}
	out := []any{}
	for {
		_, ok, err := child(d)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return out, skip(d)
		}
		v, err := value(d, untyped)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

func members(d *xml.Decoder, untyped bool) (map[string]any, error) {
	out := map[string]any{}
	for {
		_, ok, err := child(d)
		switch {
		case err != nil:
			return nil, err
		case !ok:
			return out, nil
		}
		if err := expect(d, "name"); err != nil {
			return nil, &fault{parseError, "struct member missing name element"}
		}
		name, err := text(d)
		if err != nil {
			return nil, err
		}
		if out[name], err = inner(d, untyped); err != nil {
			return nil, err
		}
	}
}

// style is how a version writes its responses: xmlrpc-c's lines and i4
// fault codes, or tinyxml2's one compact line.
type style struct {
	lines   bool
	faultI4 bool
}

func (s style) nl() string {
	if s.lines {
		return "\r\n"
	}
	return ""
}

func (s style) response(v any) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + s.nl() + "<methodResponse>" + s.nl() + "<params>" + s.nl() + "<param>")
	s.value(&b, v)
	b.WriteString("</param>" + s.nl() + "</params>" + s.nl() + "</methodResponse>" + s.nl())
	return b.Bytes()
}

func (s style) fault(f *fault) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + s.nl() + "<methodResponse>" + s.nl() + "<fault>" + s.nl())
	s.value(&b, s.faultStruct(f))
	b.WriteString(s.nl() + "</fault>" + s.nl() + "</methodResponse>" + s.nl())
	return b.Bytes()
}

// faultCode is an int that xmlrpc-c writes as i4.
type faultCode int

func (s style) faultStruct(f *fault) map[string]any {
	return map[string]any{"faultCode": faultCode(f.code), "faultString": f.message}
}

func (s style) value(b *bytes.Buffer, v any) {
	b.WriteString("<value>")
	switch v := v.(type) {
	case string:
		b.WriteString("<string>")
		_ = xml.EscapeText(b, []byte(v))
		b.WriteString("</string>")
	case int64:
		b.WriteString("<i8>" + strconv.FormatInt(v, 10) + "</i8>")
	case faultCode:
		kind := "i8"
		if s.faultI4 {
			kind = "i4"
		}
		b.WriteString("<" + kind + ">" + strconv.Itoa(int(v)) + "</" + kind + ">")
	case []any:
		b.WriteString("<array><data>" + s.nl())
		for _, x := range v {
			s.value(b, x)
			b.WriteString(s.nl())
		}
		b.WriteString("</data></array>")
	case map[string]any:
		// The only struct rTorrent writes is a fault in a system.multicall.
		b.WriteString("<struct>" + s.nl())
		for _, k := range []string{"faultCode", "faultString"} {
			b.WriteString("<member><name>" + k + "</name>" + s.nl())
			s.value(b, v[k])
			b.WriteString("</member>" + s.nl())
		}
		b.WriteString("</struct>")
	}
	b.WriteString("</value>")
}
