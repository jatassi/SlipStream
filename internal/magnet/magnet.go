// Package magnet reads and writes magnet links (BEP 9) to v1 and hybrid
// torrents, and composes a link's .torrent once a peer sent its info
// dictionary.
package magnet

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Link is a magnet link to a v1 or hybrid torrent.
type Link struct {
	InfoHash bencode.InfoHash
	// Name is the dn display name, empty when the link has none.
	Name string
	// Trackers is each tr URL in link order, duplicates dropped.
	Trackers []string
}

// InvalidError is a magnet link SlipStream can't read.
type InvalidError struct{ Why string }

func (e *InvalidError) Error() string { return "magnet: " + e.Why }

// ErrWrongInfo is an info dictionary whose SHA-1 isn't the link's info
// hash.
var ErrWrongInfo = errors.New("magnet: the info dictionary's hash isn't the link's info hash")

const scheme = "magnet:"

// Is reports whether link uses the magnet scheme, in any case.
func Is(link string) bool {
	return len(link) >= len(scheme) && strings.EqualFold(link[:len(scheme)], scheme)
}

// Parse reads link. xt=urn:btih: takes 40 hex digits or 32 base32
// characters, in any case. A link with only urn:btmh: fails with
// bencode.ErrV2Only, and a hybrid link's btmh is ignored. A link with no
// btih, or two different ones, fails with *InvalidError.
func Parse(link string) (Link, error) {
	if !Is(link) {
		return Link{}, &InvalidError{Why: "the link isn't a magnet link"}
	}
	query, _ := strings.CutPrefix(link[len(scheme):], "?")
	var p params
	for part := range strings.SplitSeq(query, "&") {
		key, value, _ := strings.Cut(part, "=")
		value, err := url.QueryUnescape(value)
		if err != nil {
			return Link{}, &InvalidError{Why: key + " has a % that two hex digits don't follow"}
		}
		if err := p.add(key, value); err != nil {
			return Link{}, err
		}
	}
	switch {
	case len(p.hashes) == 0 && p.v2:
		return Link{}, bencode.ErrV2Only
	case len(p.hashes) == 0:
		return Link{}, &InvalidError{Why: "the link has no BitTorrent v1 info hash"}
	case len(p.hashes) > 1:
		return Link{}, &InvalidError{Why: "the link names two different info hashes"}
	}
	p.link.InfoHash = p.hashes[0]
	return p.link, nil
}

// params is what a magnet link's parameters said so far.
type params struct {
	link   Link
	hashes []bencode.InfoHash
	v2     bool
}

func (p *params) add(key, value string) error {
	switch key {
	case "xt":
		urn := strings.ToLower(value)
		switch {
		case strings.HasPrefix(urn, "urn:btih:"):
			h, err := infoHash(value[len("urn:btih:"):])
			if err != nil {
				return err
			}
			if !slices.Contains(p.hashes, h) {
				p.hashes = append(p.hashes, h)
			}
		case strings.HasPrefix(urn, "urn:btmh:"):
			p.v2 = true
		}
	case "dn":
		if p.link.Name == "" {
			p.link.Name = value
		}
	case "tr":
		if value != "" && !slices.Contains(p.link.Trackers, value) {
			p.link.Trackers = append(p.link.Trackers, value)
		}
	}
	return nil
}

func infoHash(s string) (bencode.InfoHash, error) {
	var h bencode.InfoHash
	var b []byte
	var err error
	switch len(s) {
	case hex.EncodedLen(len(h)):
		b, err = hex.DecodeString(s)
	case base32.StdEncoding.EncodedLen(len(h)):
		b, err = base32.StdEncoding.DecodeString(strings.ToUpper(s))
	default:
		err = errors.New("wrong length")
	}
	if err != nil {
		return bencode.InfoHash{}, &InvalidError{Why: fmt.Sprintf("info hash %q is neither 40 hex digits nor 32 base32 characters", s)}
	}
	copy(h[:], b)
	return h, nil
}

// String is l as a magnet link: its info hash in hex, then its name and
// trackers when it has them.
func (l Link) String() string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:" + l.InfoHash.String())
	if l.Name != "" {
		b.WriteString("&dn=" + url.QueryEscape(l.Name))
	}
	for _, tr := range l.Trackers {
		b.WriteString("&tr=" + url.QueryEscape(tr))
	}
	return b.String()
}

// Torrent is the .torrent of l whose info dictionary is info, byte for
// byte: l's first tracker as announce and each tracker its own tier of
// announce-list, as libtorrent lists a magnet link's trackers. A link with
// no tracker has neither. It fails with ErrWrongInfo when info's SHA-1
// isn't l's info hash, with bencode.ErrV2Only for a v2-only info
// dictionary, and with bencode's error for one it can't read.
func (l Link) Torrent(info []byte) ([]byte, error) {
	top := map[string]any{"info": bencode.Raw(info)}
	if len(l.Trackers) > 0 {
		tiers := make([]any, len(l.Trackers))
		for i, tr := range l.Trackers {
			tiers[i] = []any{tr}
		}
		top["announce"], top["announce-list"] = l.Trackers[0], tiers
	}
	out, err := bencode.Encode(top)
	if err != nil {
		return nil, fmt.Errorf("magnet: %w", err)
	}
	m, err := bencode.DecodeMetainfo(out)
	if err != nil {
		return nil, err
	}
	if m.InfoHash != l.InfoHash {
		return nil, ErrWrongInfo
	}
	return out, nil
}
