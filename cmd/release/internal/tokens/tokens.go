// Package tokens reads DESIGN.md's design tokens, writes the web theme from
// them and proves their contrast over every tint the ambient can take.
package tokens

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

const designPath = "DESIGN.md"

// Tint is a value of the two registered properties the ambient drives:
// --h, a hue in degrees, and --s, a saturation in percent.
type Tint struct {
	H, S float64
}

// Formula is a tinted color: hsl(var(--h) <sat> <light>% / <alpha>). Its
// saturation is fixed, or a share of --s when Scaled.
type Formula struct {
	Sat    float64
	Scaled bool
	Light  float64
	Alpha  float64
}

// At resolves the formula at a tint.
func (f Formula) At(t Tint) HSLA {
	sat := f.Sat
	if f.Scaled {
		sat = t.S * f.Sat
	}
	return HSLA{t.H, sat, f.Light, f.Alpha}
}

func (f Formula) css(alpha bool) string {
	sat := num(f.Sat) + "%"
	if f.Scaled {
		sat = "calc(var(--s) * " + num(f.Sat) + ")"
	}
	out := "hsl(var(--h) " + sat + " " + num(f.Light) + "%"
	if alpha && f.Alpha != 1 {
		out += " / " + num(f.Alpha)
	}
	return out + ")"
}

// Color is one colors token. A tinted token has a Formula, and a fixed
// signal token is its literal Value.
type Color struct {
	Name    string
	Value   string
	Formula *Formula
	fixed   RGBA
}

// At resolves the color at a tint.
func (c Color) At(t Tint) RGBA {
	if c.Formula == nil {
		return c.fixed
	}
	return c.Formula.At(t).RGBA()
}

// CSS is the color's value in the theme.
func (c Color) CSS() string {
	if c.Formula == nil {
		return c.Value
	}
	return c.Formula.css(true)
}

// Role is one typography role, emitted as a text-<name> utility.
type Role struct {
	Name          string
	Size          string
	Weight        int
	LineHeight    string
	LetterSpacing string
}

// Theme is DESIGN.md's tokens, in the frontmatter's order.
type Theme struct {
	Neutral   Tint
	Ceiling   float64
	Colors    []Color
	Type      []Role
	Radii     []Named
	Spacing   string
	Shadows   []Named
	Easings   []Named
	Durations []Named
}

// Color finds a colors token by name.
func (t *Theme) Color(name string) (Color, bool) {
	i := slices.IndexFunc(t.Colors, func(c Color) bool { return c.Name == name })
	if i < 0 {
		return Color{}, false
	}
	return t.Colors[i], true
}

// Load reads DESIGN.md under root.
func Load(root string) (*Theme, error) {
	data, err := os.ReadFile(filepath.Join(root, designPath))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", designPath, err)
	}
	return Parse(data)
}

type frontmatter struct {
	Colors     yaml.Node `yaml:"colors"`
	Typography yaml.Node `yaml:"typography"`
	Rounded    yaml.Node `yaml:"rounded"`
	Spacing    yaml.Node `yaml:"spacing"`
	Shadows    yaml.Node `yaml:"shadows"`
	Motion     struct {
		Easing   yaml.Node `yaml:"easing"`
		Duration yaml.Node `yaml:"duration"`
	} `yaml:"motion"`
	Ambient struct {
		Neutral struct {
			Hue        float64 `yaml:"hue"`
			Saturation string  `yaml:"saturation"`
		} `yaml:"neutral"`
		Ceiling float64 `yaml:"ceiling"`
	} `yaml:"ambient"`
}

type typeSpec struct {
	FontSize      string `yaml:"fontSize"`
	FontWeight    int    `yaml:"fontWeight"`
	LineHeight    string `yaml:"lineHeight"`
	LetterSpacing string `yaml:"letterSpacing"`
}

// Parse reads DESIGN.md's frontmatter and its formula table and checks that
// they agree: every tinted color resolves, at the neutral tint, to its
// frontmatter value.
func Parse(design []byte) (*Theme, error) {
	head, body, err := split(design)
	if err != nil {
		return nil, err
	}
	var fm frontmatter
	if err := yaml.Unmarshal(head, &fm); err != nil {
		return nil, fmt.Errorf("%s frontmatter: %w", designPath, err)
	}
	neutralSat, ok := strings.CutSuffix(fm.Ambient.Neutral.Saturation, "%")
	if !ok {
		return nil, fmt.Errorf("ambient.neutral.saturation %q is not a percentage", fm.Ambient.Neutral.Saturation)
	}
	t := &Theme{Ceiling: fm.Ambient.Ceiling}
	t.Neutral.H = fm.Ambient.Neutral.Hue
	if t.Neutral.S, err = strconv.ParseFloat(neutralSat, 64); err != nil {
		return nil, fmt.Errorf("ambient.neutral.saturation: %w", err)
	}
	formulas, err := formulaTable(body)
	if err != nil {
		return nil, err
	}
	if t.Colors, err = colors(&fm.Colors, formulas, t.Neutral); err != nil {
		return nil, err
	}
	for _, name := range opaqueUnderPreferences {
		if c, ok := t.Color(name); !ok || c.Formula == nil {
			return nil, fmt.Errorf("colors.%s must be a tinted token", name)
		}
	}
	if t.Type, err = roles(&fm.Typography); err != nil {
		return nil, err
	}
	if t.Radii, err = entries[string](&fm.Rounded, "rounded"); err != nil {
		return nil, err
	}
	if t.Spacing, err = spacing(&fm.Spacing); err != nil {
		return nil, err
	}
	if t.Shadows, err = shadows(&fm.Shadows, t); err != nil {
		return nil, err
	}
	if t.Easings, err = entries[string](&fm.Motion.Easing, "motion.easing"); err != nil {
		return nil, err
	}
	if t.Durations, err = entries[string](&fm.Motion.Duration, "motion.duration"); err != nil {
		return nil, err
	}
	return t, nil
}

func split(design []byte) (head, body []byte, err error) {
	rest, ok := bytes.CutPrefix(design, []byte("---\n"))
	if !ok {
		return nil, nil, fmt.Errorf("%s does not open with a frontmatter", designPath)
	}
	head, body, ok = bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return nil, nil, fmt.Errorf("%s frontmatter is not closed", designPath)
	}
	return head, body, nil
}

type entry[T any] struct {
	Name  string
	Value T
}

// Named is a token that is its CSS value as written.
type Named = entry[string]

func entries[T any](n *yaml.Node, section string) ([]entry[T], error) {
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter %s is missing or not a map", section)
	}
	out := make([]entry[T], 0, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		var v T
		if err := n.Content[i+1].Decode(&v); err != nil {
			return nil, fmt.Errorf("frontmatter %s.%s: %w", section, n.Content[i].Value, err)
		}
		out = append(out, entry[T]{n.Content[i].Value, v})
	}
	return out, nil
}

var (
	tableRow   = regexp.MustCompile("^\\| `([a-z0-9-]+)` \\| `([^`]+)` \\|$")
	formulaCSS = regexp.MustCompile(`^hsl\(var\(--h\) (?:calc\(var\(--s\) \* ([0-9.]+)\)|([0-9.]+)%) ([0-9.]+)%(?: / ([0-9.]+))?\)$`)
)

const tableHeader = "| Token | Formula |"

func formulaTable(body []byte) (map[string]Formula, error) {
	lines := strings.Split(string(body), "\n")
	start := slices.Index(lines, tableHeader)
	if start < 0 {
		return nil, fmt.Errorf("%s has no %q table", designPath, tableHeader)
	}
	out := map[string]Formula{}
	for _, line := range lines[start+2:] {
		if !strings.HasPrefix(line, "|") {
			break
		}
		m := tableRow.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("formula table row %q is not | `token` | `formula` |", line)
		}
		if _, dup := out[m[1]]; dup {
			return nil, fmt.Errorf("formula table names %s twice", m[1])
		}
		f, err := parseFormula(m[2])
		if err != nil {
			return nil, fmt.Errorf("formula for %s: %w", m[1], err)
		}
		out[m[1]] = f
	}
	return out, nil
}

func parseFormula(s string) (Formula, error) {
	m := formulaCSS.FindStringSubmatch(s)
	if m == nil {
		return Formula{}, fmt.Errorf("%q is not hsl(var(--h) <sat> <light>%%[ / <alpha>])", s)
	}
	sat := m[2]
	if m[1] != "" {
		sat = m[1]
	}
	v, err := floats(sat, m[3], cmp.Or(m[4], "1"))
	if err != nil {
		return Formula{}, fmt.Errorf("%q: %w", s, err)
	}
	return Formula{Sat: v[0], Scaled: m[1] != "", Light: v[1], Alpha: v[2]}, nil
}

// fixedSignal names the colors the Fixed Signal Rule keeps out of the tint.
func fixedSignal(name string) bool {
	if name == "solid" || name == "on-solid" {
		return true
	}
	for _, family := range []string{"fail", "warn", "ok"} {
		if name == family || strings.HasPrefix(name, family+"-") {
			return true
		}
	}
	return false
}

func colors(n *yaml.Node, formulas map[string]Formula, neutral Tint) ([]Color, error) {
	values, err := entries[string](n, "colors")
	if err != nil {
		return nil, err
	}
	var out []Color
	for _, v := range values {
		c := Color{Name: v.Name, Value: v.Value}
		f, tinted := formulas[v.Name]
		delete(formulas, v.Name)
		switch {
		case tinted && fixedSignal(v.Name):
			return nil, fmt.Errorf("colors.%s is a fixed signal and takes no formula", v.Name)
		case tinted:
			if err := checkNeutral(v.Name, v.Value, f.At(neutral)); err != nil {
				return nil, err
			}
			c.Formula = &f
		case fixedSignal(v.Name):
			if c.fixed, err = parseLiteral(v.Value); err != nil {
				return nil, fmt.Errorf("colors.%s: %w", v.Name, err)
			}
		default:
			return nil, fmt.Errorf("colors.%s has no row in the formula table", v.Name)
		}
		out = append(out, c)
	}
	if len(formulas) > 0 {
		return nil, fmt.Errorf("the formula table names %s, which colors does not", strings.Join(slices.Sorted(maps.Keys(formulas)), ", "))
	}
	return out, nil
}

func checkNeutral(name, value string, want HSLA) error {
	got, err := parseHSL(value)
	if err != nil {
		return fmt.Errorf("colors.%s: %w", name, err)
	}
	const eps = 1e-9
	if math.Abs(got.H-want.H) > eps || math.Abs(got.S-want.S) > eps || math.Abs(got.L-want.L) > eps || math.Abs(got.A-want.A) > eps {
		return fmt.Errorf("colors.%s is %s, but its formula resolves to %s at the neutral tint", name, value, want)
	}
	return nil
}

var weights = []int{400, 500, 600, 700}

func roles(n *yaml.Node) ([]Role, error) {
	specs, err := entries[typeSpec](n, "typography")
	if err != nil {
		return nil, err
	}
	out := make([]Role, 0, len(specs))
	for _, s := range specs {
		if !slices.Contains(weights, s.Value.FontWeight) {
			return nil, fmt.Errorf("typography.%s weight %d is not one of %v", s.Name, s.Value.FontWeight, weights)
		}
		if s.Value.FontSize == "" || s.Value.LineHeight == "" {
			return nil, fmt.Errorf("typography.%s needs a fontSize and a lineHeight", s.Name)
		}
		out = append(out, Role{s.Name, s.Value.FontSize, s.Value.FontWeight, s.Value.LineHeight, s.Value.LetterSpacing})
	}
	return out, nil
}

// spacing checks that every step sits on the 4px grid and returns the unit.
func spacing(n *yaml.Node) (string, error) {
	steps, err := entries[string](n, "spacing")
	if err != nil {
		return "", err
	}
	const unit = 4.0
	for _, s := range steps {
		step, err := strconv.ParseFloat(s.Name, 64)
		if err != nil {
			return "", fmt.Errorf("spacing step %q: %w", s.Name, err)
		}
		if s.Value != num(step*unit)+"px" {
			return "", fmt.Errorf("spacing.%s is %s, not %s on the 4px grid", s.Name, s.Value, num(step*unit)+"px")
		}
	}
	return num(unit) + "px", nil
}

var colorRef = regexp.MustCompile(`\{([a-z]+)\.([a-z0-9-]+)\}`)

// shadows resolves {colors.<name>} references to the theme's variables.
func shadows(n *yaml.Node, t *Theme) ([]Named, error) {
	values, err := entries[string](n, "shadows")
	if err != nil {
		return nil, err
	}
	out := make([]Named, 0, len(values))
	for _, v := range values {
		var bad error
		css := colorRef.ReplaceAllStringFunc(v.Value, func(ref string) string {
			m := colorRef.FindStringSubmatch(ref)
			if _, ok := t.Color(m[2]); m[1] != "colors" || !ok {
				bad = fmt.Errorf("shadows.%s refers to %s, which is not a colors token", v.Name, ref)
			}
			return "var(--color-" + m[2] + ")"
		})
		if bad != nil {
			return nil, bad
		}
		out = append(out, Named{v.Name, css})
	}
	return out, nil
}

func num(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
