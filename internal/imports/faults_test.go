package imports

import (
	"strings"
	"testing"
)

func leftOut(codes ...ReasonCode) []File {
	out := make([]File, len(codes))
	for i, c := range codes {
		out[i] = File{Position: i, Outcome: LeftOut, Reason: &Reason{Code: c}}
	}
	return out
}

// TestEveryReleaseFaultRow runs each row of the spec's release-fault table
// that faults owns. The Blocklist causes, the import waits and the
// Transfer's faults are each their own step's.
func TestEveryReleaseFaultRow(t *testing.T) {
	_, table, _ := specReasons(t)
	unreadable := []File{{Outcome: FailedOut, Reason: &Reason{Code: Unreadable}}, {Position: 1, Outcome: FailedOut, Reason: &Reason{Code: Unreadable}}}
	cases := map[string]faultCase{
		"No video file":                     {files: leftOut(Sample, Bonus, HiddenFolder), failure: NoVideoFile},
		"ffprobe can't read any video":      {files: unreadable, failure: UnreadableFile},
		"Videos left out only by the Admin": {files: leftOut(TooSmall, Sample), waits: SmallestVideo.Key},
	}
	ran := 0
	for _, row := range table {
		for prefix, c := range cases {
			if strings.HasPrefix(row[0], prefix) {
				ran++
				c.check(t, prefix, row[1])
			}
		}
	}
	if ran != len(cases) {
		t.Errorf("ran %d of the %d rows faults owns: the fixture lost one", ran, len(cases))
	}
}

// faultCase is a release-fault row's files and what faults makes of them:
// the Import's failure, or the setting its first file waits for.
type faultCase struct {
	files   []File
	failure FailureCode
	waits   string
}

func (c faultCase) check(t *testing.T, prefix, want string) {
	t.Helper()
	im := Import{Source: FromGrab, Files: c.files}
	faults(&im, Rules{Smallest: 50 << 20, Sample: 25})
	got := FailureCode("")
	if im.Failure != nil {
		got = *im.Failure
	}
	if got != c.failure {
		t.Errorf("%s: fails %q, want %q (%s)", prefix, got, c.failure, want)
	}
	if c.waits == "" {
		return
	}
	if f := im.Files[0]; f.Outcome != Waiting || f.Reason.Code != Settings || f.Reason.Setting != c.waits {
		t.Errorf("%s: the file is %s %+v, want waiting settings for %s (%s)", prefix, f.Outcome, f.Reason, c.waits, want)
	}
	if im.Files[1].Outcome != LeftOut {
		t.Errorf("%s: a sample by name is %s, want still left out", prefix, im.Files[1].Outcome)
	}
}

func TestAnImportWithAFileStillToPlaceHasNoFault(t *testing.T) {
	im := Import{Source: FromGrab, Files: append(leftOut(Sample), File{Position: 1, Outcome: Sending})}
	faults(&im, Rules{})
	if im.Failure != nil {
		t.Errorf("an Import with a file sending fails %s, want no fault yet", *im.Failure)
	}
}
