package bencode_test

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
)

func TestDecodeMetainfoKeepsTheRawInfo(t *testing.T) {
	data, err := os.ReadFile("testdata/multi.torrent")
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(data)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.InfoHash.String(), "810c158dc4e63a093e093516a0ced88429f4f2fd"; got != want {
		t.Errorf("InfoHash = %s, want %s", got, want)
	}
	if got, want := string(m.RawInfo), string(data[59:len(data)-1]); got != want {
		t.Errorf("RawInfo = %q, want bytes 59 to %d", got, len(data)-1)
	}
	m.RawInfo, m.Info.Pieces = nil, m.Info.Pieces[:0]
	want := bencode.Metainfo{
		Announce: "https://127.0.0.1:8443/passkey/announce",
		Info: bencode.Info{
			Name:        "Orchard",
			PieceLength: 16384,
			Pieces:      []byte{},
			Private:     true,
			Files: []bencode.File{
				{Path: []string{"Extras", "trailer.mkv"}, Length: 6},
				{Path: []string{"feature.mkv"}, Length: 84000},
			},
		},
		InfoHash: m.InfoHash,
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("DecodeMetainfo =\n%+v\nwant\n%+v", m, want)
	}
}

func TestDecodeMetainfoReadsASingleFile(t *testing.T) {
	info := "d6:lengthi5e4:name5:a.mkv12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaa7:privatei1ee"
	m, err := bencode.DecodeMetainfo([]byte("d8:announce4:http4:info" + info + "e"))
	if err != nil {
		t.Fatal(err)
	}
	if string(m.RawInfo) != info || m.Info.Name != "a.mkv" || !m.Info.Private || !reflect.DeepEqual(m.Info.Files, []bencode.File{{Length: 5}}) {
		t.Errorf("DecodeMetainfo = %+v", m)
	}
	if got, want := m.InfoHash.String(), "dc680d8df49555364f9be42ab78c83e633e1b121"; got != want {
		t.Errorf("InfoHash = %s, want %s", got, want)
	}
}

func TestTrackersFollowTheAnnounceList(t *testing.T) {
	const info = "4:infod6:lengthi5e4:name5:a.mkv12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaae"
	for _, c := range []struct {
		name, top string
		want      []string
	}{
		{"announce alone", "8:announce10:http://a/a", []string{"http://a/a"}},
		{"tiers in order, announce ignored", "8:announce10:http://a/a13:announce-listll10:http://b/b10:http://c/cel10:http://d/dee", []string{"http://b/b", "http://c/c", "http://d/d"}},
		{"malformed tiers skipped", "8:announce10:http://a/a13:announce-listli7el10:http://b/bi3eee", []string{"http://b/b"}},
		{"an empty list falls back to announce", "8:announce10:http://a/a13:announce-listle", []string{"http://a/a"}},
		{"no tracker", "", nil},
	} {
		m, err := bencode.DecodeMetainfo([]byte("d" + c.top + info + "e"))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := m.Trackers(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Trackers = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDecodeMetainfoRefusesV2Only(t *testing.T) {
	for _, info := range []string{
		"d9:file treed5:a.mkvd0:d6:lengthi5e11:pieces root32:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaeee12:meta versioni2e4:name5:a.mkv12:piece lengthi16384ee",
		"d12:meta versioni2e4:name5:a.mkv12:piece lengthi16384ee",
	} {
		_, err := bencode.DecodeMetainfo([]byte("d4:info" + info + "e"))
		if !errors.Is(err, bencode.ErrV2Only) {
			t.Errorf("DecodeMetainfo(%s) err = %v, want ErrV2Only", info, err)
		}
	}
}

func TestDecodeRefusesMalformedData(t *testing.T) {
	for _, tc := range []struct{ data, want string }{
		{"i03e", `bencode: at byte 1: "03" is not a bencode integer`},
		{"i-0e", `bencode: at byte 1: "-0" is not a bencode integer`},
		{"i+1e", `bencode: at byte 1: "+1" is not a bencode integer`},
		{"ie", `bencode: at byte 1: "" is not a bencode integer`},
		{"i1", "bencode: at byte 1: unterminated integer"},
		{"5:ab", "bencode: at byte 2: string length 5 runs past the end"},
		{"di1ei2ee", "bencode: at byte 1: a dictionary key is not a string"},
		{"l", "bencode: at byte 1: unexpected end"},
		{"i1ei2e", "bencode: at byte 3: data after the value"},
		{"x", `bencode: at byte 0: unexpected 'x'`},
	} {
		if _, err := bencode.Decode([]byte(tc.data)); err == nil || err.Error() != tc.want {
			t.Errorf("Decode(%q) err = %v, want %s", tc.data, err, tc.want)
		}
	}
}

func TestEncodeRoundTrips(t *testing.T) {
	v := map[string]any{
		"zeta":  []any{int64(-3), "x"},
		"alpha": map[string]any{"n": int64(0), "b": "bytes"},
	}
	data, err := bencode.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	if want := "d5:alphad1:b5:bytes1:ni0ee4:zetali-3e1:xee"; string(data) != want {
		t.Fatalf("Encode = %s, want %s", data, want)
	}
	back, err := bencode.Decode(data)
	if err != nil || !reflect.DeepEqual(back, v) {
		t.Fatalf("Decode(Encode(v)) = %#v, %v; want %#v", back, err, v)
	}
	if _, err := bencode.Encode(1.5); err == nil || err.Error() != "bencode: cannot encode float64" {
		t.Fatalf("Encode(1.5) err = %v", err)
	}
}

func TestParseInfoHash(t *testing.T) {
	h, err := bencode.ParseInfoHash("810c158dc4e63a093e093516a0ced88429f4f2fd")
	if err != nil || h.String() != "810c158dc4e63a093e093516a0ced88429f4f2fd" {
		t.Fatalf("ParseInfoHash = %s, %v", h, err)
	}
	if _, err := bencode.ParseInfoHash("810c"); err == nil || err.Error() != `bencode: info hash "810c" is not 40 hex digits` {
		t.Fatalf("ParseInfoHash(short) err = %v", err)
	}
}
