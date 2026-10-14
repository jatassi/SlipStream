// Package shape is what a conformance suite compares between a fake and the
// real service: the key names and JSON types of each answer, never values.
package shape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Shape is shape.json. For each answer a suite records, keyed "<route>
// <status>", it maps every key path in the answer's bodies, such as
// "$.results[].title", to the JSON types seen there, such as "null|string".
type Shape map[string]map[string]string

// Body is one answer a Recorder kept.
type Body struct {
	Key  string
	JSON []byte
}

// Of is the shape of bodies: every key path any of them carries, with
// every type it had.
func Of(bodies []Body) (Shape, error) {
	seen := map[string]map[string]map[string]bool{}
	for _, b := range bodies {
		paths, err := flatten(b)
		if err != nil {
			return nil, err
		}
		if seen[b.Key] == nil {
			seen[b.Key] = map[string]map[string]bool{}
		}
		for p, types := range paths {
			if seen[b.Key][p] == nil {
				seen[b.Key][p] = map[string]bool{}
			}
			maps.Copy(seen[b.Key][p], types)
		}
	}
	s := Shape{}
	for key, paths := range seen {
		s[key] = map[string]string{}
		for p, types := range paths {
			s[key][p] = strings.Join(slices.Sorted(maps.Keys(types)), "|")
		}
	}
	return s, nil
}

// Check lists, sorted, every key path and type in bodies that s lacks.
func (s Shape) Check(bodies []Body) ([]string, error) {
	problems := map[string]bool{}
	for _, b := range bodies {
		paths, err := flatten(b)
		if err != nil {
			return nil, err
		}
		want, ok := s[b.Key]
		if !ok {
			problems[b.Key+": not in shape.json"] = true
			continue
		}
		for p, types := range paths {
			allowed, ok := want[p]
			for typ := range types {
				switch {
				case !ok:
					problems[fmt.Sprintf("%s: %s (%s) is not in shape.json", b.Key, p, typ)] = true
				case !slices.Contains(strings.Split(allowed, "|"), typ):
					problems[fmt.Sprintf("%s: %s is %s, shape.json has %s", b.Key, p, typ, allowed)] = true
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(problems)), nil
}

// flatten maps each key path in b to the types it holds there. An array's
// elements share one path.
func flatten(b Body) (map[string]map[string]bool, error) {
	var v any
	if err := json.Unmarshal(b.JSON, &v); err != nil {
		return nil, fmt.Errorf("%s: %w", b.Key, err)
	}
	paths := map[string]map[string]bool{}
	walk(paths, "$", v)
	return paths, nil
}

func walk(paths map[string]map[string]bool, p string, v any) {
	if paths[p] == nil {
		paths[p] = map[string]bool{}
	}
	switch v := v.(type) {
	case map[string]any:
		paths[p]["object"] = true
		for k, child := range v {
			walk(paths, p+"."+k, child)
		}
	case []any:
		paths[p]["array"] = true
		for _, child := range v {
			walk(paths, p+"[]", child)
		}
	case string:
		paths[p]["string"] = true
	case float64:
		paths[p]["number"] = true
	case bool:
		paths[p]["boolean"] = true
	case nil:
		paths[p]["null"] = true
	}
}

// Recorder is an http.RoundTripper that keeps every answer's body, keyed by
// the route its request path names and the status.
type Recorder struct {
	next  http.RoundTripper
	route func(path string) string

	mu     sync.Mutex
	bodies []Body
}

func NewRecorder(next http.RoundTripper, route func(path string) string) *Recorder {
	if next == nil {
		next = http.DefaultTransport
	}
	return &Recorder{next: next, route: route}
}

func (r *Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, err //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", req.URL.Path, err)
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bodies = append(r.bodies, Body{Key: r.route(req.URL.Path) + " " + strconv.Itoa(res.StatusCode), JSON: body})
	return res, nil
}

func (r *Recorder) Bodies() []Body {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.bodies)
}

// Verify hands every body to record when it is set, for a run that
// rewrites shape.json, and otherwise fails t for each key path or type the
// bodies carry that want lacks.
func Verify(t *testing.T, bodies []Body, want []byte, record func(key string, body []byte)) {
	t.Helper()
	if record != nil {
		for _, b := range bodies {
			record(b.Key, b.JSON)
		}
		return
	}
	var s Shape
	if err := json.Unmarshal(want, &s); err != nil {
		t.Fatalf("shape.json: %v", err)
	}
	problems, err := s.Check(bodies)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// Write rewrites the shape.json at path from bodies.
func Write(path string, bodies []Body) error {
	s, err := Of(bodies)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
