package transfers

import (
	"bytes"
	"slices"
	"testing"
	"time"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

func TestAnActiveTransferNamesWhatEachEndNeeds(t *testing.T) {
	w := newWorld().add(1, seedbox, onHome, gb, 2*gb, 3*gb).file(1, 0, func(f *File) { f.Verified = true }).file(1, 1, func(f *File) { f.Placed = true })
	w.s.limits[seedbox] = 5 << 20
	j := judge(w.s)

	send := j.intents.of(seedbox)
	if send.GetSendLimit() != 5<<20 || len(send.GetActive()) != 1 || len(send.GetKeep()) != 0 {
		t.Fatalf("the sender's intent = %v, want its limit and one Transfer to send", send)
	}
	a := send.GetActive()[0]
	if a.GetSide() != tv1.Side_SIDE_SEND || a.GetDestination() != nil || len(a.GetPeer().GetAddresses()) != 0 || a.GetPeer().GetKey()[0] != 3 {
		t.Errorf("the sender's Transfer = %v, want it sent to the Controller's key, with no addresses", a)
	}
	if fs := a.GetFiles(); len(fs) != 1 || fs[0].GetPosition() != 2 || fs[0].GetSize() != 3*gb || fs[0].GetSource().GetTorrent().GetFileIndex() != 2 {
		t.Errorf("the files to send = %v, want only the one neither verified nor placed", fs)
	}

	recv := j.intents.of(home)
	r := recv.GetActive()[0]
	if r.GetSide() != tv1.Side_SIDE_RECEIVE || r.GetDestination().GetRoot().GetPlace() != 1 || !bytes.Equal(r.GetDestination().GetRoot().GetRootId(), onHome.Bytes()) {
		t.Errorf("the receiver's Transfer = %v, want the root at its place", r)
	}
	if got := r.GetPeer().GetAddresses(); !slices.Equal(got, []string{"seedbox:8485"}) {
		t.Errorf("the receiver dials %v, want the Worker's addresses", got)
	}
	if !slices.EqualFunc(recv.GetKeep(), [][]byte{tid(1).Bytes()}, bytes.Equal) {
		t.Errorf("the receiver keeps %x, want the Transfer it stages", recv.GetKeep())
	}
	if in := j.intents.of(vps); len(in.GetActive()) != 0 || len(in.GetKeep()) != 0 {
		t.Errorf("a Node no Transfer names has %v, want the empty intent", in)
	}
}

func TestTheReceiverKeepsEveryRecordItStagesInIDOrder(t *testing.T) {
	w := newWorld().add(2, seedbox, onNas, gb).add(1, vps, onNas, 200*gb).add(3, seedbox, onNas, gb).
		with(func(s *snapshot) { s.transfers[2].Failure = &Failure{Code: SourceGone, Position: 0} })
	got := judge(w.s).intents.of(nas).GetKeep()
	want := [][]byte{tid(1).Bytes(), tid(2).Bytes(), tid(3).Bytes()}
	if !slices.EqualFunc(got, want, bytes.Equal) {
		t.Errorf("keep = %x, want queued, needing space and failed Transfers, sorted", got)
	}
}

func TestARecentHigherRouteIsJudgedAgainWhenItsRetryEnds(t *testing.T) {
	at := t0.Add(-time.Minute)
	j := judge(newWorld().add(1, seedbox, onNas, gb).pair(seedbox, nas, tv1.Route_ROUTE_SENDER_DIALS, at).s)
	if want := at.Add(time.Hour); !j.next.Equal(want) {
		t.Errorf("next = %v, want the route retry's end %v", j.next, want)
	}
	if j := judge(newWorld().add(1, seedbox, onNas, gb).s); !j.next.IsZero() {
		t.Errorf("next = %v with no pair row, want never", j.next)
	}
}

func TestJudgedFailuresNameTheirFile(t *testing.T) {
	w := newWorld().add(1, seedbox, onNas, gb).add(2, seedbox, onVps, gb, gb).
		dest(onNas, func(d *roots.Destination) { d.Node = seedbox }).
		with(func(s *snapshot) {
			s.sources[sourceAt{seedbox, torrent(2, 1)}] = source.Readiness{State: source.WrittenOff}
		})
	got := judge(w.s).fail
	want := []judged{{tid(1), Rerouted, -1}, {tid(2), SourceWrittenOff, 1}}
	if !slices.Equal(got, want) {
		t.Errorf("judged failures = %v, want %v", got, want)
	}
}

func TestAStoredFailureIsNotJudgedAgain(t *testing.T) {
	w := newWorld().add(1, seedbox, onNas, gb).dest(onNas, func(d *roots.Destination) { d.Node = seedbox }).
		with(func(s *snapshot) { s.transfers[0].Failure = &Failure{Code: Damaged, Position: 0} })
	if j := judge(w.s); len(j.fail) != 0 || j.views[0].Failure.Code != Damaged {
		t.Errorf("judge = %v failing %v, want the stored failure alone", j.fail, j.views[0].Failure)
	}
}

func TestACopyingTransferEstimatesFromItsReceiversRate(t *testing.T) {
	w := newWorld().add(1, seedbox, onNas, 10*gb).
		receiving(nas, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, Moving: true, VerifiedBytes: 4 * gb, Rate: gb / 100, Lanes: 3})
	v := judge(w.s).views[0]
	if v.Verified != 4*gb || v.Total != 10*gb || v.Rate != gb/100 || v.Estimate != 600*time.Second || v.Details.Lanes != 3 {
		t.Errorf("view = %d of %d at %d/s, %v left, %+v; want 4 GB of 10 GB with 10 min left on 3 lanes", v.Verified, v.Total, v.Rate, v.Estimate, v.Details)
	}
}
