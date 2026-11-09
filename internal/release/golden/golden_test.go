package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper()             {}
func (r *recorder) Logf(string, ...any) {}
func (r *recorder) Errorf(format string, a ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, a...))
}

func (r *recorder) Fatal(a ...any) { panic(fmt.Sprint(a...)) }

func (r *recorder) Fatalf(format string, a ...any) { panic(fmt.Sprintf(format, a...)) }

func parity(odd bool) Check {
	return func(r Row) error {
		if n := r.Int("n"); (n%2 == 1) != odd {
			return fmt.Errorf("%d has the other parity", n)
		}
		return nil
	}
}

var checks = map[string]Check{"should_be_even": parity(false), "should_be_odd": parity(true)}

// fixture copies testdata/Fixture.json beside the pending and overrides
// lists given, and runs it.
func fixture(t *testing.T, pending, overrides string, f Fixture) (rec *recorder, dir string) {
	t.Helper()
	dir = t.TempDir()
	b, err := os.ReadFile("testdata/Fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"Fixture.json": string(b), "Fixture.pending.json": pending, "Fixture.overrides.json": overrides} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.Path = filepath.Join(dir, "Fixture.json")
	rec = &recorder{TB: t}
	Run(rec, f)
	return rec, dir
}

func failsWith(t *testing.T, rec *recorder, wants ...string) {
	t.Helper()
	if len(rec.errors) != len(wants) {
		t.Fatalf("the run failed with %q, want %d failures", rec.errors, len(wants))
	}
	for _, want := range wants {
		if !slices.ContainsFunc(rec.errors, func(e string) bool { return strings.Contains(e, want) }) {
			t.Errorf("the run failed with %q, want one naming %q", rec.errors, want)
		}
	}
}

const bothOddPending = `[{"method":"should_be_even","args":[3]},{"method":"should_be_even","args":[5]}]`

func TestPendingRowsThatFailPass(t *testing.T) {
	rec, _ := fixture(t, bothOddPending, "[]", Fixture{Checks: checks})
	failsWith(t, rec)
}

func TestARowThatFailsAndIsNotPendingFails(t *testing.T) {
	rec, _ := fixture(t, `[{"method":"should_be_even","args":[3]}]`, "[]", Fixture{Checks: checks})
	failsWith(t, rec, `should_be_even [5]: 5 has the other parity`)
}

func TestAPendingRowThatPassesFails(t *testing.T) {
	pending := `[{"method":"should_be_even","args":[2]},{"method":"should_be_even","args":[3]},{"method":"should_be_even","args":[5]}]`
	rec, _ := fixture(t, pending, "[]", Fixture{Checks: checks})
	failsWith(t, rec, "should_be_even [2] passes now")
}

func TestAMethodWithNoCheckFails(t *testing.T) {
	rec, _ := fixture(t, bothOddPending, "[]", Fixture{Checks: map[string]Check{"should_be_even": parity(false)}})
	failsWith(t, rec, "Upstream method should_be_odd")
}

func TestAnUnportedMethodNeedsNoCheck(t *testing.T) {
	rec, _ := fixture(t, bothOddPending, "[]", Fixture{
		Checks:   map[string]Check{"should_be_even": parity(false)},
		Unported: map[string]string{"should_be_odd": "SlipStream reads no parity."},
	})
	failsWith(t, rec)
}

func TestACheckForNoMethodFails(t *testing.T) {
	more := map[string]Check{"should_be_prime": parity(true)}
	for k, v := range checks {
		more[k] = v
	}
	rec, _ := fixture(t, bothOddPending, "[]", Fixture{Checks: more})
	failsWith(t, rec, "should_be_prime has a check")
}

func TestAPendingRowTheFixtureLacksFails(t *testing.T) {
	pending := `[{"method":"should_be_even","args":[3]},{"method":"should_be_even","args":[5]},{"method":"should_be_even","args":[7]}]`
	rec, _ := fixture(t, pending, "[]", Fixture{Checks: checks})
	failsWith(t, rec, "should_be_even [7] is pending")
}

func TestAnOverrideRunsTheRowWithSlipStreamsArguments(t *testing.T) {
	overrides := `[{"method":"should_be_even","args":[5],"ours":[6],"reason":"SlipStream reads it as 6."}]`
	rec, _ := fixture(t, `[{"method":"should_be_even","args":[3]}]`, overrides, Fixture{Checks: checks})
	failsWith(t, rec)
}

func TestAnOverrideCarriesSlipStreamsReading(t *testing.T) {
	even := map[string]Check{
		"should_be_even": func(r Row) error {
			odd := false
			r.Reads(&odd)
			if n := r.Int("n"); (n%2 == 1) != odd {
				return fmt.Errorf("%d has the other parity", n)
			}
			return nil
		},
		"should_be_odd": parity(true),
	}
	overrides := `[{"method":"should_be_even","args":[3],"reads":true,"reason":"SlipStream reads 3 as odd."}]`
	rec, _ := fixture(t, `[{"method":"should_be_even","args":[5]}]`, overrides, Fixture{Checks: even})
	failsWith(t, rec)
}

func TestUpdatePendingWritesTheFailingRows(t *testing.T) {
	*updatePending = true
	t.Cleanup(func() { *updatePending = false })
	rec, dir := fixture(t, `[{"method":"should_be_even","args":[2]}]`, "[]", Fixture{Checks: checks})
	failsWith(t, rec)
	b, err := os.ReadFile(filepath.Join(dir, "Fixture.pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n  {\"method\":\"should_be_even\",\"args\":[3]},\n  {\"method\":\"should_be_even\",\"args\":[5]}\n]\n"
	if string(b) != want {
		t.Errorf("the pending list reads\n%s\nwant\n%s", b, want)
	}
}
