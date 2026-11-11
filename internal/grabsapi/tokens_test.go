package grabsapi

import (
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAReleaseIDNamesTheReleaseItWasMintedFor(t *testing.T) {
	ref := grabs.ReleaseRef{Run: grabs.SearchID{UUID: storage.NewUUID()}, Position: 41}
	if got, ok := parseReleaseID(releaseIDOf(ref)); !ok || got != ref {
		t.Fatalf("parseReleaseID(releaseIDOf(%v)) = %v, %t", ref, got, ok)
	}
	for _, s := range []string{"", "41", ref.Run.String(), ref.Run.String() + ".", ref.Run.String() + ".-1", ref.Run.String() + ".041", "nope.41"} {
		if got, ok := parseReleaseID(s); ok {
			t.Errorf("parseReleaseID(%q) = %v, want no Release", s, got)
		}
	}
}

func TestAPinNamesTheCellAndReleaseItWasMintedFor(t *testing.T) {
	cell := files.CellKey{Item: works.ItemID{UUID: storage.NewUUID()}, Edition: "directors-cut", Target: targets.TargetID{UUID: storage.NewUUID()}}
	pin := grabs.Pin{Cell: cell, Release: grabs.ReleaseRef{Run: grabs.SearchID{UUID: storage.NewUUID()}, Position: 7}}
	got, ok := parsePin(pinOf(pin))
	if !ok || got != pin {
		t.Fatalf("parsePin(pinOf(%v)) = %v, %t", pin, got, ok)
	}
	other := pin
	other.Release.Position = 8
	if pinOf(other) == pinOf(pin) || cellKeyOf(other.Cell) != cellKeyOf(pin.Cell) {
		t.Fatal("two pins of one cell share its key and differ as tokens")
	}
	whole := cell
	whole.Edition = targets.AnyEdition
	if cellKeyOf(whole) == cellKeyOf(cell) {
		t.Fatal("two Editions of one Item and Target share a key")
	}
	for _, s := range []string{"", "a", releaseIDOf(pin.Release), cellKeyOf(cell), pinOf(pin) + "!"} {
		if got, ok := parsePin(s); ok {
			t.Errorf("parsePin(%q) = %v, want no pin", s, got)
		}
	}
}

func TestACellKeyNamesTheCellItWasMintedFor(t *testing.T) {
	for _, cell := range []files.CellKey{
		{Item: works.ItemID{UUID: storage.NewUUID()}, Edition: "directors-cut", Target: targets.TargetID{UUID: storage.NewUUID()}},
		{Item: works.ItemID{UUID: storage.NewUUID()}, Edition: targets.AnyEdition, Target: targets.TargetID{UUID: storage.NewUUID()}},
	} {
		if got, ok := parseCellKey(cellKeyOf(cell)); !ok || got != cell {
			t.Fatalf("parseCellKey(cellKeyOf(%v)) = %v, %t", cell, got, ok)
		}
	}
	for _, s := range []string{"", "a", "!" + cellKeyOf(files.CellKey{})} {
		if got, ok := parseCellKey(s); ok {
			t.Errorf("parseCellKey(%q) = %v, want no cell", s, got)
		}
	}
}
