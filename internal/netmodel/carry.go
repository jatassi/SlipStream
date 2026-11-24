package netmodel

import (
	"crypto/sha256"
	"net/netip"
	"slices"
)

// Stored is what the Controller keeps of a Node's addresses beside its
// facts, so a Node restart can't lose the address it had before.
type Stored struct {
	Public4, Previous4   netip.Addr
	Prefixes6, Previous6 []netip.Prefix
}

// Carry moves a stored row's addresses into fresh facts' previous fields
// and answers the row to store with them. Previous4 becomes the stored
// Public4 when both are known and differ, else stays the stored Previous4.
// Previous6 is the stored /64s the fresh set lacks when the set changed,
// else the stored Previous6. An unknown fresh Public4 keeps the stored one,
// so the next known answer is still compared with the last known address.
func (r Rules) Carry(stored Stored, fresh NetworkFacts) (NetworkFacts, Stored) {
	public4 := r.Public4(fresh.Observed())
	prefixes := r.Prefixes6(fresh.Interfaces)
	next := Stored{Public4: public4, Previous4: stored.Previous4, Prefixes6: prefixes, Previous6: stored.Previous6}
	switch {
	case !public4.IsValid():
		next.Public4 = stored.Public4
	case stored.Public4.IsValid() && stored.Public4 != public4:
		next.Previous4 = stored.Public4
	}
	if !slices.Equal(stored.Prefixes6, prefixes) {
		next.Previous6 = nil
		for _, p := range stored.Prefixes6 {
			if !slices.Contains(prefixes, p) {
				next.Previous6 = append(next.Previous6, p)
			}
		}
	}
	fresh.Previous4, fresh.Previous6 = next.Previous4, next.Previous6
	return fresh, next
}

// Summary is the network channel's facts. Revision is a digest of the
// class, Public4 and the Global /64s only, so rotating addresses inside one
// /64 never change it.
type Summary struct {
	Watched  bool
	Class    Class4
	Public4  netip.Addr
	Prefix6  []netip.Prefix
	Revision [16]byte
}

func (r Rules) Summary(f NetworkFacts) Summary {
	o := f.Observed()
	class, _ := r.Classify(o)
	s := Summary{Watched: f.Watched, Class: class, Public4: r.Public4(o), Prefix6: r.Prefixes6(f.Interfaces)}
	h := sha256.New()
	h.Write([]byte{byte(s.Class)})
	a := s.Public4.As16()
	h.Write(a[:])
	for _, p := range s.Prefix6 {
		b, _ := p.MarshalBinary()
		h.Write(b)
	}
	copy(s.Revision[:], h.Sum(nil))
	return s
}
