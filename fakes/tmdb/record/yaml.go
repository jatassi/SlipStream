package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// rewrite sets the Scenario file's tmdb.movies and omdb.titles to the
// recorded ones. Only the tmdb and omdb blocks are written again, so every
// other line of the file stays as it was.
func rewrite(path string, movies []scenario.TMDBMovie, titles []scenario.OMDbTitle) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read the Scenario: %w", err)
	}
	var m, t yaml.Node
	if err := m.Encode(movies); err != nil {
		return fmt.Errorf("encode the movies: %w", err)
	}
	if err := t.Encode(titles); err != nil {
		return fmt.Errorf("encode the titles: %w", err)
	}
	tidy(&m)
	if data, err = replace(data, "tmdb", "movies", &m); err != nil {
		return fmt.Errorf("rewrite %s: %w", path, err)
	}
	if data, err = replace(data, "omdb", "titles", &t); err != nil {
		return fmt.Errorf("rewrite %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write the Scenario: %w", err)
	}
	return nil
}

// replace sets data's top-level key's sub to value. It writes again only
// the lines of key's block, from key's line to the next line that starts
// at the first column, and adds the block at the end when data has none.
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
	*child(block, sub) = *value
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
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	value := &yaml.Node{}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
	return value
}

// flowKeys are the keys whose items read best on one line each, as the
// catalog writes them.
var flowKeys = map[string]bool{"genres": true, "dates": true, "titles": true, "cast": true, "crew": true, "images": true, "collection": true, "studio": true}

// tidy writes each movie's runtime in minutes and its short lists one item
// a line.
func tidy(movies *yaml.Node) {
	for _, m := range movies.Content {
		tidyMapping(m)
	}
}

func tidyMapping(m *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, value := m.Content[i].Value, m.Content[i+1]
		switch {
		case key == "runtime":
			if d, err := time.ParseDuration(value.Value); err == nil {
				value.Value = strconv.Itoa(int(d/time.Minute)) + "m"
			}
		case key == "credits":
			tidyMapping(value)
		case flowKeys[key] && (value.Kind == yaml.MappingNode || value.Kind == yaml.SequenceNode && len(value.Content) > 0 && value.Content[0].Kind == yaml.ScalarNode):
			value.Style = yaml.FlowStyle
		case flowKeys[key]:
			for _, item := range value.Content {
				item.Style = yaml.FlowStyle
			}
		}
	}
}
