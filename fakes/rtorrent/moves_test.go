package rtorrent_test

import (
	"bytes"
	"encoding/base64"
	"html"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

var integer = regexp.MustCompile(`<(?:i4|i8|int)>(-?\d+)</`)

// call is one XML-RPC call with string params, base64 for a param
// starting with base64:, answering the response's integers in order.
func call(t *testing.T, f faketest.Served, method string, params ...string) []int64 {
	t.Helper()
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?><methodCall><methodName>" + method + "</methodName><params>")
	for _, p := range params {
		if data, ok := strings.CutPrefix(p, "base64:"); ok {
			b.WriteString("<param><value><base64>" + data + "</base64></value></param>")
			continue
		}
		b.WriteString("<param><value><string>" + html.EscapeString(p) + "</string></value></param>")
	}
	b.WriteString("</params></methodCall>")
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, f.Address.URL("/RPC2"), bytes.NewReader([]byte(b.String())))
	req.Header.Set("Content-Type", "text/xml")
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	if bytes.Contains(body, []byte("faultCode")) {
		t.Fatalf("%s: %s", method, body)
	}
	var out []int64
	for _, m := range integer.FindAllSubmatch(body, -1) {
		n, _ := strconv.ParseInt(string(m[1]), 10, 64)
		out = append(out, n)
	}
	return out
}

func TestAStoppedLoadHoldsNothingUntilACheckAndTheLastSuccessRises(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cat, shared := clientfaketest.Tracked(t)
		c := rtorrent.Client{Node: "den", Version: "0.16", Folder: "/downloads", Dir: t.TempDir(), Rate: 1}
		whole, short := cat.Releases[0], cat.Releases[1]
		clientfaketest.Place(t, filepath.Join(c.Dir, "Movies"), whole, clientfaketest.Size)
		clientfaketest.Place(t, filepath.Join(c.Dir, "Movies"), short, clientfaketest.Size/2)
		f := faketest.InMemory(t, rtorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
		for _, r := range cat.Releases {
			call(t, f, "load.raw", "", "base64:"+base64.StdEncoding.EncodeToString(r.Metainfo), "d.directory.set=/downloads/Movies")
		}
		completed := func() []int64 {
			return []int64{call(t, f, "d.completed_bytes", whole.InfoHash.String())[0], call(t, f, "d.completed_bytes", short.InfoHash.String())[0]}
		}
		if got := completed(); got[0] != 0 || got[1] != 0 {
			t.Fatalf("after the stopped loads the torrents hold %v bytes, want none until a check", got)
		}
		for _, r := range cat.Releases {
			call(t, f, "d.check_hash", r.InfoHash.String())
		}
		if got := completed(); got[0] != clientfaketest.Size || got[1] != clientfaketest.Size/2 {
			t.Fatalf("after the checks the torrents hold %v bytes, want %d over the whole file and %d over the short one", got, clientfaketest.Size, clientfaketest.Size/2)
		}
		if state := call(t, f, "d.state", whole.InfoHash.String())[0]; state != 0 {
			t.Fatalf("after its check d.state is %d, want 0", state)
		}
		call(t, f, "d.start", whole.InfoHash.String())
		last := func() int64 { return call(t, f, "t.multicall", whole.InfoHash.String(), "", "t.success_time_last=")[0] }
		before := last()
		time.Sleep(31 * time.Second)
		if after := last(); before == 0 || after != before+31 {
			t.Fatalf("t.success_time_last read %d then %d 31s on, want it 31 seconds later", before, after)
		}
	})
}
