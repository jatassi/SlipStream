package cardigann

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONSelector provides dot-notation path extraction from JSON data.
type JSONSelector struct {
	data interface{}
}

// NewJSONSelector creates a new JSON selector from raw JSON bytes.
func NewJSONSelector(jsonBytes []byte) (*JSONSelector, error) {
	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &JSONSelector{data: data}, nil
}

// NewJSONSelectorFromData creates a new JSON selector from parsed data.
func NewJSONSelectorFromData(data interface{}) *JSONSelector {
	return &JSONSelector{data: data}
}

// Select extracts a value using dot notation path.
// Supports: object.field, array[0], object.nested.field
func (s *JSONSelector) Select(path string) (interface{}, error) {
	if path == "" || path == "." {
		return s.data, nil
	}

	return selectPath(s.data, path)
}

// SelectString extracts a value and converts it to string.
func (s *JSONSelector) SelectString(path string) (string, error) {
	val, err := s.Select(path)
	if err != nil {
		return "", err
	}
	return toString(val), nil
}

// SelectInt extracts a value and converts it to int.
func (s *JSONSelector) SelectInt(path string) (int, error) {
	val, err := s.Select(path)
	if err != nil {
		return 0, err
	}
	return toInt(val), nil
}

// SelectFloat extracts a value and converts it to float64.
func (s *JSONSelector) SelectFloat(path string) (float64, error) {
	val, err := s.Select(path)
	if err != nil {
		return 0, err
	}
	return toFloat(val), nil
}

// SelectBool extracts a value and converts it to bool.
func (s *JSONSelector) SelectBool(path string) (bool, error) {
	val, err := s.Select(path)
	if err != nil {
		return false, err
	}
	return toBool(val), nil
}

// SelectArray extracts an array value.
func (s *JSONSelector) SelectArray(path string) ([]interface{}, error) {
	val, err := s.Select(path)
	if err != nil {
		return nil, err
	}
	if arr, ok := val.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("value at path %s is not an array", path)
}

// SelectMap extracts a map/object value.
func (s *JSONSelector) SelectMap(path string) (map[string]interface{}, error) {
	val, err := s.Select(path)
	if err != nil {
		return nil, err
	}
	if m, ok := val.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("value at path %s is not an object", path)
}

// Exists returns true if the path exists in the JSON.
func (s *JSONSelector) Exists(path string) bool {
	_, err := s.Select(path)
	return err == nil
}

// selectPath navigates through the data structure using the path.
func selectPath(data interface{}, path string) (interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("nil data")
	}

	// Parse the path into segments
	segments := parsePath(path)
	current := data

	for _, seg := range segments {
		if current == nil {
			return nil, fmt.Errorf("null value at path segment: %s", seg)
		}

		// Handle array index
		if idx, isIndex := parseArrayIndex(seg); isIndex {
			arr, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array at %s", seg)
			}
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("array index out of bounds: %d", idx)
			}
			current = arr[idx]
			continue
		}

		// Handle object key
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[seg]
			if !exists {
				return nil, fmt.Errorf("key not found: %s", seg)
			}
			current = val
		case []interface{}:
			// Try to iterate and collect field from each element
			// This handles paths like "results.name" where results is an array
			return nil, fmt.Errorf("cannot access field %s on array", seg)
		default:
			return nil, fmt.Errorf("cannot access field %s on %T", seg, current)
		}
	}

	return current, nil
}

// parsePath splits a dot-notation path into segments.
func parsePath(path string) []string {
	var segments []string
	var current strings.Builder

	inBracket := false
	for _, r := range path {
		switch r {
		case '.':
			if !inBracket && current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			} else if inBracket {
				current.WriteRune(r)
			}
		case '[':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			inBracket = true
		case ']':
			if inBracket && current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			inBracket = false
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// parseArrayIndex checks if a segment is an array index and returns it.
func parseArrayIndex(seg string) (int, bool) {
	// Check for numeric string
	if idx, err := strconv.Atoi(seg); err == nil {
		return idx, true
	}
	return 0, false
}

// toString converts an interface value to string.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toInt converts an interface value to int.
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

// toFloat converts an interface value to float64.
func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// toBool converts an interface value to bool.
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		return val != "" && val != "0" && strings.ToLower(val) != "false"
	default:
		return false
	}
}

// ExtractJSONField extracts a value from JSON data using a Field definition.
func ExtractJSONField(data interface{}, field Field, ctx *TemplateContext) (string, error) {
	// Handle static text
	if field.Text != "" {
		engine := NewTemplateEngine()
		return engine.Evaluate(field.Text, ctx)
	}

	selector := NewJSONSelectorFromData(data)

	// Extract using selector path
	value, err := selector.SelectString(field.Selector)
	if err != nil {
		if field.Optional {
			return field.Default, nil
		}
		if field.Default != "" {
			return field.Default, nil
		}
		return "", nil
	}

	// Handle case mapping
	if len(field.Case) > 0 {
		if mapped, ok := field.Case[value]; ok {
			value = mapped
		} else if defaultVal, ok := field.Case["*"]; ok {
			value = defaultVal
		}
	}

	// Apply filters
	if len(field.Filters) > 0 {
		filtered, err := ApplyFilters(value, field.Filters)
		if err != nil {
			return "", err
		}
		value = filtered
	}

	// Use default if value is empty
	if value == "" && field.Default != "" {
		value = field.Default
	}

	return value, nil
}
