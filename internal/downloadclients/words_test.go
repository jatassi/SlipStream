package downloadclients_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

var allCauses = []tc.Cause{
	tc.NoResponse, tc.TimedOut, tc.TLSFailed, tc.ServerError,
	tc.WrongCredentials, tc.WrongAPIKey, tc.Banned, tc.LockedOut,
	tc.VersionTooOld, tc.LabelPluginOff, tc.FolderUnreadable, tc.FolderInRoot, tc.AddressRefused, tc.HostRefused,
	tc.DaemonNeeded, tc.WrongDaemon, tc.NotKept, tc.MostMissing,
}

func subject(t *testing.T, name string, kind tc.Kind, address string) dc.Subject {
	t.Helper()
	a, err := tc.ParseAddress(address, kind)
	if err != nil {
		t.Fatal(err)
	}
	return dc.Subject{Name: name, Kind: kind, Address: a}
}

func version(t *testing.T, s string) tc.Version {
	t.Helper()
	v, err := tc.ParseVersion(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// machineWritten matches what WRITING.md keeps out of a message: courtesy
// words, dashes and semicolons joining clauses, and a clause spliced on
// with ", so".
var machineWritten = regexp.MustCompile(`(?i)\bplease\b|\bsorry\b|[;—–]|, so\b|\binvalid\b`)

// readsAsWritten fails unless s is whole sentences that say what happened
// and then what to do, with none of WRITING.md's machine-written patterns.
func readsAsWritten(t *testing.T, what, s string) {
	t.Helper()
	if !strings.HasSuffix(s, ".") || strings.Count(s, ". ") < 1 || machineWritten.MatchString(s) {
		t.Errorf("%s reads %q, want what happened, then what to do, in sentences WRITING.md allows", what, s)
	}
}

func TestEveryProblemHasASentenceNamingTheClientOrItsNode(t *testing.T) {
	for _, kind := range tc.Kinds {
		s := subject(t, "Living room", kind, "http://192.168.1.20:9091")
		for _, c := range allCauses {
			p := tc.Problem{Cause: c, Status: 503, Version: version(t, "1.0"), Folder: "/downloads", Host: "client.lan", Missing: 14}
			got := dc.Says(p, s, "Media server")
			readsAsWritten(t, string(kind)+" "+string(c), got)
			if !strings.Contains(got, "Living room") && !strings.Contains(got, "Media server") {
				t.Errorf("%s %s reads %q, want it to name the client or its Node", kind, c, got)
			}
		}
	}
}

func TestEveryWarningAndRefusalHasASentence(t *testing.T) {
	s := subject(t, "Seedbox", tc.RTorrent, "http://192.168.1.20")
	for _, w := range []tc.Warning{tc.RemovesAtLimit, tc.MovesAfterFinish, tc.NoMovingState} {
		got := dc.Warns(w, s)
		readsAsWritten(t, "warning", got)
		if !strings.Contains(got, "Seedbox") {
			t.Errorf("warning %d reads %q, want it to name the client", w, got)
		}
	}
	for _, c := range []struct {
		field tc.Field
		err   error
	}{
		{tc.FieldAddress, tc.ErrAddressScheme}, {tc.FieldAddress, tc.ErrAddressNotURL}, {tc.FieldAddress, tc.ErrAddressHost},
		{tc.FieldAddress, tc.ErrAddressPort}, {tc.FieldAddress, tc.ErrSignInInAddress}, {tc.FieldAddress, tc.ErrQueryInAddress},
		{tc.FieldLabel, tc.ErrLabelEmpty}, {tc.FieldLabel, tc.ErrLabelControl}, {tc.FieldLabel, tc.ErrLabelComma}, {tc.FieldLabel, tc.ErrLabelWord},
		{tc.FieldFolders, &tc.FolderError{Index: 1, Err: tc.ErrClientFolderNotAbsolute}},
		{tc.FieldFolders, &tc.FolderError{Index: 1, Err: tc.ErrNodeFolderNotAbsolute}},
		{tc.FieldFolders, &tc.FolderError{Index: 1, Err: tc.ErrClientFolderTwice}},
		{tc.FieldUsername, dc.ErrNotTaken}, {tc.FieldAPIKey, dc.ErrNotTaken}, {tc.FieldDaemon, dc.ErrNotTaken}, {tc.FieldPassword, dc.ErrNeeded},
	} {
		got := dc.Refused(&dc.RefusedError{Field: c.field, Err: c.err}, tc.Deluge)
		if !strings.HasSuffix(got, ".") || machineWritten.MatchString(got) || got[0] < 'A' || got[0] > 'Z' {
			t.Errorf("refusing %s with %v reads %q, want a sentence WRITING.md allows", c.field, c.err, got)
		}
	}
	for _, got := range []string{
		dc.Unreached("Media server"), dc.InUse("Media server"), dc.HasTorrents("Transmission", 12, false), dc.HasTorrents("Transmission", 12, true),
		dc.GrabsToRefused(&dc.GrabsToError{Name: "Transmission", Off: true}, "Media server"),
		dc.GrabsToRefused(&dc.GrabsToError{Name: "Transmission"}, "Media server"),
	} {
		readsAsWritten(t, "refusal", got)
	}
	for _, n := range []int{1, 2} {
		kept, moving := dc.HasTorrents("Transmission", n, false), dc.HasTorrents("Transmission", n, true)
		for _, got := range []string{kept, moving} {
			if !strings.Contains(got, "Transmission") || !strings.Contains(got, strconv.Itoa(n)+" torrent") {
				t.Errorf("%d held torrents read %q, want the client and the count", n, got)
			}
		}
		if kept == moving {
			t.Errorf("a delete and a move with %d torrents both read %q, want each to say what blocks it", n, kept)
		}
	}
}
