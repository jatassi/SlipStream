package clients_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

var causesByState = map[clients.PollState][]clients.Cause{
	clients.NotResponding: {clients.NoResponse, clients.TimedOut, clients.TLSFailed, clients.ServerError},
	clients.SignInRefused: {clients.WrongCredentials, clients.WrongAPIKey, clients.Banned, clients.LockedOut},
	clients.NeedsAttention: {
		clients.VersionTooOld, clients.LabelPluginOff, clients.FolderUnreadable, clients.FolderInRoot,
		clients.AddressRefused, clients.HostRefused, clients.DaemonNeeded, clients.WrongDaemon,
		clients.NotKept, clients.MostMissing,
	},
}

func TestEveryCausePutsItsClientInTheSpecsPollState(t *testing.T) {
	for state, causes := range causesByState {
		for _, c := range causes {
			if got := c.State(); got != state {
				t.Errorf("%s.State() = %s, want %s", c, got, state)
			}
		}
	}
}

func TestOnlyACauseFoundByAPollKeepsPolling(t *testing.T) {
	for _, causes := range causesByState {
		for _, c := range causes {
			if got, want := c.Polls(), c == clients.NotKept || c == clients.MostMissing; got != want {
				t.Errorf("%s.Polls() = %v, want %v", c, got, want)
			}
		}
	}
}

func TestCauseFieldsMarkWhatTheAdminChanges(t *testing.T) {
	cases := map[clients.Cause][]clients.Field{
		clients.WrongCredentials: {clients.FieldUsername, clients.FieldPassword},
		clients.WrongAPIKey:      {clients.FieldAPIKey},
		clients.NoResponse:       {clients.FieldAddress},
		clients.LabelPluginOff:   {clients.FieldLabel},
		clients.FolderUnreadable: {clients.FieldFolder, clients.FieldFolders},
		clients.DaemonNeeded:     {clients.FieldDaemon},
		clients.Banned:           nil,
	}
	for c, want := range cases {
		if got := c.Fields(); !slices.Equal(got, want) {
			t.Errorf("%s.Fields() = %v, want %v", c, got, want)
		}
	}
}

func TestProblemErrorNamesItsFacts(t *testing.T) {
	old, err := clients.ParseVersion("4.5.2")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		p    clients.Problem
		want string
	}{
		{clients.Problem{Cause: clients.ServerError, Status: 503}, "503"},
		{clients.Problem{Cause: clients.VersionTooOld, Version: old}, "4.5.2"},
		{clients.Problem{Cause: clients.FolderUnreadable, Folder: `D:\Torrents`}, `D:\Torrents`},
		{clients.Problem{Cause: clients.HostRefused, Host: "transmission.lan"}, "transmission.lan"},
		{clients.Problem{Cause: clients.MostMissing, Missing: 14}, "14"},
	}
	for _, c := range cases {
		if got := c.p.Error(); !strings.Contains(got, c.want) {
			t.Errorf("%s's Error() = %q, want it to name %q", c.p.Cause, got, c.want)
		}
	}
	for _, causes := range causesByState {
		for _, c := range causes {
			msg := (&clients.Problem{Cause: c}).Error()
			if msg == "" || msg[0] < 'a' || msg[0] > 'z' || strings.HasSuffix(msg, ".") {
				t.Errorf("%s's Error() = %q, want a lowercase Go error string", c, msg)
			}
		}
	}
}
