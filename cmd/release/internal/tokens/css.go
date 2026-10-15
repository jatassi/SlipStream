package tokens

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const generateCommand = "go run ./cmd/release gen tokens"

// outputs are the files the generator writes, each rendered from the theme.
var outputs = []struct {
	path   string
	render func(*Theme) []byte
}{
	{"web/src/theme.gen.css", (*Theme).CSS},
	{"web/src/theme.gen.ts", (*Theme).TS},
}

var weightNames = map[int]string{400: "normal", 500: "medium", 600: "semibold", 700: "bold"}

// opaqueUnderPreferences names the tinted tokens the preference overrides
// redraw without alpha.
var opaqueUnderPreferences = append(slices.Clone(surfaces), "hairline-strong")

// surfaces are the translucent materials the Reduce Transparency Rule
// turns opaque at the same tint and lightness.
var surfaces = []string{"glass", "glass-raised", "chrome", "popover", "sheet", "well"}

// CSS renders the Tailwind 4 theme: the registered tint properties, the
// tokens, and the preference overrides DESIGN.md's Named Rules ask for.
func (t *Theme) CSS() []byte {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	p("/* Generated from DESIGN.md by %s. Do not edit. */", generateCommand)
	p("")
	p("@property --h {\n  syntax: \"<number>\";\n  inherits: true;\n  initial-value: %s;\n}", num(t.Neutral.H))
	p("")
	p("@property --s {\n  syntax: \"<percentage>\";\n  inherits: true;\n  initial-value: %s%%;\n}", num(t.Neutral.S))
	p("")
	p("@theme static {")
	p("  --color-*: initial;")
	p("  --font-*: initial;")
	p("  --font-weight-*: initial;")
	p("  --text-*: initial;")
	p("  --radius-*: initial;")
	p("  --shadow-*: initial;")
	p("  --ease-*: initial;")
	p("  --spacing: %s;", t.Spacing)
	p("")
	for _, c := range t.Colors {
		p("  --color-%s: %s;", c.Name, c.CSS())
	}
	p("")
	for _, w := range weights {
		p("  --font-weight-%s: %d;", weightNames[w], w)
	}
	for _, r := range t.Type {
		p("")
		p("  --text-%s: %s;", r.Name, r.Size)
		p("  --text-%s--line-height: %s;", r.Name, r.LineHeight)
		if r.LetterSpacing != "" {
			p("  --text-%s--letter-spacing: %s;", r.Name, r.LetterSpacing)
		}
		p("  --text-%s--font-weight: %d;", r.Name, r.Weight)
	}
	for _, group := range []struct {
		prefix string
		tokens []Named
	}{{"radius", t.Radii}, {"shadow", t.Shadows}, {"ease", t.Easings}, {"duration", t.Durations}} {
		p("")
		for _, n := range group.tokens {
			p("  --%s-%s: %s;", group.prefix, n.Name, n.Value)
		}
	}
	p("}")
	p("")
	p("@media (prefers-reduced-transparency: reduce) {\n  :root {")
	for _, name := range surfaces {
		p("    --color-%s: %s;", name, t.formula(name).css(false))
	}
	p("  }\n}")
	p("")
	strong := t.formula("hairline-strong").css(false)
	p("@media (prefers-contrast: more) {\n  :root {")
	p("    --color-ink-2: var(--color-ink-1);")
	p("    --color-ink-3: var(--color-ink-1);")
	p("    --color-hairline: %s;", strong)
	p("    --color-hairline-strong: %s;", strong)
	p("  }\n}")
	return []byte(b.String())
}

// TS renders the ambient's constants for the web app's sampler.
func (t *Theme) TS() []byte {
	return []byte(fmt.Sprintf(`/* Generated from DESIGN.md by %s. Do not edit. */

/** DESIGN.md's ambient.neutral, the registered initial values of `+"`--h` and `--s`"+`. */
export const neutralTint = { hue: %s, saturation: %s } as const;

/** DESIGN.md's ambient.ceiling: the brightest relative luminance the ambient may composite to. */
export const ceiling = %s;
`, generateCommand, num(t.Neutral.H), num(t.Neutral.S), num(t.Ceiling)))
}

// formula is a tinted token's formula, which Parse has checked exists.
func (t *Theme) formula(name string) Formula {
	c, _ := t.Color(name)
	return *c.Formula
}

// Write regenerates the web theme from DESIGN.md.
func Write(root string) error {
	t, err := Load(root)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		if err := os.WriteFile(filepath.Join(root, out.path), out.render(t), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", out.path, err)
		}
	}
	return nil
}

// Check reports a committed web theme file that DESIGN.md no longer generates.
func Check(root string) error {
	t, err := Load(root)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		got, err := os.ReadFile(filepath.Join(root, out.path))
		if err != nil {
			return fmt.Errorf("read %s: %w", out.path, err)
		}
		if !bytes.Equal(got, out.render(t)) {
			return fmt.Errorf("%s is stale; run %s", out.path, generateCommand)
		}
	}
	return nil
}
