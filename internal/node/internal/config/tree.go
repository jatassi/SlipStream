// Package config loads a Node's typed config tree from its layers and walks
// the tree's leaves, the one place that knows the closed set of leaf types.
package config

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Leaf is one key of a config tree, addressable so a layer can set it.
type Leaf struct {
	Path  string
	value reflect.Value
	codec codec
}

func (l Leaf) Env() string {
	return "SLIPSTREAM_" + strings.ToUpper(strings.ReplaceAll(l.Path, ".", "_"))
}

func (l Leaf) String() string {
	elems := l.elems()
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = l.codec.format(e)
	}
	return strings.Join(out, ",")
}

// setText parses the environment form, where a list is comma-separated.
func (l Leaf) setText(s string) error {
	if l.value.Kind() != reflect.Slice {
		return l.codec.parse(s, l.value)
	}
	if s == "" {
		return l.setList(nil)
	}
	return l.setList(strings.Split(s, ","))
}

func (l Leaf) isList() bool { return l.value.Kind() == reflect.Slice }

func (l Leaf) setList(items []string) error {
	list := reflect.MakeSlice(l.value.Type(), len(items), len(items))
	for i, s := range items {
		if err := l.codec.parse(s, list.Index(i)); err != nil {
			return fmt.Errorf("item %d: %w", i+1, err)
		}
	}
	l.value.Set(list)
	return nil
}

// Hosts are the hosts of an endpoint leaf, one per item, and nil for any other leaf.
func (l Leaf) Hosts() []string {
	var hosts []string
	for _, e := range l.elems() {
		if h, ok := e.Interface().(hostnamer); ok {
			hosts = append(hosts, h.Hostname())
		}
	}
	return hosts
}

func (l Leaf) elems() []reflect.Value {
	if !l.isList() {
		return []reflect.Value{l.value}
	}
	out := make([]reflect.Value, l.value.Len())
	for i := range out {
		out[i] = l.value.Index(i)
	}
	return out
}

// Walk returns the leaves of tree, a pointer to a struct, in field order. It
// panics on a field without a yaml tag or of a type outside the leaf set:
// strings, booleans, integers, time.Duration, encoding.TextUnmarshaler
// values and slices of those. A text leaf with a Redacted method shows only
// that wherever a value is printed, its parse errors included.
func Walk(tree any) []Leaf {
	v := reflect.ValueOf(tree)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("config: tree is %T, not a pointer to a struct", tree))
	}
	var leaves []Leaf
	walk(v.Elem(), "", &leaves)
	return leaves
}

func walk(v reflect.Value, prefix string, leaves *[]Leaf) {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			panic(fmt.Sprintf("config: field %s.%s has no yaml name", t, f.Name))
		}
		path := prefix + name
		fv := v.Field(i)
		if c, ok := scalarCodec(f.Type); ok {
			*leaves = append(*leaves, Leaf{Path: path, value: fv, codec: c})
			continue
		}
		if f.Type.Kind() == reflect.Slice {
			if c, ok := scalarCodec(f.Type.Elem()); ok {
				*leaves = append(*leaves, Leaf{Path: path, value: fv, codec: c})
				continue
			}
		}
		if f.Type.Kind() == reflect.Struct {
			walk(fv, path+".", leaves)
			continue
		}
		panic(fmt.Sprintf("config: %s has type %s, outside the leaf set", path, f.Type))
	}
}

type codec struct {
	parse  func(string, reflect.Value) error
	format func(reflect.Value) string
}

// redacter is a leaf whose value is a secret.
type redacter interface {
	Redacted() string
}

var (
	durationType        = reflect.TypeFor[time.Duration]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	redacterType        = reflect.TypeFor[redacter]()
)

func scalarCodec(t reflect.Type) (codec, bool) {
	switch {
	case reflect.PointerTo(t).Implements(textUnmarshalerType) && t.Implements(textMarshalerType) && t.Implements(redacterType):
		return redactedCodec, true
	case reflect.PointerTo(t).Implements(textUnmarshalerType) && t.Implements(textMarshalerType):
		return textCodec, true
	case t == durationType:
		return durationCodec, true
	}
	switch t.Kind() { //nolint:exhaustive // every other kind is outside the leaf set
	case reflect.String:
		return stringCodec, true
	case reflect.Bool:
		return boolCodec, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intCodec, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintCodec, true
	}
	return codec{}, false
}

var (
	textCodec = codec{
		parse: func(s string, v reflect.Value) error {
			u, _ := v.Addr().Interface().(encoding.TextUnmarshaler)
			if err := u.UnmarshalText([]byte(s)); err != nil {
				return fmt.Errorf("%q: %w", s, err)
			}
			return nil
		},
		format: func(v reflect.Value) string {
			m, _ := v.Interface().(encoding.TextMarshaler)
			b, err := m.MarshalText()
			if err != nil {
				panic(fmt.Sprintf("config: format %s: %v", v.Type(), err))
			}
			return string(b)
		},
	}
	redactedCodec = codec{
		parse: func(s string, v reflect.Value) error {
			u, _ := v.Addr().Interface().(encoding.TextUnmarshaler)
			return u.UnmarshalText([]byte(s))
		},
		format: func(v reflect.Value) string {
			r, _ := v.Interface().(redacter)
			return r.Redacted()
		},
	}
	durationCodec = codec{
		parse: func(s string, v reflect.Value) error {
			d, err := time.ParseDuration(s)
			if err != nil {
				return fmt.Errorf("not a duration: %w", err)
			}
			v.SetInt(int64(d))
			return nil
		},
		format: func(v reflect.Value) string { return time.Duration(v.Int()).String() },
	}
	stringCodec = codec{
		parse:  func(s string, v reflect.Value) error { v.SetString(s); return nil },
		format: func(v reflect.Value) string { return v.String() },
	}
	boolCodec = codec{
		parse: func(s string, v reflect.Value) error {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return fmt.Errorf("not a boolean: %w", err)
			}
			v.SetBool(b)
			return nil
		},
		format: func(v reflect.Value) string { return strconv.FormatBool(v.Bool()) },
	}
	intCodec = codec{
		parse: func(s string, v reflect.Value) error {
			n, err := strconv.ParseInt(s, 10, v.Type().Bits())
			if err != nil {
				return fmt.Errorf("not an integer: %w", err)
			}
			v.SetInt(n)
			return nil
		},
		format: func(v reflect.Value) string { return strconv.FormatInt(v.Int(), 10) },
	}
	uintCodec = codec{
		parse: func(s string, v reflect.Value) error {
			n, err := strconv.ParseUint(s, 10, v.Type().Bits())
			if err != nil {
				return fmt.Errorf("not an unsigned integer: %w", err)
			}
			v.SetUint(n)
			return nil
		},
		format: func(v reflect.Value) string { return strconv.FormatUint(v.Uint(), 10) },
	}
)
