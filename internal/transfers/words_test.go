package transfers

import "testing"

func TestReasons(t *testing.T) {
	heat := Transfer{Source: seedbox, Files: []File{{Name: "Heat (1995).mkv"}}}
	cases := []struct {
		view View
		want string
	}{
		{View{State: Queued, Ahead: 1}, "Queued behind 1 other Transfer from seedbox."},
		{View{State: Queued, Ahead: 2}, "Queued behind 2 other Transfers from seedbox."},
		{View{State: Waiting, Wait: &Wait{Kind: NodeAway, Node: seedbox}}, "Waiting for seedbox to come back."},
		{View{State: Waiting, Wait: &Wait{Kind: NodeOtherBuild, Node: nas}}, "Waiting for nas to update."},
		{View{State: Waiting, Wait: &Wait{Kind: Connecting, Node: seedbox}}, "Connecting to seedbox."},
		{View{State: Waiting, Wait: &Wait{Kind: NoRoute, Node: nas}}, "seedbox and nas can't reach each other. Open the link port on one of them, or add a Relay Node."},
		{View{State: Waiting, Wait: &Wait{Kind: RootNotReady, Node: nas}}, "Waiting for the library root on nas."},
		{View{State: Waiting, Wait: &Wait{Kind: StorageDraining, Node: nas}}, "Waiting while nas drains Storage."},
		{View{State: Waiting, Wait: &Wait{Kind: SourceChecking, Node: seedbox, Name: "Heat (1995)"}}, "Waiting for seedbox to check Heat (1995)."},
		{View{State: Waiting, Wait: &Wait{Kind: SourceNotReady, Node: seedbox, Name: "Heat (1995).mkv"}}, "Waiting for seedbox to open Heat (1995).mkv."},
		{View{State: NeedsSpace, Space: &Space{Node: nas, Volume: "Media", Need: 12 * gb}}, "Waiting for 12 GB free on nas (Media)."},
		{View{State: Copying}, ""},
	}
	for _, c := range cases {
		c.view.Transfer, c.view.DestNode = heat, nas
		if got := c.view.Reason(names); got != c.want {
			t.Errorf("Reason = %q, want %q", got, c.want)
		}
	}
}

func TestFailureSentences(t *testing.T) {
	heat := Transfer{Source: seedbox, Files: []File{{Name: "Heat (1995).mkv"}}}
	cases := []struct {
		code     Code
		position int
		detail   string
		want     string
	}{
		{SourceGone, 0, "", "seedbox no longer has Heat (1995).mkv."},
		{SourceFailed, 0, "file not found", "seedbox's torrent client failed Heat (1995).mkv: file not found."},
		{Damaged, 0, "", "Heat (1995).mkv kept arriving damaged from seedbox. Check seedbox's memory."},
		{Damaged, 0, readBack, "Heat (1995).mkv kept reading back damaged on nas. Check nas's disk."},
		{SourceWrittenOff, 0, "", "seedbox was written off before Heat (1995).mkv was copied."},
		{SourceGone, -1, "", "seedbox no longer has the files."},
		{Rerouted, -1, "", "The library root is on seedbox now, which already has the files."},
	}
	for _, c := range cases {
		if got := sentence(heat, nas, c.code, c.position, c.detail, names); got != c.want {
			t.Errorf("%s = %q, want %q", c.code, got, c.want)
		}
	}
}
