package magnet_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
)

const (
	info    = "d6:lengthi5e4:name5:a.mkv12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaa7:privatei1ee"
	hexHash = "dc680d8df49555364f9be42ab78c83e633e1b121"
	b32Hash = "3RUA3DPUSVKTMT434QVLPDED4YZ6DMJB"
	btmh    = "urn:btmh:1220caf1e1c30e81cb361b9ee167c4aa64228a7fa4fa9f6105232b28ad099f3a302e"
)

func hash(t *testing.T) bencode.InfoHash {
	t.Helper()
	h, err := bencode.ParseInfoHash(hexHash)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestParseReadsAMagnetLink(t *testing.T) {
	h := hash(t)
	for _, tc := range []struct {
		name, link string
		want       magnet.Link
	}{
		{"lowercase hex", "magnet:?xt=urn:btih:" + hexHash, magnet.Link{InfoHash: h}},
		{"uppercase hex and scheme", "MAGNET:?xt=urn:BTIH:DC680D8DF49555364F9BE42AB78C83E633E1B121", magnet.Link{InfoHash: h}},
		{"uppercase base32", "magnet:?xt=urn:btih:" + b32Hash, magnet.Link{InfoHash: h}},
		{"lowercase base32", "magnet:?xt=urn:btih:3rua3dpusvktmt434qvlpded4yz6dmjb", magnet.Link{InfoHash: h}},
		{"hybrid", "magnet:?xt=" + btmh + "&xt=urn:btih:" + hexHash, magnet.Link{InfoHash: h}},
		{"the same btih twice", "magnet:?xt=urn:btih:" + hexHash + "&xt=urn:btih:" + b32Hash, magnet.Link{InfoHash: h}},
		{
			"escaped name and trackers in order without duplicates",
			"magnet:?xt=urn:btih:" + hexHash + "&dn=The+Long%20Night%3A+Part+1" +
				"&tr=https%3A%2F%2Ftracker.example%2Fa%2Fannounce&tr=udp://open.example:6969&tr=https%3A%2F%2Ftracker.example%2Fa%2Fannounce",
			magnet.Link{InfoHash: h, Name: "The Long Night: Part 1", Trackers: []string{"https://tracker.example/a/announce", "udp://open.example:6969"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := magnet.Parse(tc.link)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Parse(%q) = %+v, %v; want %+v", tc.link, got, err, tc.want)
			}
		})
	}
}

func TestParseRefusesWhatItCantRead(t *testing.T) {
	invalid := new(*magnet.InvalidError)
	for _, tc := range []struct {
		name, link string
		v2         bool
	}{
		{"only btmh", "magnet:?xt=" + btmh + "&dn=a", true},
		{"no xt", "magnet:?dn=a&tr=udp://open.example:6969", false},
		{"two different btih", "magnet:?xt=urn:btih:" + hexHash + "&xt=urn:btih:" + "0000000000000000000000000000000000000000", false},
		{"a short hash", "magnet:?xt=urn:btih:dc680d8d", false},
		{"a hash that isn't hex", "magnet:?xt=urn:btih:zc680d8df49555364f9be42ab78c83e633e1b121", false},
		{"a broken escape", "magnet:?xt=urn:btih:" + hexHash + "&dn=a%zz", false},
		{"another scheme", "https://indexer.example/download/" + hexHash + ".torrent", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := magnet.Parse(tc.link)
			if tc.v2 && !errors.Is(err, bencode.ErrV2Only) || !tc.v2 && !errors.As(err, invalid) {
				t.Fatalf("Parse(%q) = %v, want ErrV2Only %t, else *InvalidError", tc.link, err, tc.v2)
			}
		})
	}
}

func TestIsTakesTheSchemeInAnyCase(t *testing.T) {
	for link, want := range map[string]bool{"magnet:?xt=x": true, "Magnet:?xt=x": true, "https://a/b": false, "magne": false} {
		if got := magnet.Is(link); got != want {
			t.Errorf("Is(%q) = %t, want %t", link, got, want)
		}
	}
}

func TestStringParsesBackToTheLink(t *testing.T) {
	l := magnet.Link{InfoHash: hash(t), Name: "Café & Co: 100%", Trackers: []string{"https://tracker.example/a/announce?x=1&y=2", "udp://open.example:6969"}}
	got, err := magnet.Parse(l.String())
	if err != nil || !reflect.DeepEqual(got, l) {
		t.Fatalf("Parse(%q) = %+v, %v; want %+v", l.String(), got, err, l)
	}
	if bare := (magnet.Link{InfoHash: l.InfoHash}).String(); bare != "magnet:?xt=urn:btih:"+hexHash {
		t.Errorf("a link with only a hash is %q", bare)
	}
}

func TestTorrentKeepsTheInfoAndListsEachTrackerAsItsOwnTier(t *testing.T) {
	l := magnet.Link{InfoHash: hash(t), Trackers: []string{"https://a.example/announce", "udp://b.example:6969"}}
	raw, err := l.Torrent([]byte(info))
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(m.RawInfo) != info || m.InfoHash != l.InfoHash {
		t.Errorf("the .torrent's info is %q with hash %s, want %q with %s", m.RawInfo, m.InfoHash, info, l.InfoHash)
	}
	if want := [][]string{{"https://a.example/announce"}, {"udp://b.example:6969"}}; m.Announce != l.Trackers[0] || !reflect.DeepEqual(m.AnnounceList, want) {
		t.Errorf("the .torrent announces to %q with tiers %q, want %q with %q", m.Announce, m.AnnounceList, l.Trackers[0], want)
	}
}

func TestTorrentOfATrackerlessLinkAnnouncesNowhere(t *testing.T) {
	raw, err := magnet.Link{InfoHash: hash(t)}.Torrent([]byte(info))
	if err != nil {
		t.Fatal(err)
	}
	if want := "d4:info" + info + "e"; string(raw) != want {
		t.Errorf("Torrent = %q, want %q", raw, want)
	}
}

func TestTorrentRefusesAnInfoOfAnotherHash(t *testing.T) {
	other := magnet.Link{InfoHash: bencode.InfoHash{1}}
	if _, err := other.Torrent([]byte(info)); !errors.Is(err, magnet.ErrWrongInfo) {
		t.Errorf("Torrent of another hash's info = %v, want ErrWrongInfo", err)
	}
	if _, err := (magnet.Link{InfoHash: hash(t)}).Torrent([]byte("d4:name1:ae")); err == nil {
		t.Error("Torrent of an info with no pieces succeeded")
	}
}
