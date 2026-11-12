package transfers_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/bencode"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// episode is a one-MiB file of torrent 1, in group g.
func episode(n int, g string) transfers.FileRequest {
	return transfers.FileRequest{Group: g, Name: fmt.Sprintf("Episode %d.mkv", n), Source: source.Torrent{Hash: bencode.InfoHash{1}, File: n}, Size: 1 << 20}
}

// want asks for exactly files of job from seedbox into home's root.
func (r *cluster) want(t *testing.T, job string, files ...transfers.FileRequest) transfers.ID {
	t.Helper()
	var id transfers.ID
	r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, err = r.tr.Want(ctx, tx, transfers.Request{Purpose: transfers.Grab, Job: job, Source: r.seedbox, Destination: transfers.Root{ID: r.root}, Files: files})
		return err
	})
	return id
}

// files is v's files, "0 Episode 1.mkv" each.
func files(v transfers.View) []string {
	var out []string
	for _, f := range v.Files {
		out = append(out, fmt.Sprintf("%d %s", f.Position, f.Name))
	}
	return out
}

func TestWantingTheSameFilesAgainChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.want(t, "pack", episode(1, "a"), episode(2, "b"))
		before := r.view(t, id)
		r.takeTouched()
		if again := r.want(t, "pack", episode(1, "a"), episode(2, "b")); again != id {
			t.Fatalf("wanting the same files again recorded %s, want %s", again, id)
		}
		after := r.view(t, id)
		check(t, "files", files(after), files(before))
		check(t, "files", files(after), []string{"0 Episode 1.mkv", "1 Episode 2.mkv"})
		check(t, "home runs", r.homeEnd.actives(), []string{"receive 0 1"})
		check(t, "Nodes touched", r.takeTouched(), nil)
	})
}

func TestADroppedFileWantedAgainIsSentAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.want(t, "pack", episode(1, "a"), episode(2, "b"), episode(3, "c"))
		r.homeEnd.report(t, verifiedResult(id, 1, 1))
		r.want(t, "pack", episode(1, "a"), episode(3, "c"))
		check(t, "files once episode 2 is dropped", files(r.view(t, id)), []string{"0 Episode 1.mkv", "2 Episode 3.mkv"})
		check(t, "home runs", r.homeEnd.actives(), []string{"receive 0 2"})

		r.want(t, "pack", episode(1, "a"), episode(2, "b"), episode(3, "c"))
		v := r.view(t, id)
		check(t, "files once episode 2 is wanted again", files(v), []string{"0 Episode 1.mkv", "2 Episode 3.mkv", "3 Episode 2.mkv"})
		check(t, "home runs", r.homeEnd.actives(), []string{"receive 0 2 3"})
		if v.Verified != 0 {
			t.Errorf("the Transfer reads %d bytes verified, want none: the file wanted again starts over", v.Verified)
		}
		r.homeEnd.report(t, verifiedResult(id, 3, 1))
		verified, _, _ := r.purpose.seen()
		check(t, "Verified", verified, []string{"pack b", "pack b"})
	})
}

func TestTheRecordGoesWhenNothingIsLeftToSend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.want(t, "pack", episode(1, "a"), episode(2, "b"))
		r.homeEnd.report(t, verifiedResult(id, 0, 1))
		r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.tr.Placed(ctx, tx, id, 0) })
		if got := r.want(t, "pack", episode(1, "a")); !got.IsZero() {
			t.Errorf("Want returned %s once only a placed file is left, want no Transfer", got)
		}
		if _, err := r.tr.Get(t.Context(), id); !errors.Is(err, transfers.ErrNotFound) {
			t.Fatalf("the Transfer reads %v, want it gone", err)
		}
		if in, _ := r.homeEnd.now(); len(in.GetKeep()) != 0 {
			t.Errorf("home keeps %v, want nothing", in.GetKeep())
		}
	})
}

func TestAPathSourceIsSentByItsPath(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		at := source.Path{Abs: "/downloads/Heat (1995)/Heat.mkv"}
		id := r.want(t, "folder", transfers.FileRequest{Name: "Heat.mkv", Source: at, Size: 1 << 20})
		if got := r.view(t, id).Files[0].Source; got != at {
			t.Errorf("the recorded source is %v, want %v", got, at)
		}
		in, _ := r.seedboxEnd.now()
		if got := in.GetActive()[0].GetFiles()[0].GetSource().GetPath().GetAbs(); got != at.Abs {
			t.Errorf("seedbox is asked to send %q, want %q", got, at.Abs)
		}
		r.homeEnd.report(t, failed(id, tv1.Failed_CODE_SOURCE_FAILED, "permission denied"))
		_, f, _ := r.purpose.seen()
		check(t, "Failed", f, []string{"folder: seedbox can't read Heat.mkv: permission denied."})
	})
}

func TestReservedInCountsWhatLiveTransfersStillSend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.want(t, "pack", episode(1, "a"), episode(2, "b"), episode(3, "c"))
		r.homeEnd.report(t, verifiedResult(id, 0, 1))
		r.want(t, "pack", episode(1, "a"), episode(2, "b"))
		r.homeEnd.facts(&tv1.Facts{Receiving: []*tv1.Receiving{{TransferId: id.Bytes(), VerifiedBytes: 1 << 18}}})
		reserved := func(volume string) int64 {
			t.Helper()
			var n int64
			err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
				var err error
				n, err = r.tr.ReservedIn(ctx, rd, r.home, volume)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			return n
		}
		if got, want := reserved("v1"), int64(1<<20-1<<18); got != want {
			t.Errorf("ReservedIn = %d, want %d: episode 2 less what home verified of it", got, want)
		}
		if got := reserved("v2"); got != 0 {
			t.Errorf("ReservedIn on another volume = %d, want 0", got)
		}
	})
}
