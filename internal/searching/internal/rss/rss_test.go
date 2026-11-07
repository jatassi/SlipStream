package rss_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/searching/internal/rss"
)

const hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func feed(items ...string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torrent="http://xmlns.ezrss.it/0.1/">
  <channel>
    <title>Brook</title>
` + strings.Join(items, "") + `  </channel>
</rss>
`)
}

func TestCheckFindsHowTheItemsAreRead(t *testing.T) {
	for _, tc := range []struct {
		name string
		body []byte
		want rss.Reading
	}{
		{
			name: "an enclosure with its length, seeders in the description",
			body: feed(`<item><title>Orchard.2021</title><guid>1</guid><link>https://brook.test/details/1</link>
<enclosure url="https://brook.test/download/1.torrent" length="6000" type="application/x-bittorrent" />
<description>Seeders: 12, Leechers: 3</description></item>`),
			want: rss.Reading{Link: rss.Enclosure, SeedersFromDescription: true},
		},
		{
			name: "the torrent namespace's magnet link, size and seeds",
			body: feed(`<item><title>Orchard.2021</title><torrent:magnetURI>magnet:?xt=urn:btih:` + hashA + `</torrent:magnetURI>
<torrent:contentLength>6000</torrent:contentLength><torrent:seeds>4</torrent:seeds></item>`),
			want: rss.Reading{Link: rss.Torrent},
		},
		{
			name: "the item's link, size in the description",
			body: feed(`<item><title>Orchard.2021</title><link>https://brook.test/download/1.torrent</link>
<description>Size: 1.5 GB</description></item>`),
			want: rss.Reading{Link: rss.Item, SizeFromDescription: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := rss.Check(tc.body, false)
			if err != nil || got != tc.want || n != 1 {
				t.Fatalf("Check = %+v, %d items, %v; want %+v, 1 item", got, n, err, tc.want)
			}
		})
	}
}

func TestCheckRefusesAFeedItCantUse(t *testing.T) {
	good := `<item><title>Orchard.2021</title><guid>1</guid><enclosure url="https://brook.test/1.torrent" length="6000" /></item>`
	for _, tc := range []struct {
		name        string
		body        []byte
		allowNoSize bool
		want        rss.Refusal
	}{
		{"no items", feed(), false, rss.NoItems},
		{"no title", feed(good, `<item><guid>2</guid><enclosure url="https://brook.test/2.torrent" length="6000" /></item>`), false, rss.NoTitle},
		{"no link", feed(good, `<item><title>Road.2019</title><guid>2</guid></item>`), false, rss.NoLink},
		{"repeated GUID", feed(good, `<item><title>Road.2019</title><guid>1</guid><enclosure url="https://brook.test/2.torrent" length="10" /></item>`), false, rss.RepeatedGUID},
		{"no size", feed(good, `<item><title>Road.2019</title><guid>2</guid><enclosure url="https://brook.test/2.torrent" /></item>`), false, rss.NoSize},
		{"no size, allowed", feed(good, `<item><title>Road.2019</title><guid>2</guid><enclosure url="https://brook.test/2.torrent" /></item>`), true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := rss.Check(tc.body, tc.allowNoSize)
			var got rss.Refusal
			if err != nil && !errors.As(err, &got) {
				t.Fatalf("Check = %v, want a Refusal", err)
			}
			if got != tc.want {
				t.Fatalf("Check refused with %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMalformedXMLFailsWithoutARefusal(t *testing.T) {
	_, _, err := rss.Check([]byte("\x00malformed"), false)
	var refusal rss.Refusal
	if err == nil || errors.As(err, &refusal) {
		t.Fatalf("Check of malformed XML = %v, want a read error", err)
	}
	if _, err := rss.Read([]byte("<rss><channel><item>"), rss.Reading{Link: rss.Enclosure}); err == nil {
		t.Fatal("Read of a cut-off feed passed, want a read error")
	}
}

func TestReadReadsEachItemByTheRulesAndLeavesOutTheRest(t *testing.T) {
	body := feed(
		`<item><title>Orchard.2021</title><guid>1</guid><link>https://brook.test/details/1</link><pubDate>Mon, 01 Jun 2026 12:00:00 +0000</pubDate>
<enclosure url="https://brook.test/download/1.torrent" length="6000" /><torrent:infoHash>`+strings.ToUpper(hashA)+`</torrent:infoHash>
<description>6 seeders</description></item>`,
		`<item><title>Meadow.2020</title><guid>2</guid><enclosure url="magnet:?xt=urn:btih:`+strings.Repeat("b", 40)+`&amp;dn=Meadow" />
<description>Size: 2.5 KiB, Seeders: 1</description></item>`,
		`<item><title>Orchard.2021.Again</title><guid>1</guid><enclosure url="https://brook.test/download/3.torrent" length="1" /></item>`,
		`<item><title>No.Link</title><guid>4</guid></item>`,
		`<item><title>No.Size</title><guid>5</guid><enclosure url="https://brook.test/download/5.torrent" /></item>`,
	)
	got, err := rss.Read(body, rss.Reading{Link: rss.Enclosure, SizeFromDescription: true, SeedersFromDescription: true})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, e := range got {
		lines = append(lines, fmt.Sprintf("%s %s %q %q %v %d %d %s %s", e.GUID, e.Title, e.Link, e.Magnet, e.InfoHash, e.Size, e.Seeders, e.Published.Format("2006-01-02"), e.Details))
	}
	want := []string{
		`1 Orchard.2021 "https://brook.test/download/1.torrent" "" ` + hashA + ` 6000 6 2026-06-01 https://brook.test/details/1`,
		`2 Meadow.2020 "" "magnet:?xt=urn:btih:` + strings.Repeat("b", 40) + `&dn=Meadow" ` + strings.Repeat("b", 40) + ` 2560 1 0001-01-01 `,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Read =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}

	all, err := rss.Read(body, rss.Reading{Link: rss.Enclosure, SizeFromDescription: true, AllowNoSize: true})
	if err != nil || len(all) != 3 || all[2].Title != "No.Size" || all[2].Size != 0 {
		t.Fatalf("Read allowing no size = %+v, %v; want the item with no size kept as the third", all, err)
	}
}
