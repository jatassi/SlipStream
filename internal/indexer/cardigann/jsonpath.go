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

	segments := parsePath(path)
	current := data

	for _, seg := range segments {
		if current == nil {
			return nil, fmt.Errorf("null value at path segment: %s", seg)
		}

		var err error
		current, err = navigateSegment(current, seg)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

func navigateSegment(current interface{}, seg string) (interface{}, error) {
	if idx, isIndex := parseArrayIndex(seg); isIndex {
		return accessArrayIndex(current, seg, idx)
	}
	return accessObjectKey(current, seg)
}

func accessArrayIndex(current interface{}, seg string, idx int) (interface{}, error) {
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
	return arr[idx], nil
}

func accessObjectKey(current interface{}, seg string) (interface{}, error) {
	m, ok := current.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot access field %s on %T", seg, current)
	}
	val, exists := m[seg]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", seg)
	}
	return val, nil
}

// parsePath splits a dot-notation path into segments.
func parsePath(path string) []string {
	var segments []string
	var current strings.Builder
	inBracket := false

	for _, r := range path {
		segments, inBracket = parsePathRune(r, &current, segments, inBracket)
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

func parsePathRune(r rune, current *strings.Builder, segments []string, inBracket bool) ([]string, bool) {
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
	return segments, inBracket
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
		return val != "" && val != "0" && !strings.EqualFold(val, "false")
	default:
		return false
	}
}

// applyCaseMappingJSON applies case mapping to value (helper for JSON extraction)
func applyCaseMappingJSON(value string, caseMap map[string]string) string {
	if len(caseMap) == 0 {
		return value
	}

	if mapped, ok := caseMap[value]; ok {
		return mapped
	}
	if defaultVal, ok := caseMap["*"]; ok {
		return defaultVal
	}
	return value
}

// getDefaultValueJSON returns default if value is empty (helper for JSON extraction)
func getDefaultValueJSON(value string, field *Field) string {
	if value == "" && field.Default != "" {
		return field.Default
	}
	return value
}

// ExtractJSONField extracts a value from JSON data using a Field definition.
func ExtractJSONField(data interface{}, field *Field, ctx *TemplateContext) (string, error) {
	if field.Text != "" {
		engine := NewTemplateEngine()
		return engine.Evaluate(field.Text, ctx)
	}

	selector := NewJSONSelectorFromData(data)
	value, err := selector.SelectString(field.Selector)
	if err != nil {
		if field.Optional || field.Default != "" {
			return field.Default, nil
		}
		return "", nil
	}

	value = applyCaseMappingJSON(value, field.Case)

	if len(field.Filters) > 0 {
		filtered, err := ApplyFilters(value, field.Filters)
		if err != nil {
			return "", err
		}
		value = filtered
	}

	value = getDefaultValueJSON(value, field)
	return value, nil
}
