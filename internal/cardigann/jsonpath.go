package cardigann

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// jsonToken is a JSON token as Newtonsoft's JToken holds it after JToken.Parse:
// objects keep their key order, integers and floats stay apart, and a
// string in ISO 8601 form is a date.
type jsonToken struct {
	kind   jsonKind
	text   string
	float  float64
	bool   bool
	date   time.Time
	offset bool
	items  []*jsonToken
	keys   []string
	parent *jsonToken
}

type jsonKind int

const (
	jsonNull jsonKind = iota
	jsonBool
	jsonInteger
	jsonFloat
	jsonString
	jsonDate
	jsonArray
	jsonObject
)

func parseJSON(body string) (*jsonToken, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	v, err := readJSON(dec)
	if err != nil {
		return nil, fmt.Errorf("parse the JSON response: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("parse the JSON response: text follows the JSON value")
	}
	return v, nil
}

func readJSON(dec *json.Decoder) (*jsonToken, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err //nolint:wrapcheck // parseJSON wraps it
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			return readArray(dec)
		}
		return readObject(dec)
	case string:
		if d, offset, ok := parseISODate(t); ok {
			return &jsonToken{kind: jsonDate, text: t, date: d, offset: offset}, nil
		}
		return &jsonToken{kind: jsonString, text: t}, nil
	case json.Number:
		return jsonNumber(string(t)), nil
	case bool:
		return &jsonToken{kind: jsonBool, bool: t}, nil
	}
	return &jsonToken{kind: jsonNull}, nil
}

func jsonNumber(s string) *jsonToken {
	if !strings.ContainsAny(s, ".eE") {
		if n, ok := new(big.Int).SetString(s, 10); ok {
			return &jsonToken{kind: jsonInteger, text: n.String()}
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return &jsonToken{kind: jsonFloat, float: f}
}

func readArray(dec *json.Decoder) (*jsonToken, error) {
	arr := &jsonToken{kind: jsonArray}
	for dec.More() {
		v, err := readJSON(dec)
		if err != nil {
			return nil, err
		}
		v.parent = arr
		arr.items = append(arr.items, v)
	}
	_, err := dec.Token()
	return arr, err //nolint:wrapcheck // parseJSON wraps it
}

func readObject(dec *json.Decoder) (*jsonToken, error) {
	obj := &jsonToken{kind: jsonObject}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err //nolint:wrapcheck // parseJSON wraps it
		}
		key, _ := tok.(string)
		v, err := readJSON(dec)
		if err != nil {
			return nil, err
		}
		v.parent = obj
		if i := indexOf(obj.keys, key); i >= 0 {
			obj.items[i] = v
			continue
		}
		obj.keys = append(obj.keys, key)
		obj.items = append(obj.items, v)
	}
	_, err := dec.Token()
	return obj, err //nolint:wrapcheck // parseJSON wraps it
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// parseISODate is Newtonsoft's DateTimeParser: yyyy-MM-ddTHH:mm:ss, up to
// seven fraction digits, and Z or an offset. The string must be 19 to 40
// characters and start with a digit.
func parseISODate(s string) (t time.Time, hasOffset, ok bool) {
	if len(s) < 19 || len(s) > 40 || !isASCIIDigit(rune(s[0])) || s[10] != 'T' {
		return time.Time{}, false, false
	}
	p := isoParser{s: s}
	year, month, day := p.digits(4), p.sep('-'), p.sep('-')
	p.expect('T')
	hour, minute, second := p.digits(2), p.sep(':'), p.sep(':')
	nanos := p.fraction()
	offset, hasOffset := p.zone()
	if p.bad || p.i != len(s) || !validDate(year, month, day) || hour > 24 || minute > 59 || second > 59 ||
		(hour == 24 && (minute != 0 || second != 0 || nanos != 0)) {
		return time.Time{}, false, false
	}
	t = time.Date(year, time.Month(month), day, hour, minute, second, nanos, time.UTC)
	if hasOffset {
		t = t.Add(-time.Duration(offset) * time.Second)
	}
	return t, hasOffset, true
}

type isoParser struct {
	s   string
	i   int
	bad bool
}

func (p *isoParser) digits(n int) int {
	if p.i+n > len(p.s) {
		p.bad = true
		return 0
	}
	v, err := strconv.Atoi(p.s[p.i : p.i+n])
	if err != nil || strings.ContainsAny(p.s[p.i:p.i+n], "+-") {
		p.bad = true
	}
	p.i += n
	return v
}

func (p *isoParser) expect(c byte) {
	if p.i >= len(p.s) || p.s[p.i] != c {
		p.bad = true
		return
	}
	p.i++
}

// sep reads c and the two digits after it.
func (p *isoParser) sep(c byte) int {
	p.expect(c)
	return p.digits(2)
}

func (p *isoParser) fraction() int {
	if p.i >= len(p.s) || p.s[p.i] != '.' {
		return 0
	}
	p.i++
	v, n := 0, 0
	for n < 7 && p.i < len(p.s) && isASCIIDigit(rune(p.s[p.i])) {
		v = v*10 + int(p.s[p.i]-'0')
		n++
		p.i++
	}
	for range 9 - n {
		v *= 10
	}
	return v
}

// zone reads Z or a signed hour and minute offset, in seconds east of UTC.
func (p *isoParser) zone() (int, bool) {
	if p.i >= len(p.s) {
		return 0, false
	}
	switch c := p.s[p.i]; c {
	case 'Z', 'z':
		p.i++
		return 0, true
	case '+', '-':
		p.i++
		hours := p.digits(2)
		if p.i < len(p.s) && p.s[p.i] == ':' {
			p.i++
		}
		minutes := 0
		if p.i < len(p.s) {
			minutes = p.digits(2)
		}
		off := hours*3600 + minutes*60
		if c == '-' {
			off = -off
		}
		return off, true
	}
	p.bad = true
	return 0, false
}

// Text is j as Newtonsoft's Value<string>() gives it: null for a JSON null,
// True or False for a boolean, MM/dd/yyyy HH:mm:ss for a date, and an
// error for an object or array.
func (j *jsonToken) Text(loc *time.Location) (value, error) {
	switch j.kind {
	case jsonNull:
		return nullValue, nil
	case jsonObject, jsonArray:
		return nullValue, errors.New("a JSON object or array isn't a value")
	case jsonBool, jsonInteger, jsonFloat, jsonString, jsonDate:
	}
	return textValue(j.scalarString(loc)), nil
}

func (j *jsonToken) scalarString(loc *time.Location) string {
	switch j.kind {
	case jsonBool:
		if j.bool {
			return "True"
		}
		return "False"
	case jsonFloat:
		return formatDotNetDouble(j.float)
	case jsonDate:
		d := j.date
		if j.offset {
			d = d.In(loc)
		}
		return d.Format("01/02/2006 15:04:05")
	case jsonNull, jsonArray, jsonObject, jsonInteger, jsonString:
	}
	return j.text
}

// String is JToken.ToString(): a scalar's text, or an object or array as
// indented JSON.
func (j *jsonToken) String() string {
	if j.kind == jsonObject || j.kind == jsonArray {
		var b strings.Builder
		j.writeIndented(&b, 0)
		return b.String()
	}
	return j.scalarString(time.UTC)
}

func (j *jsonToken) writeIndented(b *strings.Builder, depth int) {
	open, closing := "[", "]"
	if j.kind == jsonObject {
		open, closing = "{", "}"
	}
	if len(j.items) == 0 {
		b.WriteString(open + closing)
		return
	}
	b.WriteString(open)
	for i, item := range j.items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n" + strings.Repeat("  ", depth+1))
		if j.kind == jsonObject {
			writeJSONString(b, j.keys[i])
			b.WriteString(": ")
		}
		item.writeValue(b, depth+1)
	}
	b.WriteString("\n" + strings.Repeat("  ", depth) + closing)
}

func (j *jsonToken) writeValue(b *strings.Builder, depth int) {
	switch j.kind {
	case jsonObject, jsonArray:
		j.writeIndented(b, depth)
	case jsonNull:
		b.WriteString("null")
	case jsonBool:
		b.WriteString(strconv.FormatBool(j.bool))
	case jsonInteger:
		b.WriteString(j.text)
	case jsonFloat:
		s := formatDotNetDouble(j.float)
		if !strings.ContainsAny(s, ".E") {
			s += ".0"
		}
		b.WriteString(s)
	case jsonString, jsonDate:
		writeJSONString(b, j.text)
	}
}

// writeJSONString writes s quoted as Newtonsoft's default escaping does:
// the quote, the backslash, control characters, U+0085, U+2028 and U+2029.
func writeJSONString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\f':
			b.WriteString(`\f`)
		case '\b':
			b.WriteString(`\b`)
		case '\u0085', '\u2028', '\u2029':
			fmt.Fprintf(b, `\u%04x`, r)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// formatDotNetDouble is double.ToString with the invariant culture: the
// shortest digits that read back, in E notation from 1E+15 up and below
// 0.0001.
func formatDotNetDouble(f float64) string {
	if f == 0 {
		return "0"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64)
	mant, expText, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expText)
	if exp >= 15 || exp < -4 {
		sign := "+"
		if exp < 0 {
			sign, exp = "-", -exp
		}
		return strings.ToUpper(mant) + "E" + sign + fmt.Sprintf("%02d", exp)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// jsonPath is a compiled SelectToken path: property names and array
// indexes.
type jsonPath []jsonStep

type jsonStep struct {
	name  string
	index int
	isIdx bool
}

// compileJSONPath reads the SelectToken subset the catalog uses: an
// optional $, dotted names, ['name'] and [n].
func compileJSONPath(path string) (jsonPath, error) {
	var steps jsonPath
	s := strings.TrimPrefix(path, "$")
	for i := 0; i < len(s); {
		switch s[i] {
		case '.':
			i++
		case '[':
			end := strings.IndexByte(s[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("the JSON path %q has an unclosed [", path)
			}
			step, err := bracketStep(s[i+1:i+end], path)
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
			i += end + 1
		default:
			end := strings.IndexAny(s[i:], ".[")
			if end < 0 {
				end = len(s) - i
			}
			name := s[i : i+end]
			if strings.ContainsAny(name, "]()*?") {
				return nil, fmt.Errorf("the JSON path %q isn't supported", path)
			}
			steps = append(steps, jsonStep{name: name})
			i += end
		}
	}
	return steps, nil
}

func bracketStep(inner, path string) (jsonStep, error) {
	if len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') && inner[len(inner)-1] == inner[0] {
		return jsonStep{name: inner[1 : len(inner)-1]}, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(inner))
	if err != nil {
		return jsonStep{}, fmt.Errorf("the JSON path %q isn't supported", path)
	}
	return jsonStep{index: n, isIdx: true}, nil
}

// SelectToken follows path from j, nil when a step finds nothing.
func (j *jsonToken) SelectToken(path string) (*jsonToken, error) {
	p, err := compileJSONPath(path)
	if err != nil {
		return nil, err
	}
	return j.follow(p), nil
}

func (j *jsonToken) follow(p jsonPath) *jsonToken {
	cur := j
	for _, step := range p {
		cur = cur.step(step)
		if cur == nil {
			return nil
		}
	}
	return cur
}

func (j *jsonToken) step(s jsonStep) *jsonToken {
	if s.isIdx {
		if j.kind != jsonArray || s.index < 0 || s.index >= len(j.items) {
			return nil
		}
		return j.items[s.index]
	}
	if j.kind != jsonObject {
		return nil
	}
	if i := indexOf(j.keys, s.name); i >= 0 {
		return j.items[i]
	}
	return nil
}

var jsonFilterRe = regexp2.MustCompile(`\:(?<filter>.+?)\((?<key>.+?)\)(?=:|\z)`, regexp2.None)

// fieldSelector is JsonParseFieldSelector (CardigannBase.cs:860-934): the
// path before the first ':', and whether j passes the :has, :not and
// :contains filters after it.
func (j *jsonToken) fieldSelector(sel string) (text string, found bool, err error) {
	base, _, _ := strings.Cut(sel, ":")
	target := j
	if strings.TrimSpace(base) != "" {
		t, err := j.SelectToken(base)
		if err != nil || t == nil {
			return "", false, err
		}
		target = t
	}
	m, err := jsonFilterRe.FindStringMatch(sel)
	for ; m != nil && err == nil; m, err = jsonFilterRe.FindNextMatch(m) {
		pass, err := target.passes(group(m, "filter"), group(m, "key"))
		if err != nil || !pass {
			return "", false, err
		}
	}
	if err != nil {
		return "", false, fmt.Errorf("read the JSON selector %q: %w", sel, err)
	}
	return base, true, nil
}

func (j *jsonToken) passes(filter, key string) (bool, error) {
	switch filter {
	case "has", "not":
		found, err := j.has(key)
		if err != nil {
			return false, err
		}
		return found == (filter == "has"), nil
	case "contains":
		return strings.Contains(j.String(), key), nil
	}
	return true, nil
}

func (j *jsonToken) has(key string) (bool, error) {
	nested, err := jsonFilterRe.MatchString(key)
	if err != nil {
		return false, fmt.Errorf("read the JSON selector %q: %w", key, err)
	}
	if nested {
		_, ok, err := j.fieldSelector(key)
		return ok, err
	}
	t, err := j.SelectToken(key)
	return t != nil, err
}

// Rows is JsonParseRowsSelector (CardigannBase.cs:841-858) over a selector
// already rendered: the array the path before the first ':' names, keeping
// the objects that pass the filters after it. It reports false where
// Prowlarr returns null: the path finds no array or a row isn't an object.
func (j *jsonToken) Rows(selector string) ([]*jsonToken, bool) {
	base, _, _ := strings.Cut(selector, ":")
	filters := selector[len(base):]
	arr, err := j.SelectToken(base)
	if err != nil || arr == nil || arr.kind != jsonArray {
		return nil, false
	}
	var rows []*jsonToken
	for _, row := range arr.items {
		if row.kind != jsonObject {
			return nil, false
		}
		_, ok, err := row.fieldSelector(filters)
		if err != nil {
			return nil, false
		}
		if ok {
			rows = append(rows, row)
		}
	}
	return rows, true
}

func jsonJoinArray(s string, p jsonPath, sep string) (string, error) {
	obj, err := parseJSON(s)
	if err != nil {
		return "", err
	}
	if obj.kind != jsonObject {
		return "", errors.New("the value isn't a JSON object")
	}
	arr := obj.follow(p)
	if arr == nil {
		return "", errors.New("the JSON path finds nothing")
	}
	parts := make([]string, len(arr.items))
	for i, item := range arr.items {
		parts[i] = item.String()
	}
	return strings.Join(parts, sep), nil
}
