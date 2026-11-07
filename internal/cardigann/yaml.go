package cardigann

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// This file reads a definition as YamlDotNet does for Prowlarr: keys match
// the camelCase property names exactly, unknown keys are ignored, every
// scalar keeps its text as written, and booleans and numbers are parsed
// only where the model holds one.

func decodeDefinition(src []byte) (*cardigannDefinition, error) {
	file, err := parser.ParseBytes(prepareYAML(src), 0, parser.AllowDuplicateMapKey())
	if err != nil {
		return nil, &yamlError{err: err}
	}
	if len(file.Docs) == 0 || file.Docs[0].Body == nil {
		return nil, &yamlError{err: errors.New("the file is empty")}
	}
	def := &cardigannDefinition{}
	def.setDefaults()
	if err := decodeNode(file.Docs[0].Body, reflect.ValueOf(def).Elem(), ""); err != nil {
		return nil, err
	}
	return def, nil
}

type yamlError struct{ err error }

func (e *yamlError) Error() string { return "read the YAML: " + e.err.Error() }
func (e *yamlError) Unwrap() error { return e.err }

// fieldError is a value the model can't hold, such as a list where a
// string belongs.
type fieldError struct {
	path string
	err  error
}

func (e *fieldError) Error() string { return e.path + ": " + e.err.Error() }
func (e *fieldError) Unwrap() error { return e.err }

func failAt(path string, err error) error {
	var fe *fieldError
	if errors.As(err, &fe) {
		return err
	}
	return &fieldError{path: path, err: err}
}

// prepareYAML strips a byte order mark and escapes a literal tab inside a
// double-quoted scalar, which goccy refuses inside a flow mapping.
func prepareYAML(src []byte) []byte {
	src = bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))
	if !bytes.Contains(src, []byte("\t")) {
		return src
	}
	lines := bytes.Split(src, []byte("\n"))
	for i, line := range lines {
		lines[i] = escapeQuotedTabs(line)
	}
	return bytes.Join(lines, []byte("\n"))
}

func escapeQuotedTabs(line []byte) []byte {
	var out []byte
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote == 0 && c == '#':
			return append(out, line[i:]...)
		case quote == 0 && (c == '"' || c == '\''):
			quote = c
		case quote == '"' && c == '\\' && i+1 < len(line):
			out = append(out, c)
			i++
			c = line[i]
		case c == quote:
			quote = 0
		case quote == '"' && c == '\t':
			out = append(out, '\\', 't')
			continue
		}
		out = append(out, c)
	}
	return out
}

type nodeDecoder interface {
	decodeNode(n ast.Node, path string) error
}

type defaulter interface {
	setDefaults()
}

var templateType = reflect.TypeFor[*template]()

func decodeNode(n ast.Node, v reflect.Value, path string) error {
	n = unwrap(n)
	if isNull(n) {
		return nil
	}
	if v.CanAddr() {
		if d, ok := v.Addr().Interface().(nodeDecoder); ok {
			return d.decodeNode(n, path)
		}
	}
	if v.Type() == templateType {
		return decodeTemplate(n, v, path)
	}
	switch v.Kind() { //nolint:exhaustive // every other kind is a scalar
	case reflect.Pointer:
		elem := reflect.New(v.Type().Elem())
		if d, ok := elem.Interface().(defaulter); ok {
			d.setDefaults()
		}
		if err := decodeNode(n, elem.Elem(), path); err != nil {
			return err
		}
		v.Set(elem)
		return nil
	case reflect.Struct:
		return decodeStruct(n, v, path)
	case reflect.Slice:
		return decodeSlice(n, v, path)
	default:
		return decodeScalar(n, v, path)
	}
}

func unwrap(n ast.Node) ast.Node {
	for {
		switch t := n.(type) {
		case *ast.TagNode:
			n = t.Value
		case *ast.AnchorNode:
			n = t.Value
		case *ast.DocumentNode:
			n = t.Body
		default:
			return n
		}
	}
}

func isNull(n ast.Node) bool {
	if n == nil {
		return true
	}
	_, ok := n.(*ast.NullNode)
	return ok
}

// scalarText is a scalar's text as written: a string's value, or a
// number's or boolean's token.
func scalarText(n ast.Node) (string, bool) {
	switch t := unwrap(n).(type) {
	case *ast.StringNode:
		return t.Value, true
	case *ast.LiteralNode:
		return t.Value.Value, true
	case *ast.IntegerNode, *ast.FloatNode, *ast.BoolNode, *ast.InfinityNode, *ast.NanNode:
		return t.GetToken().Value, true
	}
	return "", false
}

func decodeTemplate(n ast.Node, v reflect.Value, path string) error {
	text, ok := scalarText(n)
	if !ok {
		return failAt(path, errors.New("expected a string"))
	}
	t, err := parseTemplate(text)
	if err != nil {
		return failAt(path, err)
	}
	v.Set(reflect.ValueOf(t))
	return nil
}

func decodeScalar(n ast.Node, v reflect.Value, path string) error {
	text, ok := scalarText(n)
	if !ok {
		return failAt(path, fmt.Errorf("expected a %s", v.Kind()))
	}
	switch v.Kind() { //nolint:exhaustive // the model holds no other scalar kind
	case reflect.String:
		v.SetString(text)
	case reflect.Bool:
		b, err := parseYAMLBool(text)
		if err != nil {
			return failAt(path, err)
		}
		v.SetBool(b)
	case reflect.Int:
		i, err := strconv.ParseInt(strings.ReplaceAll(text, "_", ""), 0, 32)
		if err != nil {
			return failAt(path, fmt.Errorf("%q isn't a whole number", text))
		}
		v.SetInt(i)
	case reflect.Float64:
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return failAt(path, fmt.Errorf("%q isn't a number", text))
		}
		v.SetFloat(f)
	default:
		return failAt(path, fmt.Errorf("can't hold a %s", v.Kind()))
	}
	return nil
}

// parseYAMLBool reads a boolean as YamlDotNet does.
func parseYAMLBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "y", "yes", "on":
		return true, nil
	case "false", "n", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("%q isn't true or false", s)
}

func decodeSlice(n ast.Node, v reflect.Value, path string) error {
	seq, ok := n.(*ast.SequenceNode)
	if !ok {
		return failAt(path, errors.New("expected a list"))
	}
	out := reflect.MakeSlice(v.Type(), len(seq.Values), len(seq.Values))
	for i, item := range seq.Values {
		elem := out.Index(i)
		if d, ok := elem.Addr().Interface().(defaulter); ok {
			d.setDefaults()
		}
		if err := decodeNode(item, elem, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	v.Set(out)
	return nil
}

// mappingPairs is a mapping node's entries, whether goccy read it as a
// mapping or as a single key and value.
func mappingPairs(n ast.Node, path string) ([]*ast.MappingValueNode, error) {
	switch t := n.(type) {
	case *ast.MappingNode:
		return t.Values, nil
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{t}, nil
	}
	return nil, failAt(path, errors.New("expected a mapping"))
}

func keyText(kv *ast.MappingValueNode, path string) (string, error) {
	key, ok := scalarText(kv.Key)
	if !ok {
		return "", failAt(path, errors.New("a key isn't a scalar"))
	}
	return key, nil
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func decodeStruct(n ast.Node, v reflect.Value, path string) error {
	pairs, err := mappingPairs(n, path)
	if err != nil {
		return err
	}
	fields := structFields(v.Type())
	for _, kv := range pairs {
		key, err := keyText(kv, path)
		if err != nil {
			return err
		}
		index, ok := fields[key]
		if !ok {
			continue
		}
		if err := decodeNode(kv.Value, v.FieldByIndex(index), joinPath(path, key)); err != nil {
			return err
		}
	}
	return nil
}

var fieldCache sync.Map

// structFields maps each yaml key of t, inline structs included, to its
// field.
func structFields(t reflect.Type) map[string][]int {
	if f, ok := fieldCache.Load(t); ok {
		if m, ok := f.(map[string][]int); ok {
			return m
		}
	}
	out := map[string][]int{}
	for _, f := range reflect.VisibleFields(t) {
		tag := f.Tag.Get("yaml")
		name, opts, _ := strings.Cut(tag, ",")
		switch {
		case tag == "-" || !f.IsExported():
		case opts == "inline":
		case name != "":
			out[name] = f.Index
		}
	}
	fieldCache.Store(t, out)
	return out
}

func (m *keyedList[T]) decodeNode(n ast.Node, path string) error {
	pairs, err := mappingPairs(n, path)
	if err != nil {
		return err
	}
	for _, kv := range pairs {
		key, err := keyText(kv, path)
		if err != nil {
			return err
		}
		var value T
		if err := decodeNode(kv.Value, reflect.ValueOf(&value).Elem(), joinPath(path, key)); err != nil {
			return err
		}
		if i := m.index(key); i >= 0 {
			(*m)[i].Value = value
			continue
		}
		*m = append(*m, entry[T]{Key: key, Value: value})
	}
	return nil
}

func (m keyedList[T]) index(key string) int {
	for i, e := range m {
		if e.Key == key {
			return i
		}
	}
	return -1
}

func (f *fieldList) decodeNode(n ast.Node, path string) error {
	pairs, err := mappingPairs(n, path)
	if err != nil {
		return err
	}
	for _, kv := range pairs {
		key, err := keyText(kv, path)
		if err != nil {
			return err
		}
		var block selectorBlock
		if err := decodeNode(kv.Value, reflect.ValueOf(&block).Elem(), joinPath(path, key)); err != nil {
			return err
		}
		*f = append(*f, entry[selectorBlock]{Key: key, Value: block})
	}
	return nil
}

func (f *filterBlock) decodeNode(n ast.Node, path string) error {
	pairs, err := mappingPairs(n, path)
	if err != nil {
		return err
	}
	var name string
	var args filterArgs
	for _, kv := range pairs {
		key, err := keyText(kv, path)
		if err != nil {
			return err
		}
		switch key {
		case "name":
			name, _ = scalarText(kv.Value)
		case "args":
			if args, err = decodeArgs(kv.Value, joinPath(path, "args")); err != nil {
				return err
			}
		}
	}
	compiled, err := compileFilter(name, args)
	if err != nil {
		return failAt(path, err)
	}
	*f = compiled
	return nil
}

func decodeArgs(n ast.Node, path string) (filterArgs, error) {
	n = unwrap(n)
	if isNull(n) {
		return filterArgs{}, nil
	}
	if s, ok := scalarText(n); ok {
		return filterArgs{Items: []string{s}}, nil
	}
	seq, ok := n.(*ast.SequenceNode)
	if !ok {
		return filterArgs{}, failAt(path, errors.New("expected a string or a list"))
	}
	args := filterArgs{IsList: true, Items: make([]string, len(seq.Values))}
	for i, item := range seq.Values {
		s, ok := scalarText(item)
		if !ok && !isNull(unwrap(item)) {
			return filterArgs{}, failAt(fmt.Sprintf("%s[%d]", path, i), errors.New("expected a string"))
		}
		args.Items[i] = s
	}
	return args, nil
}
