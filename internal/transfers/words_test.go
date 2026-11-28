package transfers

import "testing"

func TestReasons(t *testing.T) {
	heat := Transfer{Source: seedbox, Files: []File{{Name: "Heat (1995).mkv"}}}
	cases := []struct {
		view View
		dest Destination
		want string
	}{
		{View{State: Queued, Ahead: 1}, nil, "Queued behind 1 other Transfer from seedbox."},
		{View{State: Queued, Ahead: 2}, nil, "Queued behind 2 other Transfers from seedbox."},
		{View{State: Waiting, Wait: &Wait{Kind: NodeAway, Node: seedbox}}, nil, "Waiting for seedbox to come back."},
		{View{State: Waiting, Wait: &Wait{Kind: NodeOtherBuild, Node: nas}}, nil, "Waiting for nas to update."},
		{View{State: Waiting, Wait: &Wait{Kind: Connecting, Node: seedbox}}, nil, "Connecting to seedbox."},
		{View{State: Waiting, Wait: &Wait{Kind: NoRoute, Node: nas}}, nil, "seedbox and nas can't connect to each other. Open the port in link.listen on one of them."},
		{View{State: Waiting, Wait: &Wait{Kind: RootNotReady, Node: nas}}, nil, "Waiting for the library root on nas."},
		{View{State: Waiting, Wait: &Wait{Kind: StorageDraining, Node: nas}}, nil, "Waiting while nas finishes its Storage work."},
		{View{State: Waiting, Wait: &Wait{Kind: SourceChecking, Node: seedbox, Name: "Heat (1995)"}}, nil, "Waiting for seedbox to check Heat (1995)."},
		{View{State: Waiting, Wait: &Wait{Kind: SourceNotReady, Node: seedbox, Name: "Heat (1995).mkv"}}, nil, "Waiting for seedbox to open Heat (1995).mkv."},
		{View{State: NeedsSpace, Space: &Space{Node: nas, Volume: "Media", Need: 12 * gb}}, nil, "Waiting for 12 GB free on nas (Media)."},
		{View{State: Waiting, Wait: &Wait{Kind: FolderNotReady, Node: nas}}, Scratch{Node: nas, Name: "in/s-7.mkv"}, "Waiting for the scratch folder on nas."},
		{View{State: Copying}, nil, ""},
	}
	for _, c := range cases {
		c.view.Transfer, c.view.DestNode = heat, nas
		c.view.Destination = c.dest
		if got := c.view.Reason(names); got != c.want {
			t.Errorf("Reason = %q, want %q", got, c.want)
		}
	}
}

func TestFailureSentences(t *testing.T) {
	heat := Transfer{Source: seedbox, Files: []File{{Name: "Heat (1995).mkv"}}}
	cases := []struct {
		dest     Destination
		code     Code
		position int
		detail   string
		want     string
	}{
		{nil, SourceGone, 0, "", "seedbox no longer has Heat (1995).mkv."},
		{nil, SourceFailed, 0, "file not found", "seedbox's torrent client couldn't read Heat (1995).mkv: file not found."},
		{nil, Damaged, 0, "", "Heat (1995).mkv kept arriving damaged from seedbox. Check seedbox's memory."},
		{nil, Damaged, 0, readBack, "Heat (1995).mkv was damaged each time nas read it back from disk. Check nas's disk."},
		{nil, SourceWrittenOff, 0, "", "SlipStream gave up seedbox's work before Heat (1995).mkv was copied."},
		{nil, SourceGone, -1, "", "seedbox no longer has the files."},
		{nil, Rerouted, -1, "", "The library root moved to seedbox, which already has these files. Nothing needs copying."},
		{Scratch{Node: nas, Name: "in/s-7.mkv"}, FolderFailed, 0, "rename: access denied", "nas couldn't move Heat (1995).mkv into its scratch folder. Check that SlipStream can write to the folder."},
	}
	for _, c := range cases {
		heat.Destination = c.dest
		if got := sentence(heat, nas, c.code, c.position, c.detail, names); got != c.want {
			t.Errorf("%s = %q, want %q", c.code, got, c.want)
		}
	}
}
