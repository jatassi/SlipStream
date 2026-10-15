package tokens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repo = "../../../.."

func design(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, designPath))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestLoad(t *testing.T) {
	theme, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Neutral != (Tint{214, 14}) || theme.Ceiling != 0.053 || theme.Spacing != "4px" {
		t.Errorf("neutral %v, ceiling %v, spacing %q", theme.Neutral, theme.Ceiling, theme.Spacing)
	}
	for name, want := range map[string]string{
		"page":       "hsl(var(--h) calc(var(--s) * 0.5) 5%)",
		"glass":      "hsl(var(--h) calc(var(--s) * 0.55) 8% / 0.56)",
		"fill-hover": "hsl(var(--h) 40% 92% / 0.16)",
		"ink-3":      "hsl(var(--h) 14% 71%)",
		"fail-dim":   "rgba(255, 226, 223, 0.74)",
		"solid":      "#f6f4f0",
	} {
		c, ok := theme.Color(name)
		if !ok || c.CSS() != want {
			t.Errorf("%s: got %q, want %q", name, c.CSS(), want)
		}
	}
	solid, _ := theme.Color("solid")
	if got := solid.At(Tint{40, 70}); got != (RGBA{246, 244, 240, 1}) {
		t.Errorf("solid takes the tint: %v", got)
	}
	if got := theme.Type[len(theme.Type)-1]; got != (Role{"data", "0.92em", 400, "1.45", "-0.01em"}) {
		t.Errorf("data role: %+v", got)
	}
	if got := theme.Shadows[2]; got != (Named{"signal-lift", "inset 0 0 0 1px var(--color-fail-line), 0 18px 40px -24px rgba(0,0,0,.8)"}) {
		t.Errorf("signal-lift: %+v", got)
	}
}

func TestParseRefuses(t *testing.T) {
	tests := []struct {
		name, old, new, want string
	}{
		{
			name: "formula off its neutral value",
			old:  "| `ink-3` | `hsl(var(--h) 14% 71%)` |",
			new:  "| `ink-3` | `hsl(var(--h) 15% 71%)` |",
			want: "colors.ink-3 is hsl(214 14% 71%), but its formula resolves to hsl(214 15% 71% / 1) at the neutral tint",
		},
		{
			name: "scaled formula off its neutral value",
			old:  "| `sheet` | `hsl(var(--h) calc(var(--s) * .4) 10% / .88)` |",
			new:  "| `sheet` | `hsl(var(--h) calc(var(--s) * .4) 10% / .8)` |",
			want: "colors.sheet is hsl(214 5.6% 10% / 0.88), but its formula resolves to hsl(214 5.6000000000000005% 10% / 0.8) at the neutral tint",
		},
		{
			name: "unknown token",
			old:  "| `glow` |",
			new:  "| `mist` | `hsl(var(--h) 1% 2%)` |\n| `glow` |",
			want: "the formula table names mist, which colors does not",
		},
		{
			name: "tinted token without a formula",
			old:  "| `fill-hover` | `hsl(var(--h) 40% 92% / .16)` |\n",
			new:  "",
			want: "colors.fill-hover has no row in the formula table",
		},
		{
			name: "formula for a fixed signal",
			old:  "| `glow` |",
			new:  "| `fail` | `hsl(var(--h) 90% 60%)` |\n| `glow` |",
			want: "colors.fail is a fixed signal and takes no formula",
		},
		{
			name: "shorthand row",
			old:  "| `fill` | `hsl(var(--h) 40% 92% / .1)` |\n| `fill-hover` | `hsl(var(--h) 40% 92% / .16)` |",
			new:  "| `fill`, `fill-hover` | `hsl(var(--h) 40% 92% / .1)`, `/ .16` |",
			want: "formula table row \"| `fill`, `fill-hover` | `hsl(var(--h) 40% 92% / .1)`, `/ .16` |\" is not | `token` | `formula` |",
		},
		{
			name: "in-between weight",
			old:  "    fontSize: \"14px\"\n    fontWeight: 600",
			new:  "    fontSize: \"14px\"\n    fontWeight: 650",
			want: "typography.heading weight 650 is not one of [400 500 600 700]",
		},
		{
			name: "spacing off the grid",
			old:  `"3": "12px"`,
			new:  `"3": "13px"`,
			want: "spacing.3 is 13px, not 12px on the 4px grid",
		},
		{
			name: "shadow naming no color",
			old:  "{colors.fail-line}",
			new:  "{colors.fail-edge}",
			want: "shadows.signal-lift refers to {colors.fail-edge}, which is not a colors token",
		},
	}
	doc := design(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(doc, tt.old) {
				t.Fatalf("DESIGN.md has no %q", tt.old)
			}
			_, err := Parse([]byte(strings.Replace(doc, tt.old, tt.new, 1)))
			if err == nil || err.Error() != tt.want {
				t.Errorf("got %v, want %s", err, tt.want)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, designPath), []byte(design(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "web", "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	theme := filepath.Join(root, filepath.FromSlash(themePath))
	if err := os.WriteFile(theme, []byte("@theme {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err == nil || err.Error() != "web/src/theme.gen.css is stale; run go run ./cmd/release gen tokens" {
		t.Fatalf("stale theme: %v", err)
	}
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatalf("after Write: %v", err)
	}
	css, err := os.ReadFile(theme)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"  --color-ink-1: hsl(var(--h) 25% 96%);",
		"  --color-ok: #5fd394;",
		"  --text-label-caps--letter-spacing: 0.06em;",
		"  --text-body--font-weight: 400;",
		"  --radius-capsule: 999px;",
		"  --shadow-float: inset 0 1px 0 rgba(255,255,255,.1), inset 0 0 0 1px var(--color-hairline), 0 18px 40px rgba(0,0,0,.55);",
		"  --ease-spring: cubic-bezier(.32, .72, 0, 1);",
		"  --duration-dialog: 360ms;",
		"    --color-glass: hsl(var(--h) calc(var(--s) * 0.55) 8%);",
		"    --color-hairline: hsl(var(--h) 40% 88%);",
		"  initial-value: 14%;",
		"  --text-*: initial;",
		"  --radius-*: initial;",
		"  --shadow-*: initial;",
		"  --ease-*: initial;",
	} {
		if !strings.Contains(string(css), line+"\n") {
			t.Errorf("theme lacks %q", line)
		}
	}
}
