package tokens

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// pair is an ink over a stack of surfaces, listed top to bottom, laid on
// the ambient.
type pair struct {
	ink   string
	stack []string
	min   float64
	label string
}

var pairs = []pair{
	{"ink-1", nil, 4.5, "primary text on bare ambient"},
	{"ink-2", nil, 4.5, "secondary text on bare ambient"},
	{"ink-3", nil, 4.5, "tertiary text on bare ambient"},
	{"ink-4", nil, 3.0, "separators and idle marks on bare ambient (non-text)"},
	{"ink-3", []string{"glass"}, 4.5, "tertiary text on glass"},
	{"ink-3", []string{"chrome"}, 4.5, "tertiary text on chrome"},
	{"ink-3", []string{"glass-raised", "glass"}, 4.5, "tertiary text on raised glass"},
	{"ink-3", []string{"fill", "chrome"}, 4.5, "placeholder in a top-bar field"},
	{"ink-2", []string{"fill", "glass"}, 4.5, "ghost button text on glass"},
	{"ink-1", []string{"fill-hover", "glass"}, 4.5, "button text, hovered, on glass"},
	{"ink-1", []string{"selected", "fill", "glass"}, 4.5, "selected segment"},
	{"ink-2", []string{"fill", "glass"}, 4.5, "unselected segment"},
	{"ink-3", []string{"popover"}, 4.5, "menu meta on popover"},
	{"ink-3", []string{"sheet"}, 4.5, "sheet subtitle"},
	{"ink-1", []string{"live-fill", "glass"}, 4.5, "live chip label"},
	{"fail-ink", nil, 4.5, "failure link in the status sentence"},
	{"warn", nil, 4.5, "warning link in the status sentence"},
	{"fail-ink", []string{"glass"}, 4.5, "failure ink on glass"},
	{"fail-text", []string{"fail-card"}, 4.5, "text on a failure card"},
	{"fail-dim", []string{"fail-card"}, 4.5, "dim text on a failure card"},
	{"fail-ink", []string{"fail-card"}, 4.5, "labels on a failure card"},
	{"warn-text", []string{"warn-card"}, 4.5, "text on a warning line"},
	{"warn-dim", []string{"warn-card"}, 4.5, "dim text on a warning line"},
	{"warn", []string{"warn-card"}, 4.5, "amber label on a warning line"},
	{"fail-deep", []string{"fail"}, 4.5, "ink on a failure chip or button"},
	{"warn-deep", []string{"warn"}, 4.5, "ink on a warning chip or button"},
	{"ok-deep", []string{"ok"}, 4.5, "ink on a healthy chip"},
	{"on-solid", []string{"solid"}, 4.5, "ink on solid white (primary, Met)"},
	{"fail", []string{"glass"}, 3.0, "failure dot or icon on glass (non-text)"},
	{"warn", []string{"glass"}, 3.0, "warning dot or icon on glass (non-text)"},
	{"ok", []string{"glass"}, 3.0, "healthy dot or icon on glass (non-text)"},
	{"ok", nil, 3.0, "healthy dot or icon on bare ambient (non-text)"},
}

// The tints the ambient sampler can emit: every third hue, at the neutral
// saturation and across the sampler's 18% to 70% clamp.
var (
	sweepHues = func() []float64 {
		var out []float64
		for h := 0; h < 360; h += 3 {
			out = append(out, float64(h))
		}
		return out
	}()
	sweepSats = []float64{14, 18, 30, 45, 60, 70}
)

// Verdict is a pair's worst contrast across the sweep.
type Verdict struct {
	Label string
	Ratio float64
	Min   float64
	Worst Tint
}

func (v Verdict) Pass() bool { return v.Ratio >= v.Min }

func (v Verdict) String() string {
	mark := "ok  "
	if !v.Pass() {
		mark = "FAIL"
	}
	return fmt.Sprintf("%s %5.2f >= %.1f  %-52s worst at h%s s%s", mark, v.Ratio, v.Min, v.Label, num(v.Worst.H), num(v.Worst.S))
}

// Proof is the brightest ambient every pair survives, and each pair's
// worst contrast on it.
type Proof struct {
	Ceiling  float64
	Verdicts []Verdict
}

// Prove sweeps the tints, finds the ambient ceiling and checks every pair
// at it, with the numbers of the contrast.py it replaced.
func (t *Theme) Prove() (Proof, error) {
	for _, p := range pairs {
		for _, name := range append([]string{p.ink}, p.stack...) {
			if _, ok := t.Color(name); !ok {
				return Proof{}, fmt.Errorf("contrast pair %q names %s, which is not a colors token", p.label, name)
			}
		}
	}
	lo, hi := 0.0, 0.5
	for range 30 {
		mid := (lo + hi) / 2
		if passes(t.worst(mid)) {
			lo = mid
		} else {
			hi = mid
		}
	}
	ceiling, err := strconv.ParseFloat(strconv.FormatFloat(lo-0.0005, 'f', 3, 64), 64)
	if err != nil {
		return Proof{}, fmt.Errorf("round the ceiling: %w", err)
	}
	verdicts := t.worst(ceiling)
	slices.SortStableFunc(verdicts, func(a, b Verdict) int {
		return cmp.Compare(a.Ratio/a.Min, b.Ratio/b.Min)
	})
	return Proof{ceiling, verdicts}, nil
}

func passes(vs []Verdict) bool {
	return !slices.ContainsFunc(vs, func(v Verdict) bool { return !v.Pass() })
}

// worst lays every pair over an ambient of luminance y at each tint.
func (t *Theme) worst(y float64) []Verdict {
	out := make([]Verdict, len(pairs))
	for i, p := range pairs {
		out[i] = Verdict{Label: p.label, Ratio: math.Inf(1), Min: p.min}
	}
	for _, h := range sweepHues {
		for _, s := range sweepSats {
			tint := Tint{h, s}
			amb := ambient(tint, y)
			for i, p := range pairs {
				bg := amb
				for _, layer := range slices.Backward(p.stack) {
					bg = over(t.at(layer, tint), bg)
				}
				fg := t.at(p.ink, tint)
				if fg.A != 1 {
					fg = over(fg, bg)
				}
				if r := contrast(fg, bg); r < out[i].Ratio {
					out[i].Ratio, out[i].Worst = r, tint
				}
			}
		}
	}
	return out
}

func (t *Theme) at(name string, tint Tint) RGBA {
	c, _ := t.Color(name)
	return c.At(tint)
}

// ambient is the brightest the ambient may be at a tint: its hue at 1.3
// times the saturation, at the lightness whose luminance reaches y.
func ambient(t Tint, y float64) RGBA {
	sat := math.Min(100, t.S*1.3)
	lo, hi := 0.0, 100.0
	for range 40 {
		mid := (lo + hi) / 2
		if luminance(HSLA{t.H, sat, mid, 1}.RGBA()) < y {
			lo = mid
		} else {
			hi = mid
		}
	}
	return HSLA{t.H, sat, lo, 1}.RGBA()
}

// CheckContrast proves DESIGN.md's colors hold WCAG 2.2 AA over every tint
// at the ambient ceiling DESIGN.md records.
func CheckContrast(root string) error {
	t, err := Load(root)
	if err != nil {
		return err
	}
	proof, err := t.Prove()
	if err != nil {
		return err
	}
	var failed []string
	for _, v := range proof.Verdicts {
		if !v.Pass() {
			failed = append(failed, v.String())
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("contrast pairs fail at the ambient ceiling %s:\n%s", num(proof.Ceiling), strings.Join(failed, "\n"))
	}
	if proof.Ceiling != t.Ceiling {
		return fmt.Errorf("the ambient ceiling is %s, but DESIGN.md's ambient.ceiling records %s; update it, .impeccable/design.json and the Ambient Ceiling Rule", num(proof.Ceiling), num(t.Ceiling))
	}
	return nil
}

// over composites top over an opaque bottom in sRGB, as browsers do.
func over(top, bottom RGBA) RGBA {
	a := top.A
	return RGBA{top.R*a + bottom.R*(1-a), top.G*a + bottom.G*(1-a), top.B*a + bottom.B*(1-a), 1}
}

// luminance is WCAG 2.2's relative luminance.
func luminance(c RGBA) float64 {
	ch := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

func contrast(fg, bg RGBA) float64 {
	a, b := luminance(fg), luminance(bg)
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}
