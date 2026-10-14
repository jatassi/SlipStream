package main_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBrowserWritesEachShellsConfig(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })

	desk := filepath.Join(runDir, "browser", "desk", "playwright-cli.json")
	phone := filepath.Join(runDir, "browser", "phone", "playwright-cli.json")
	if got, want := fakeworld(t, "browser", runDir), (result{0, desk + "\n" + phone + "\n", ""}); got != want {
		t.Fatalf("browser = %+v, want %+v", got, want)
	}
	config := func(width, height int, scale float64, mobile bool) string {
		return fmt.Sprintf(`{
  "browser": {
    "browserName": "chromium",
    "launchOptions": {
      "args": [
        "--host-resolver-rules=MAP home.localhost %s"
      ]
    },
    "contextOptions": {
      "viewport": {
        "width": %d,
        "height": %d
      },
      "deviceScaleFactor": %v,
      "isMobile": %t,
      "hasTouch": %t
    }
  },
  "outputDir": %s
}
`, m.Nodes[0].Address, width, height, scale, mobile, mobile, jsonText(t, filepath.Join(runDir, "browser", "output")))
	}
	if got, want := readFile(t, desk), config(1440, 900, 1, false); got != want {
		t.Errorf("desk's config =\n%s\nwant\n%s", got, want)
	}
	if got, want := readFile(t, phone), config(390, 844, 3, true); got != want {
		t.Errorf("phone's config =\n%s\nwant\n%s", got, want)
	}
	want := []string{
		`step.start {"n":1,"verb":"browser"}`,
		`step.end {"n":1,"exit":0} map[desk:browser/desk/playwright-cli.json phone:browser/phone/playwright-cli.json]`,
	}
	if got := journal(t, runDir); !reflect.DeepEqual(got, want) {
		t.Errorf("journal = %q, want %q", got, want)
	}
}
