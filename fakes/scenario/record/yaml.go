package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// rewrite sets the Scenario file's tmdb.movies, tmdb.series, tvdb.series
// and omdb.titles to the recorded ones, and drops each that's empty. Only
// the tmdb, tvdb and omdb blocks are written again, so every other line of
// the file stays as it was.
func rewrite(path string, rec recording) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read the Scenario: %w", err)
	}
	for _, b := range []struct {
		key, sub string
		value    any
	}{{"tmdb", "movies", rec.movies}, {"tmdb", "series", rec.tmdbSeries}, {"tvdb", "series", rec.tvdbSeries}, {"omdb", "titles", rec.titles}} {
		value := &yaml.Node{}
		if err := value.Encode(b.value); err != nil {
			return fmt.Errorf("encode %s.%s: %w", b.key, b.sub, err)
		}
		if len(value.Content) == 0 {
			value = nil
		} else {
			tidy(value, flowKeys[b.key])
		}
		if data, err = replace(data, b.key, b.sub, value); err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write the Scenario: %w", err)
	}
	return nil
}

// replace sets data's top-level key's sub to value, and drops sub when
// value is nil and key when it holds nothing else. It writes again only the
// lines of key's block, from key's line to the next line that starts at
// the first column, and adds the block at the end when data has none.
func replace(data []byte, key, sub string, value *yaml.Node) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("read the YAML: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("the file isn't a YAML mapping")
	}
	lines := strings.SplitAfter(string(data), "\n")
	if last := len(lines) - 1; lines[last] == "" {
		lines = lines[:last]
	} else {
		lines[last] += "\n"
	}
	start, end := len(lines), len(lines)
	block := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != key {
			continue
		}
		start, block = root.Content[i].Line-1, root.Content[i+1]
		end = start + 1
		for end < len(lines) && !topLevel(lines[end]) {
			end++
		}
		for end > start+1 && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
	}
	if block.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s isn't a mapping", key)
	}
	block.Style = 0
	if value != nil {
		*child(block, sub) = *value
	} else if i := index(block, sub); i >= 0 {
		block.Content = slices.Delete(block.Content, i, i+2)
	}
	if len(block.Content) == 0 {
		return []byte(strings.Join(lines[:start], "") + strings.Join(lines[end:], "")), nil
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	wrapper := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, block}}
	if err := enc.Encode(wrapper); err != nil {
		return nil, fmt.Errorf("encode %s: %w", key, err)
	}
	return []byte(strings.Join(lines[:start], "") + out.String() + strings.Join(lines[end:], "")), nil
}

func topLevel(line string) bool {
	return line != "" && !strings.ContainsRune(" \t\r\n", rune(line[0]))
}

// child is mapping's value at key, a new value added at its end when it
// has none.
func child(mapping *yaml.Node, key string) *yaml.Node {
	if i := index(mapping, key); i >= 0 {
		return mapping.Content[i+1]
	}
	value := &yaml.Node{}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
	return value
}

// index is where key sits in mapping's content, -1 when it has none.
func index(mapping *yaml.Node, key string) int {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// flowKeys are, in each block, the keys whose values read best as the
// catalog writes them: a list or map of plain values on one line, and
// each item of any other list or map on a line of its own.
var flowKeys = map[string]map[string]bool{
	"tmdb": {
		"genres": true, "dates": true, "titles": true, "cast": true, "crew": true, "images": true, "collection": true, "studio": true,
		"creators": true, "certifications": true, "network": true,
	},
	"tvdb": {"genres": true, "aliases": true, "dvd": true},
	"omdb": {"episodes": true},
}

// tidy writes each runtime that's a duration in minutes and the values of
// keys as flowKeys sets out.
func tidy(n *yaml.Node, keys map[string]bool) {
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			tidy(item, keys)
		}
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := n.Content[i].Value, n.Content[i+1]
			switch {
			case key == "runtime":
				if d, err := time.ParseDuration(value.Value); err == nil {
					value.Value = strconv.Itoa(int(d/time.Minute)) + "m"
				}
			case keys[key]:
				flow(value)
			default:
				tidy(value, keys)
			}
		}
	}
}

// flow writes n on one line when its items are plain values, else each of
// its items on a line of its own.
func flow(n *yaml.Node) {
	items := n.Content
	if n.Kind == yaml.MappingNode {
		items = nil
		for i := 1; i < len(n.Content); i += 2 {
			items = append(items, n.Content[i])
		}
	}
	if !slices.ContainsFunc(items, func(item *yaml.Node) bool { return item.Kind != yaml.ScalarNode }) {
		n.Style = yaml.FlowStyle
		return
	}
	for _, item := range items {
		item.Style = yaml.FlowStyle
	}
}
