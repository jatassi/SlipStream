package transfers_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// toScratch gives both ends an intent that runs tr on route 1 into name
// under the receiver's scratch folder.
func (tr *transfer) toScratch(name string) {
	recv := tr.active(tv1.Side_SIDE_RECEIVE, tv1.Route_ROUTE_RECEIVER_DIALS)
	recv.Destination = &tv1.Destination{Kind: &tv1.Destination_Scratch{Scratch: name}}
	tr.sender.h.set(&tv1.Intent{Active: []*tv1.Active{tr.active(tv1.Side_SIDE_SEND, tv1.Route_ROUTE_RECEIVER_DIALS)}}, role.Live)
	tr.receiver.h.set(&tv1.Intent{Active: []*tv1.Active{recv}, Keep: []*tv1.Kept{tr.kept()}}, role.Live)
}

func TestASenderOpensAScratchSourceOnlyUnderItsScratchFolder(t *testing.T) {
	cases := []struct {
		name    string
		arrives bool
	}{
		{name: "out/c-1-1.mkv", arrives: true},
		{name: "../payload/0.mkv"},
		{name: "out/../../payload/0.mkv"},
		{name: "/payload/0.mkv"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				_, w, _ := trio(t)
				tr := newTransfer(t, w[0], w[1])
				made := filepath.Join(w[0].scratch(), "out", "c-1-1.mkv")
				if err := os.MkdirAll(filepath.Dir(made), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.Link(w[0].source(0), made); err != nil {
					t.Fatal(err)
				}
				tr.files[0].Source = &tv1.Source{Ref: &tv1.Source_Scratch{Scratch: c.name}}
				w[0].start()
				w[1].start()
				tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
				if c.arrives {
					tr.expectArrived()
					return
				}
				if f := w[1].h.awaitFailed(t, tr.id); f.GetCode() != tv1.Failed_CODE_SOURCE_GONE || f.GetPosition() != 0 {
					t.Fatalf("a scratch source named %q failed %v, want source_gone at file 0", c.name, f)
				}
			})
		})
	}
}

// TestAScratchCopyStoppedMidFileResumesThenRenamesIntoPlace: a Conversion
// copy that steps behind a Normal Transfer leaves both intents' Active but
// stays in the receiver's Keep. Its staging in the scratch folder's
// .transfers/ keeps the verified chunks, and once it runs again it sends
// only the rest and renames the file to its name.
func TestAScratchCopyStoppedMidFileResumesThenRenamesIntoPlace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransferOf(t, w[0], w[1], sizes[:1])
		w[1].timings.SyncBytes = chunk
		if err := os.Mkdir(w[1].scratch(), 0o750); err != nil {
			t.Fatal(err)
		}
		held := make(chan struct{})
		tr.sent.setHold(func(_, index int) {
			if index == 3 {
				<-held
			}
		})
		w[0].start()
		w[1].start()
		tr.toScratch("in/s-7.mkv")
		w[1].h.await(t, "three chunks verified", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			return len(fs) > 0 && receiving(fs[len(fs)-1], tr.id).GetVerifiedBytes() >= 3*chunk
		})
		staged := filepath.Join(w[1].scratch(), ".transfers", tr.id.String(), "0.part")
		if _, err := os.Stat(staged); err != nil {
			t.Fatalf("the copy's staged file in the scratch folder: %v", err)
		}

		tr.sender.h.set(&tv1.Intent{}, role.Live)
		tr.receiver.h.set(&tv1.Intent{Keep: []*tv1.Kept{tr.kept()}}, role.Live)
		tr.sent.setHold(nil)
		close(held)
		synctest.Wait()
		tr.toScratch("in/s-7.mkv")
		w[1].h.await(t, "the file verified", func(rs []*tv1.Result, _ []*tv1.Facts) bool {
			return len(verifiedAt(rs, tr.id, 1)) == 1
		})
		for i := range 3 {
			if n := tr.sent.count(0, i); n != 1 {
				t.Errorf("chunk %d, verified before the copy stepped aside, was sent %d times", i, n)
			}
		}
		if got, want := sum(t, filepath.Join(w[1].scratch(), "in", "s-7.mkv")), sum(t, w[0].source(0)); got != want {
			t.Errorf("the copy in the scratch folder hashes %x, want its source's %x", got, want)
		}
		if _, err := os.Stat(staged); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the staged file after the rename: %v", err)
		}
	})
}
