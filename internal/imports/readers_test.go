package imports

import "testing"

// TestAnImportWaitsOnlyWhileEveryFileAtWorkWaitsForTheAdmin raises its
// Grab's Use to waiting only while a file waits for the Admin, or is
// offered after three probe timeouts, and no file reads, sends or places.
func TestAnImportWaitsOnlyWhileEveryFileAtWorkWaitsForTheAdmin(t *testing.T) {
	waiting := File{Outcome: Waiting, Reason: &Reason{Code: Confirm}}
	offered := File{Outcome: Pending, Hold: &Hold{Code: ProbeTimeout, Attempts: 3}}
	for _, c := range []struct {
		name  string
		files []File
		want  bool
	}{
		{"a file waits beside a placed one", []File{waiting, {Outcome: Placed}, {Outcome: LeftOut}}, true},
		{"a probe timed out three times", []File{offered}, true},
		{"a probe timed out twice", []File{{Outcome: Pending, Hold: &Hold{Code: ProbeTimeout, Attempts: 2}}}, false},
		{"another file still reads", []File{waiting, {Outcome: Pending}}, false},
		{"another file sends", []File{waiting, {Outcome: Sending}}, false},
		{"another file places", []File{waiting, {Outcome: Placing}}, false},
		{"nothing waits", []File{{Outcome: Placed}, {Outcome: Declined}}, false},
	} {
		if got := (Import{Files: c.files}).waits(); got != c.want {
			t.Errorf("%s: waits = %v, want %v", c.name, got, c.want)
		}
	}
}
