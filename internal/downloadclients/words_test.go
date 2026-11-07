package downloadclients_test

import (
	"regexp"
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

func TestTheSpecsSentencesReadAsTheSpecWritesThem(t *testing.T) {
	transmission := subject(t, "Transmission", tc.Transmission, "http://192.168.1.20:9091")
	qbittorrent := subject(t, "qBittorrent", tc.QBittorrent, "http://192.168.1.20:8080")
	deluge := subject(t, "Deluge", tc.Deluge, "http://192.168.1.20:8112")
	for _, c := range []struct {
		p    tc.Problem
		s    dc.Subject
		want string
	}{
		{tc.Problem{Cause: tc.WrongCredentials}, transmission, "Transmission rejected the username or password. Check them and test again."},
		{tc.Problem{Cause: tc.WrongCredentials}, deluge, "Deluge rejected the password. Check it and test again."},
		{tc.Problem{Cause: tc.WrongAPIKey}, qbittorrent, "qBittorrent rejected the API key. Replace it and test again."},
		{tc.Problem{Cause: tc.NoResponse}, transmission, "Transmission isn't responding. Check that it's running and that Media server can reach 192.168.1.20:9091."},
		{tc.Problem{Cause: tc.TimedOut}, transmission, "Transmission isn't responding. Check that it's running and that Media server can reach 192.168.1.20:9091."},
		{tc.Problem{Cause: tc.Banned}, qbittorrent, "qBittorrent banned Media server's address after too many failed sign-ins. Wait an hour or restart qBittorrent, then select Check now."},
		{tc.Problem{Cause: tc.AddressRefused}, transmission, "Transmission refused Media server's address. Add it to rpc-whitelist in Transmission's settings, then select Check now."},
		{
			tc.Problem{Cause: tc.HostRefused, Host: "transmission.lan"}, transmission,
			"Transmission refused the host name transmission.lan. Use its IP address, or add the name to rpc-host-whitelist in Transmission's settings.",
		},
		{
			tc.Problem{Cause: tc.VersionTooOld, Version: version(t, "4.5.2")}, qbittorrent,
			"Media server reached qBittorrent 4.5.2, which is older than SlipStream supports. Update it to 4.6 or later and test again.",
		},
		{tc.Problem{Cause: tc.LabelPluginOff}, deluge, "Deluge's Label plugin is off. Turn it on in Deluge's preferences and test again."},
		{
			tc.Problem{Cause: tc.FolderUnreadable, Folder: "/downloads"}, transmission,
			"Media server can't open /downloads, the folder Transmission saves to. Under Client folders, pair /downloads with the folder Media server sees.",
		},
		{
			tc.Problem{Cause: tc.FolderUnreadable, Folder: `D:\Torrents`, Paired: "/downloads"}, transmission,
			`Media server can't open D:\Torrents, which is paired with /downloads. Check the pair, and check that Media server can read that folder.`,
		},
		{tc.Problem{Cause: tc.NotKept}, transmission, "Transmission didn't keep the torrent SlipStream sent it. Check Transmission's log for the reason."},
		{
			tc.Problem{Cause: tc.MostMissing, Missing: 14}, transmission,
			"Transmission no longer lists 14 of SlipStream's torrents. Check that the address points at the right Transmission.",
		},
		{tc.Problem{Cause: tc.DaemonNeeded}, deluge, "Deluge's web UI lists more than one daemon. Choose the daemon SlipStream uses and test again."},
	} {
		if got := dc.Says(c.p, c.s, "Media server"); got != c.want {
			t.Errorf("%s reads %q, want %q", c.p.Cause, got, c.want)
		}
	}
	if got, want := dc.Stopped("Transmission"), "Transmission rejected the saved password. SlipStream stopped checking it to avoid a lockout. "+
		"Change the password, or select Check now once it's fixed."; got != want {
		t.Errorf("Stopped reads %q, want %q", got, want)
	}
	if got, want := dc.LocalError("Transmission", "No space left on device"),
		`Transmission stopped this torrent with the error "No space left on device". Fix the cause in Transmission and the download carries on.`; got != want {
		t.Errorf("LocalError reads %q, want %q", got, want)
	}
	if got, want := dc.Warns(tc.RemovesAtLimit, qbittorrent), "qBittorrent removes torrents that reach a seeding limit. A download could be removed before "+
		"SlipStream imports it. To keep downloads until they're imported, set qBittorrent to stop torrents at the limit."; got != want {
		t.Errorf("RemovesAtLimit reads %q, want %q", got, want)
	}
	if got, want := dc.Warns(tc.MovesAfterFinish, deluge), "Deluge moves finished downloads to another folder. Set an Import delay so SlipStream imports after the move."; got != want {
		t.Errorf("MovesAfterFinish reads %q, want %q", got, want)
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
}
