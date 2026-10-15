package tokens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProve(t *testing.T) {
	theme, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := theme.Prove()
	if err != nil {
		t.Fatal(err)
	}
	if proof.Ceiling != 0.053 {
		t.Errorf("ceiling %v, want 0.053", proof.Ceiling)
	}
	if len(proof.Verdicts) != 32 {
		t.Fatalf("%d verdicts, want 32", len(proof.Verdicts))
	}
	for _, v := range proof.Verdicts {
		if !v.Pass() {
			t.Errorf("fails: %s", v)
		}
	}
	for i, want := range map[int]string{
		0:  "ok    4.51 >= 4.5  tertiary text on bare ambient                        worst at h240 s45",
		2:  "ok    5.35 >= 4.5  selected segment                                     worst at h60 s70",
		31: "ok   17.19 >= 4.5  ink on solid white (primary, Met)                    worst at h0 s14",
	} {
		if got := proof.Verdicts[i].String(); got != want {
			t.Errorf("verdict %d:\n got %s\nwant %s", i, got, want)
		}
	}
}

func TestCheckContrast(t *testing.T) {
	if err := CheckContrast(repo); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	doc := strings.NewReplacer(
		`ink-3: "hsl(214 14% 71%)"`, `ink-3: "hsl(214 14% 61%)"`,
		"| `ink-3` | `hsl(var(--h) 14% 71%)` |", "| `ink-3` | `hsl(var(--h) 14% 61%)` |",
	).Replace(design(t))
	if err := os.WriteFile(filepath.Join(root, designPath), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "the ambient ceiling is 0.015, but DESIGN.md's ambient.ceiling records 0.053; update it, .impeccable/design.json and the Ambient Ceiling Rule"
	if err := CheckContrast(root); err == nil || err.Error() != want {
		t.Errorf("darker ink-3: got %v, want %s", err, want)
	}
}
