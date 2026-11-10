package grabsapi

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// The API mints every Manual search token here and parses them here alone.
// The web holds them and sends them back, and never builds or splits one.

// releaseIDOf is a Release's id: its run, a dot and its position.
func releaseIDOf(r grabs.ReleaseRef) string {
	return r.Run.String() + "." + strconv.Itoa(r.Position)
}

func parseReleaseID(s string) (grabs.ReleaseRef, bool) {
	run, position, ok := strings.Cut(s, ".")
	if !ok {
		return grabs.ReleaseRef{}, false
	}
	u, err := storage.ParseUUID(run)
	if err != nil {
		return grabs.ReleaseRef{}, false
	}
	p, err := strconv.Atoi(position)
	if err != nil || p < 0 || strconv.Itoa(p) != position {
		return grabs.ReleaseRef{}, false
	}
	return grabs.ReleaseRef{Run: grabs.SearchID{UUID: u}, Position: p}, true
}

const uuidLen = 16

// cellKeyOf is a cell's key: its Item, its Target and its Edition.
func cellKeyOf(c files.CellKey) string {
	return base64.RawURLEncoding.EncodeToString(cellBytes(c))
}

func cellBytes(c files.CellKey) []byte {
	return append(append(c.Item.Bytes(), c.Target.Bytes()...), c.Edition...)
}

// pinOf is a pin's token: its cell's Item and Target, its Release's run
// and position, then its cell's Edition.
func pinOf(p grabs.Pin) string {
	b := append(append(p.Cell.Item.Bytes(), p.Cell.Target.Bytes()...), p.Release.Run.Bytes()...)
	b = binary.BigEndian.AppendUint32(b, uint32(p.Release.Position)) //nolint:gosec // a position counts one run's Releases
	return base64.RawURLEncoding.EncodeToString(append(b, p.Cell.Edition...))
}

// parsePin is the pin token s names, false for one the API never minted.
func parsePin(s string) (grabs.Pin, bool) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) <= 3*uuidLen+4 {
		return grabs.Pin{}, false
	}
	var ids [3]storage.UUID
	for i := range ids {
		if ids[i], err = storage.UUIDFromBytes(b[i*uuidLen : (i+1)*uuidLen]); err != nil {
			return grabs.Pin{}, false
		}
	}
	position := binary.BigEndian.Uint32(b[3*uuidLen:])
	if uint64(position) > math.MaxInt32 {
		return grabs.Pin{}, false
	}
	return grabs.Pin{
		Cell: files.CellKey{
			Item: works.ItemID{UUID: ids[0]}, Target: targets.TargetID{UUID: ids[1]}, Edition: targets.Edition(b[3*uuidLen+4:]),
		},
		Release: grabs.ReleaseRef{Run: grabs.SearchID{UUID: ids[2]}, Position: int(position)},
	}, true
}
