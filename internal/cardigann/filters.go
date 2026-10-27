package cardigann

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// env is what evaluating a template, a filter or a selector reads besides
// its input.
type env struct {
	Scope   scope
	Clock   Clock
	charset charset
}

// filterBlock is one entry of a filters list, its arguments checked when the
// definition loads.
type filterBlock struct {
	Name  string
	Args  filterArgs
	apply filterFunc
}

// filterArgs are a filter's arguments as written: one string, a list, or none.
type filterArgs struct {
	Items  []string
	IsList bool
}

func (a filterArgs) absent() bool { return !a.IsList && a.Items == nil }

func (a filterArgs) str() (string, bool) {
	if a.IsList || a.Items == nil {
		return "", false
	}
	return a.Items[0], true
}

func (a filterArgs) pair() (first, second string, ok bool) {
	if !a.IsList || len(a.Items) < 2 {
		return "", "", false
	}
	return a.Items[0], a.Items[1], true
}

type filterFunc func(in value, env env) (value, error)

// filterBlocks is a filters list, applied in order.
type filterBlocks []filterBlock

// Apply runs every filter over v, as CardigannBase.ApplyFilters does. A
// null passes through the filters that accept one.
func (fs filterBlocks) Apply(v value, env env) (value, error) {
	for _, f := range fs {
		out, err := f.apply(v, env)
		if err != nil {
			return nullValue, fmt.Errorf("filter %s: %w", f.Name, err)
		}
		v = out
	}
	return v, nil
}

// filterSpecs is every schema 12 filter name. build checks the arguments
// and returns the function. A name missing here is logged and skipped by
// Prowlarr, so it compiles to a no-op.
var filterSpecs = map[string]func(a filterArgs) (filterFunc, error){
	"querystring": oneArg(func(param string) filterFunc {
		return textFilter(func(s string, _ env) (string, error) { return queryParam(s, param) })
	}),
	"dateparse":     buildDateParse,
	"timeparse":     buildDateParse,
	"regexp":        buildRegexp,
	"re_replace":    buildReReplace,
	"split":         buildSplit,
	"replace":       buildReplace,
	"trim":          buildTrim,
	"prepend":       buildAffix(true),
	"append":        buildAffix(false),
	"tolower":       noArgs(textFilter(func(s string, _ env) (string, error) { return strings.ToLower(s), nil })),
	"toupper":       noArgs(textFilter(func(s string, _ env) (string, error) { return strings.ToUpper(s), nil })),
	"base64decode":  noArgs(textFilter(func(s string, _ env) (string, error) { return base64URLDecode(s) })),
	"base64encode":  noArgs(textFilter(func(s string, _ env) (string, error) { return base64.RawURLEncoding.EncodeToString([]byte(s)), nil })),
	"urldecode":     noArgs(emptyForNull(func(s string, env env) string { return env.charset.URLDecode(s) })),
	"urlencode":     noArgs(emptyForNull(func(s string, env env) string { return env.charset.URLEncode(s) })),
	"htmldecode":    noArgs(nullSafe(htmlDecode)),
	"htmlencode":    noArgs(nullSafe(htmlEncode)),
	"timeago":       noArgs(textFilter(timeAgoFilter)),
	"reltime":       noArgs(textFilter(timeAgoFilter)),
	"fuzzytime":     noArgs(textFilter(fuzzyTimeFilter)),
	"validfilename": noArgs(textFilter(func(s string, _ env) (string, error) { return validFileName(s), nil })),
	"diacritics":    buildDiacritics,
	"jsonjoinarray": buildJSONJoinArray,
	"validate": oneArg(func(list string) filterFunc {
		return textFilter(func(s string, _ env) (string, error) { return validate(s, list), nil })
	}),
	"hexdump":  noArgs(textFilter(func(s string, _ env) (string, error) { return s, nil })),
	"strdump":  noArgs(textFilter(func(s string, _ env) (string, error) { return s, nil })),
	"andmatch": noArgs(func(v value, _ env) (value, error) { return v, nil }),
}

func compileFilter(name string, args filterArgs) (filterBlock, error) {
	build, ok := filterSpecs[name]
	if !ok {
		return filterBlock{Name: name, Args: args, apply: func(v value, _ env) (value, error) { return v, nil }}, nil
	}
	f, err := build(args)
	if err != nil {
		return filterBlock{}, fmt.Errorf("filter %s: %w", name, err)
	}
	return filterBlock{Name: name, Args: args, apply: f}, nil
}

func noArgs(f filterFunc) func(filterArgs) (filterFunc, error) {
	return func(filterArgs) (filterFunc, error) { return f, nil }
}

func oneArg(build func(string) filterFunc) func(filterArgs) (filterFunc, error) {
	return func(a filterArgs) (filterFunc, error) {
		s, ok := a.str()
		if !ok {
			return nil, errors.New("takes one string argument")
		}
		return build(s), nil
	}
}

// textFilter wraps a filter that fails on a null, as most of Prowlarr's
// throw a NullReferenceException.
func textFilter(f func(string, env) (string, error)) filterFunc {
	return func(v value, env env) (value, error) {
		if v.kind != textKind {
			return nullValue, errors.New("the value is null")
		}
		out, err := f(v.text, env)
		if err != nil {
			return nullValue, err
		}
		return textValue(out), nil
	}
}

func emptyForNull(f func(string, env) string) filterFunc {
	return func(v value, env env) (value, error) {
		if v.text == "" {
			return textValue(""), nil
		}
		return textValue(f(v.text, env)), nil
	}
}

func nullSafe(f func(string) string) filterFunc {
	return func(v value, _ env) (value, error) {
		if v.kind != textKind {
			return nullValue, nil
		}
		return textValue(f(v.text)), nil
	}
}

func buildDateParse(a filterArgs) (filterFunc, error) {
	layout, ok := a.str()
	if !ok {
		return nil, errors.New("takes a date layout")
	}
	return textFilter(func(s string, env env) (string, error) {
		if t, err := parseDateLayout(s, layout, env.Clock); err == nil {
			return formatRFC1123Z(t), nil
		}
		return s, nil
	}), nil
}

func buildRegexp(a filterArgs) (filterFunc, error) {
	pattern, ok := a.str()
	if !ok {
		return nil, errors.New("takes a pattern")
	}
	re, err := compileRegex(pattern)
	if err != nil {
		return nil, err
	}
	return textFilter(func(s string, _ env) (string, error) { return re.Group1(s) }), nil
}

func buildReReplace(a filterArgs) (filterFunc, error) {
	pattern, replacement, ok := a.pair()
	if !ok {
		return nil, errors.New("takes a pattern and a replacement")
	}
	re, err := compileRegex(pattern)
	if err != nil {
		return nil, err
	}
	tmpl, err := parseTemplate(replacement)
	if err != nil {
		return nil, err
	}
	return textFilter(func(s string, env env) (string, error) {
		repl, err := tmpl.Eval(env.Scope, nil)
		if err != nil {
			return "", err
		}
		return re.Replace(s, repl)
	}), nil
}

func buildSplit(a filterArgs) (filterFunc, error) {
	sep, pos, ok := a.pair()
	if !ok || sep == "" {
		return nil, errors.New("takes a separator and a position")
	}
	idx, err := strconv.Atoi(strings.TrimSpace(pos))
	if err != nil {
		return nil, fmt.Errorf("the position %q isn't a whole number", pos)
	}
	r, _ := utf8.DecodeRuneInString(sep)
	return textFilter(func(s string, _ env) (string, error) {
		parts := strings.Split(s, string(r))
		i := idx
		if i < 0 {
			i += len(parts)
		}
		if i < 0 || i >= len(parts) {
			return "", fmt.Errorf("%q has no part %d", s, idx)
		}
		return parts[i], nil
	}), nil
}

func buildReplace(a filterArgs) (filterFunc, error) {
	from, to, ok := a.pair()
	if !ok {
		return nil, errors.New("takes the text to find and its replacement")
	}
	tmpl, err := parseTemplate(to)
	if err != nil {
		return nil, err
	}
	return textFilter(func(s string, env env) (string, error) {
		repl, err := tmpl.Eval(env.Scope, nil)
		if err != nil {
			return "", err
		}
		if from == "" {
			return "", errors.New("the text to find is empty")
		}
		return strings.ReplaceAll(s, from, repl), nil
	}), nil
}

func buildTrim(a filterArgs) (filterFunc, error) {
	if a.absent() {
		return textFilter(func(s string, _ env) (string, error) { return strings.TrimSpace(s), nil }), nil
	}
	cutset, ok := a.str()
	if !ok || cutset == "" {
		return nil, errors.New("takes one character to trim")
	}
	r, _ := utf8.DecodeRuneInString(cutset)
	return textFilter(func(s string, _ env) (string, error) { return strings.Trim(s, string(r)), nil }), nil
}

func buildAffix(prepend bool) func(filterArgs) (filterFunc, error) {
	return func(a filterArgs) (filterFunc, error) {
		text, ok := a.str()
		if !ok {
			return nil, errors.New("takes one string argument")
		}
		tmpl, err := parseTemplate(text)
		if err != nil {
			return nil, err
		}
		return func(v value, env env) (value, error) {
			add, err := tmpl.Eval(env.Scope, nil)
			if err != nil {
				return nullValue, err
			}
			if prepend {
				return textValue(add + v.text), nil
			}
			return textValue(v.text + add), nil
		}, nil
	}
}

func buildDiacritics(a filterArgs) (filterFunc, error) {
	if op, ok := a.str(); !ok || op != "replace" {
		return nil, errors.New(`takes the argument "replace"`)
	}
	return textFilter(func(s string, _ env) (string, error) { return stripDiacritics(s), nil }), nil
}

func buildJSONJoinArray(a filterArgs) (filterFunc, error) {
	path, sep, ok := a.pair()
	if !ok {
		return nil, errors.New("takes a JSON path and a separator")
	}
	p, err := compileJSONPath(path)
	if err != nil {
		return nil, err
	}
	return textFilter(func(s string, _ env) (string, error) { return jsonJoinArray(s, p, sep) }), nil
}

func timeAgoFilter(s string, env env) (string, error) {
	t, err := fromTimeAgo(s, env.Clock)
	if err != nil {
		return "", err
	}
	return formatRFC1123Z(t), nil
}

func fuzzyTimeFilter(s string, env env) (string, error) {
	t, err := parseDate(s, env.Clock)
	if err != nil {
		return "", err
	}
	return formatRFC1123Z(t), nil
}

// queryParam is ParseUtil.GetArgumentFromQueryString: the first value of
// param, matched ignoring case, in the query after the first '?'. A key
// without '=' has an empty value, as QueryHelpers.ParseQuery reads it.
func queryParam(url, param string) (string, error) {
	_, qs, ok := strings.Cut(url, "?")
	if !ok {
		return "", fmt.Errorf("%q has no query string", url)
	}
	qs, _, _ = strings.Cut(qs, "#")
	for _, part := range strings.Split(qs, "&") {
		key, value, _ := strings.Cut(part, "=")
		if strings.EqualFold(unescapeQuery(strings.TrimLeftFunc(key, unicode.IsSpace)), param) {
			return unescapeQuery(value), nil
		}
	}
	return "", fmt.Errorf("%q has no %s parameter", url, param)
}

// unescapeQuery is Uri.UnescapeDataString after '+' becomes a space. A
// sequence that isn't UTF-8 stays escaped.
func unescapeQuery(s string) string {
	s = strings.ReplaceAll(s, "+", " ")
	if !strings.Contains(s, "%") {
		return s
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			out = append(out, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		out = append(out, s[i])
	}
	if !utf8.Valid(out) {
		return s
	}
	return string(out)
}

// base64URLDecode is WebEncoders.Base64UrlDecode, read as UTF-8.
func base64URLDecode(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	switch len(s) % 4 {
	case 1:
		return "", fmt.Errorf("%q isn't base64", s)
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	s = strings.NewReplacer("-", "+", "_", "/", " ", "", "\t", "", "\r", "", "\n", "").Replace(s)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("%q isn't base64", s)
	}
	return strings.ToValidUTF8(string(b), string(utf8.RuneError)), nil
}

// htmlDecode is WebUtility.HtmlDecode: named entities and &#N; and &#xN;
// references that end in ';'. Anything else stays as written.
func htmlDecode(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '&' {
			b.WriteByte(s[i])
			continue
		}
		end := strings.IndexAny(s[i+1:], ";&")
		if end <= 0 || s[i+1+end] != ';' {
			b.WriteByte('&')
			continue
		}
		entity := s[i+1 : i+1+end]
		if r, ok := decodeEntity(entity); ok {
			b.WriteString(r)
			i += end + 1
			continue
		}
		if len(entity) > 1 && entity[0] == '#' {
			b.WriteByte('&')
			continue
		}
		b.WriteString("&" + entity + ";")
		i += end + 1
	}
	return b.String()
}

func decodeEntity(entity string) (string, bool) {
	if len(entity) > 1 && entity[0] == '#' {
		var n uint64
		var err error
		if entity[1] == 'x' || entity[1] == 'X' {
			n, err = strconv.ParseUint(entity[2:], 16, 32)
		} else {
			n, err = strconv.ParseUint(strings.TrimSpace(entity[1:]), 10, 32)
		}
		if err != nil || (n >= 0xD800 && n <= 0xDFFF) || n > unicode.MaxRune {
			return "", false
		}
		return string(rune(n)), true
	}
	out := html.UnescapeString("&" + entity + ";")
	if strings.Contains(out, ";") {
		return "", false
	}
	return out, true
}

// htmlEncode is WebUtility.HtmlEncode.
func htmlEncode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\'':
			b.WriteString("&#39;")
		case r == '&':
			b.WriteString("&amp;")
		case (r >= 160 && r < 256) || r > 0xFFFF:
			b.WriteString("&#" + strconv.Itoa(int(r)) + ";")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// validFileName is MakeValidFileName(s, '_', false) with Windows' invalid
// file name characters on every OS, so every Node gives the same result.
func validFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 32 || strings.ContainsRune(`"<>|:*?\/`, r) {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

func stripDiacritics(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return norm.NFC.String(b.String())
}

// validate keeps the list's items that the value holds, in the list's
// order, both split on the same delimiters and compared in lowercase.
func validate(s, list string) string {
	split := func(v string) []string {
		var out []string
		for _, f := range strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return strings.ContainsRune(`, /)(.;[]"|:`, r) }) {
			if f = strings.TrimSpace(f); f != "" {
				out = append(out, f)
			}
		}
		return out
	}
	have := split(s)
	var keep []string
	for _, item := range split(list) {
		if containsString(have, item) && !containsString(keep, item) {
			keep = append(keep, item)
		}
	}
	return strings.Join(keep, ", ")
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
