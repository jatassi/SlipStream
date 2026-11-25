package reachability

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// factsVersion is the version of the facts and server_facts documents. A
// row of another version reads as never read.
const factsVersion = 1

type factsDoc struct {
	V     int                   `json:"v"`
	Facts netmodel.NetworkFacts `json:"facts"`
}

type serverDoc struct {
	V     int               `json:"v"`
	Facts judge.ServerFacts `json:"facts"`
	Local uint16            `json:"local"`
}

func encode(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode a stored document: %w", err)
	}
	return string(b), nil
}

// stored is one Node's row: its last facts and the addresses Carry keeps.
type stored struct {
	facts netmodel.NetworkFacts
	read  bool // the facts parsed at this version
	carry netmodel.Stored
}

func storedOf(row db.ReachabilityNode) (stored, error) {
	s := stored{}
	var err error
	if s.carry.Public4, err = addr4(row.Public4); err != nil {
		return s, err
	}
	if s.carry.Previous4, err = addr4(row.Previous4); err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(row.Prefixes6), &s.carry.Prefixes6); err != nil {
		return s, fmt.Errorf("read a Node's prefixes: %w", err)
	}
	if err := json.Unmarshal([]byte(row.Previous6), &s.carry.Previous6); err != nil {
		return s, fmt.Errorf("read a Node's previous prefixes: %w", err)
	}
	var doc factsDoc
	if err := json.Unmarshal([]byte(row.Facts), &doc); err != nil {
		return s, fmt.Errorf("read a Node's facts: %w", err)
	}
	s.facts, s.read = doc.Facts, doc.V == factsVersion
	return s, nil
}

func addr4(b []byte) (netip.Addr, error) {
	if b == nil {
		return netip.Addr{}, nil
	}
	a, ok := netip.AddrFromSlice(b)
	if !ok || !a.Is4() {
		return netip.Addr{}, errors.New("a stored IPv4 address isn't four bytes")
	}
	return a, nil
}

func bytes4(a netip.Addr) []byte {
	if !a.IsValid() {
		return nil
	}
	b := a.As4()
	return b[:]
}

func prefixes(ps []netip.Prefix) (string, error) {
	if ps == nil {
		ps = []netip.Prefix{}
	}
	return encode(ps)
}

func nodeRow(node role.NodeID, f netmodel.NetworkFacts, c netmodel.Stored) (db.UpsertNodeParams, error) {
	facts, err := encode(factsDoc{V: factsVersion, Facts: f})
	if err != nil {
		return db.UpsertNodeParams{}, err
	}
	p6, err := prefixes(c.Prefixes6)
	if err != nil {
		return db.UpsertNodeParams{}, err
	}
	prev6, err := prefixes(c.Previous6)
	if err != nil {
		return db.UpsertNodeParams{}, err
	}
	return db.UpsertNodeParams{
		NodeID: node, Facts: facts, ObservedAt: storage.FromTime(f.At),
		Public4: bytes4(c.Public4), Prefixes6: p6, Previous4: bytes4(c.Previous4), Previous6: prev6,
	}, nil
}

// result is one Exposure's stored result.
type result struct {
	judged    judge.Judged
	read      bool // judged parsed at this codec version
	judgedAt  time.Time
	failingAt time.Time
	origin    role.NodeID
	testedBy  role.NodeID
}

func resultOf(row db.ReachabilityResult) (result, error) {
	res := result{judgedAt: row.JudgedAt.Time(), failingAt: row.FailingAt.Time(), origin: row.OriginNodeID, testedBy: row.TestedByNodeID}
	j, err := judge.Decode([]byte(row.Judged))
	switch {
	case errors.Is(err, judge.ErrOtherVersion):
		return res, nil
	case err != nil:
		return res, err
	}
	res.judged, res.read = j, true
	return res, nil
}

// nextFailingAt is when the verdict began resting on a failing check:
// kept while it still does, now when it starts to, and zero otherwise.
func nextFailingAt(prev time.Time, j judge.Judged, now time.Time) time.Time {
	switch {
	case !j.Failing():
		return time.Time{}
	case prev.IsZero():
		return now
	}
	return prev
}

func resultRow(j judge.Judged, at, failingAt time.Time, testedBy role.NodeID) (db.UpsertResultParams, error) {
	b, err := judge.Encode(j)
	if err != nil {
		return db.UpsertResultParams{}, err
	}
	tested := int64(0)
	if j.Tested {
		tested = 1
	}
	return db.UpsertResultParams{
		ExposureID: j.Exposure, Judged: string(b), Verdict: j.Verdict.String(), Tested: tested,
		JudgedAt: storage.FromTime(at), FailingAt: storage.FromTime(failingAt), OriginNodeID: j.Origin, TestedByNodeID: testedBy,
	}, nil
}

func serverDocOf(row db.ReachabilityExposure) (serverDoc, bool) {
	if row.ServerFacts == nil {
		return serverDoc{}, false
	}
	var doc serverDoc
	if err := json.Unmarshal([]byte(*row.ServerFacts), &doc); err != nil || doc.V != factsVersion {
		return serverDoc{}, false
	}
	return doc, true
}
