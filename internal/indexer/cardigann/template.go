package cardigann

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// TemplateContext provides data available during template evaluation.
type TemplateContext struct {
	Config     map[string]string // User-provided settings
	Query      QueryContext      // Search parameters
	Keywords   string            // Top-level alias for Query.Keywords (Cardigann compatibility)
	Categories []string          // Selected category IDs
	Result     map[string]string // Previously extracted fields (for field references)
	Today      TimeContext       // Current date/time
}

// QueryContext contains search query parameters.
type QueryContext struct {
	Q           string // Raw search query
	Keywords    string // Processed keywords
	Series      string // TV series name
	Movie       string // Movie name
	Year        int
	Season      int
	Ep          int
	Episode     int    // Alias for Ep
	IMDBID      string // With "tt" prefix
	IMDBIDShort string // Without "tt" prefix
	TMDBID      int
	TVDBID      int
	Album       string
	Artist      string
	Author      string
	Title       string
	Page        int
	Limit       int
	Offset      int
}

// TimeContext provides date/time information.
type TimeContext struct {
	Year  int
	Month int
	Day   int
}

// TemplateEngine evaluates template expressions in definition strings.
type TemplateEngine struct {
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine with all built-in functions.
func NewTemplateEngine() *TemplateEngine {
	e := &TemplateEngine{
		funcMap: make(template.FuncMap),
	}

	// Register built-in functions
	e.funcMap["join"] = funcJoin
	e.funcMap["re_replace"] = funcReReplace
	e.funcMap["replace"] = funcReplace
	e.funcMap["split"] = funcSplit
	e.funcMap["trim"] = funcTrim
	e.funcMap["trimleft"] = funcTrimLeft
	e.funcMap["trimright"] = funcTrimRight
	e.funcMap["tolower"] = strings.ToLower
	e.funcMap["toupper"] = strings.ToUpper
	e.funcMap["prepend"] = funcPrepend
	e.funcMap["append"] = funcAppend
	e.funcMap["if"] = funcIf
	e.funcMap["default"] = funcDefault

	return e
}

// NewTemplateContext creates a context with current time populated.
func NewTemplateContext() *TemplateContext {
	now := time.Now()
	return &TemplateContext{
		Config:     make(map[string]string),
		Categories: []string{},
		Result:     make(map[string]string),
		Today: TimeContext{
			Year:  now.Year(),
			Month: int(now.Month()),
			Day:   now.Day(),
		},
	}
}

// Evaluate processes a template string with the given context.
func (e *TemplateEngine) Evaluate(tmplStr string, ctx *TemplateContext) (string, error) {
	// Quick check for no template markers
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr, nil
	}

	// Preprocess to handle Cardigann-specific syntax
	tmplStr = e.preprocessTemplate(tmplStr)

	tmpl, err := template.New("").Funcs(e.funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}

	return buf.String(), nil
}

// EvaluateAll processes multiple template strings.
func (e *TemplateEngine) EvaluateAll(templates map[string]string, ctx *TemplateContext) (map[string]string, error) {
	result := make(map[string]string, len(templates))
	for key, tmpl := range templates {
		val, err := e.Evaluate(tmpl, ctx)
		if err != nil {
			return nil, fmt.Errorf("error evaluating %s: %w", key, err)
		}
		result[key] = val
	}
	return result, nil
}

// preprocessTemplate converts Cardigann-specific syntax to Go template syntax.
func (e *TemplateEngine) preprocessTemplate(tmpl string) string {
	// Handle .Keywords shorthand (alias for .Query.Keywords)
	tmpl = strings.ReplaceAll(tmpl, "{{ .Keywords }}", "{{ .Query.Keywords }}")
	tmpl = strings.ReplaceAll(tmpl, "{{.Keywords}}", "{{.Query.Keywords}}")

	// Handle direct variable references without .Query prefix
	shortcuts := map[string]string{
		".IMDBID":      ".Query.IMDBID",
		".IMDBIDShort": ".Query.IMDBIDShort",
		".TMDBID":      ".Query.TMDBID",
		".TVDBID":      ".Query.TVDBID",
		".Season":      ".Query.Season",
		".Ep":          ".Query.Ep",
		".Episode":     ".Query.Episode",
		".Year":        ".Query.Year",
		".Series":      ".Query.Series",
		".Movie":       ".Query.Movie",
		".Album":       ".Query.Album",
		".Artist":      ".Query.Artist",
		".Author":      ".Query.Author",
		".Title":       ".Query.Title",
	}

	for short, full := range shortcuts {
		// Only replace when it's a standalone reference, not already qualified
		tmpl = regexp.MustCompile(`\{\{\s*`+regexp.QuoteMeta(short)+`\s*\}\}`).ReplaceAllString(
			tmpl, "{{ "+full+" }}")
	}

	return tmpl
}

// Template functions

func funcJoin(arr interface{}, sep string) string {
	switch v := arr.(type) {
	case []string:
		return strings.Join(v, sep)
	case []interface{}:
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(strs, sep)
	default:
		return fmt.Sprintf("%v", arr)
	}
}

func funcReReplace(input, pattern, replacement string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return input
	}
	return re.ReplaceAllString(input, replacement)
}

func funcReplace(input, old, newVal string) string {
	return strings.ReplaceAll(input, old, newVal)
}

func funcSplit(input, sep string) []string {
	return strings.Split(input, sep)
}

func funcTrim(input string, args ...string) string {
	if len(args) > 0 {
		return strings.Trim(input, args[0])
	}
	return strings.TrimSpace(input)
}

func funcTrimLeft(input string, args ...string) string {
	if len(args) > 0 {
		return strings.TrimLeft(input, args[0])
	}
	return strings.TrimLeft(input, " \t\n\r")
}

func funcTrimRight(input string, args ...string) string {
	if len(args) > 0 {
		return strings.TrimRight(input, args[0])
	}
	return strings.TrimRight(input, " \t\n\r")
}

func funcPrepend(input, prefix string) string {
	return prefix + input
}

func funcAppend(input, suffix string) string {
	return input + suffix
}

func funcIf(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func funcDefault(value, defaultValue interface{}) interface{} {
	if value == nil || value == "" {
		return defaultValue
	}
	return value
}
