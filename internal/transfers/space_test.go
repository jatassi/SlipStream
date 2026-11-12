package transfers

import (
	"slices"
	"testing"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/roots"
)

// sent is the positions nas is asked to receive of Transfer n.
func sent(j judgment, n byte) []uint32 {
	var out []uint32
	for _, a := range j.intents.of(nas).GetActive() {
		if a.GetTransferId()[15] == n {
			for _, f := range a.GetFiles() {
				out = append(out, f.GetPosition())
			}
		}
	}
	return out
}

// room leaves n bytes free on root's volume above its error level.
func (w *world) room(root roots.ID, n int64) *world {
	return w.dest(root, func(d *roots.Destination) {
		d.Volume.Free = n + w.s.levels.For(d.Node, d.Volume.Mount, d.Volume.Total).Error.Bytes
	})
}

func TestTenFilesWithRoomForFourSendFourThenTheRestAsRoomFrees(t *testing.T) {
	sizes := make([]int64, 10)
	for i := range sizes {
		sizes[i] = 25 * gb
	}
	w := newWorld().add(4, seedbox, onNas, sizes...).room(onNas, 100*gb)
	j := judge(w.s)
	if v, _ := j.view(tid(4)); v.State == NeedsSpace {
		t.Fatalf("the Transfer reads %s, want it sending what fits", state(v))
	}
	if got := sent(j, 4); !slices.Equal(got, []uint32{0, 1, 2, 3}) {
		t.Fatalf("nas receives %v with room for four files, want [0 1 2 3]", got)
	}

	for p := range 4 {
		w.file(4, p, func(f *File) { f.Placed = true })
	}
	w.room(onNas, 115*gb)
	if got := sent(judge(w.s), 4); !slices.Equal(got, []uint32{4, 5, 6, 7}) {
		t.Fatalf("nas receives %v once four are placed and 115 GB is free, want [4 5 6 7]", got)
	}

	w.receiving(nas, &tv1.Receiving{TransferId: tid(4).Bytes(), VerifiedBytes: 30 * gb})
	if got := sent(judge(w.s), 4); !slices.Equal(got, []uint32{4, 5, 6, 7, 8}) {
		t.Fatalf("nas receives %v once 30 GB of them verified, want [4 5 6 7 8]", got)
	}

	w.room(onNas, 10*gb)
	if got := sent(judge(w.s), 4); !slices.Equal(got, []uint32{4}) {
		t.Fatalf("nas receives %v with 10 GB free, want only [4], already verified", got)
	}
}

func TestATransferWhoseFirstFileDoesntFitNeedsWhatThatFileLacks(t *testing.T) {
	w := newWorld().add(1, seedbox, onNas, 25*gb, gb).room(onNas, 10*gb)
	j := judge(w.s)
	if v, _ := j.view(tid(1)); v.State != NeedsSpace || v.Space.Need != 15*gb {
		t.Errorf("with 10 GB free the Transfer reads %s, want it to need 15 GB", state(v))
	}
	if got := sent(j, 1); got != nil {
		t.Errorf("nas receives %v, want nothing before its first file fits", got)
	}
}
