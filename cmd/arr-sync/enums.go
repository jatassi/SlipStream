package main

import (
	"encoding/json"
	"strconv"
)

type enumValue struct {
	name  string
	value int64
}

// readEnum reads the members of "enum name { A, B = 4, ... }" with the
// values C# gives them.
func readEnum(s *source, name string) ([]enumValue, error) {
	c := &cursor{source: s}
	for c.i < len(c.toks) && !c.is(0, "enum", name, "{") {
		c.i++
	}
	if c.i >= len(c.toks) {
		return nil, s.errorf(0, "no enum %s", name)
	}
	c.i += 3
	var values []enumValue
	next := int64(0)
	err := c.list("}", func() error {
		for c.is(0, "[") {
			c.skipBalanced()
		}
		v := enumValue{name: c.peek(0).text, value: next}
		c.i++
		if c.is(0, "=") {
			c.i++
			n, err := c.value()
			if err != nil {
				return err
			}
			num, ok := n.(json.Number)
			if !ok {
				return c.errorf(c.peek(0).line, "enum %s.%s has a value cmd/arr-sync can't read", name, v.name)
			}
			if v.value, err = strconv.ParseInt(string(num), 10, 64); err != nil {
				return c.errorf(c.peek(0).line, "enum %s.%s: %v", name, v.name, err)
			}
		}
		next = v.value + 1
		values = append(values, v)
		return nil
	})
	return values, err
}

// readLanguages reads "static Language Name => new Language(id, "Label")".
func readLanguages(s *source) ([]enumValue, error) {
	c := &cursor{source: s}
	var values []enumValue
	for ; c.i < len(c.toks); c.i++ {
		if !c.is(0, "static", "Language") || c.peek(2).kind != tIdent || !c.is(3, "=>", "new", "Language", "(") {
			continue
		}
		name, line := c.peek(2).text, c.peek(7).line
		c.i += 7
		n, err := c.value()
		if err != nil {
			return nil, err
		}
		num, ok := n.(json.Number)
		if !ok {
			return nil, c.errorf(line, "language %s has an id cmd/arr-sync can't read", name)
		}
		v, err := strconv.ParseInt(string(num), 10, 64)
		if err != nil {
			return nil, c.errorf(line, "language %s: %v", name, err)
		}
		values = append(values, enumValue{name, v})
	}
	return values, nil
}

type isoLanguage struct {
	args     []string
	language string
}

// readIsoLanguages reads each new IsoLanguage("en", "", "eng", ...,
// Language.English): its string arguments in upstream's order, and its Language.
func readIsoLanguages(s *source) ([]isoLanguage, error) {
	c := &cursor{source: s}
	var out []isoLanguage
	for ; c.i < len(c.toks); c.i++ {
		if !c.is(0, "new", "IsoLanguage", "(") {
			continue
		}
		v, err := c.value()
		if err != nil {
			return nil, err
		}
		var iso isoLanguage
		made, _ := v.(map[string]any)
		args, _ := made["args"].([]any)
		for _, a := range args {
			switch a := a.(type) {
			case string:
				iso.args = append(iso.args, a)
			case map[string]any:
				iso.language, _ = a["ref"].(string)
			}
		}
		if iso.language == "" {
			return nil, c.errorf(c.peek(0).line, "an IsoLanguage without a Language")
		}
		out = append(out, iso)
		c.i--
	}
	return out, nil
}
