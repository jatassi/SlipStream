package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func benc(b *bytes.Buffer, v any) {
	switch x := v.(type) {
	case int:
		fmt.Fprintf(b, "i%de", x)
	case string:
		fmt.Fprintf(b, "%d:%s", len(x), x)
	case []byte:
		fmt.Fprintf(b, "%d:", len(x))
		b.Write(x)
	case []any:
		b.WriteByte('l')
		for _, e := range x {
			benc(b, e)
		}
		b.WriteByte('e')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('d')
		for _, k := range keys {
			benc(b, k)
			benc(b, x[k])
		}
		b.WriteByte('e')
	}
}

func enc(v any) []byte {
	var b bytes.Buffer
	benc(&b, v)
	return b.Bytes()
}

type entry struct {
	Name     string `json:"name"`
	Infohash string `json:"infohash"`
	Torrent  string `json:"torrent"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] != "gen" {
		fmt.Fprintln(os.Stderr, "usage: fixtures gen --out DIR --count N --piece-len L --pieces P --announce URL [--announce-list 'A1,A2;B1'] [--name-prefix p]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	out := fs.String("out", "", "")
	count := fs.Int("count", 1, "")
	pl := fs.Int("piece-len", 4194304, "")
	np := fs.Int("pieces", 1024, "")
	announce := fs.String("announce", "", "")
	alist := fs.String("announce-list", "", "")
	prefix := fs.String("name-prefix", "fx", "")
	fs.Parse(args[1:])

	zh := sha1.Sum(make([]byte, *pl))
	pieces := bytes.Repeat(zh[:], *np)
	size := int64(*pl) * int64(*np)

	var tiers []any
	if *alist != "" {
		for _, t := range strings.Split(*alist, ";") {
			var tier []any
			for _, u := range strings.Split(t, ",") {
				tier = append(tier, u)
			}
			tiers = append(tiers, tier)
		}
	}
	ann := *announce
	if len(tiers) > 0 {
		ann = tiers[0].([]any)[0].(string)
	}

	os.MkdirAll(filepath.Join(*out, "torrents"), 0o755)
	os.MkdirAll(filepath.Join(*out, "data"), 0o755)
	abs, _ := filepath.Abs(*out)
	var man []entry
	for i := 0; i < *count; i++ {
		name := fmt.Sprintf("%s-%06d", *prefix, i)
		info := map[string]any{
			"length":       int(size),
			"name":         name,
			"piece length": *pl,
			"pieces":       pieces,
			"private":      1,
		}
		ih := sha1.Sum(enc(info))
		mi := map[string]any{"announce": ann, "info": info}
		if len(tiers) > 0 {
			mi["announce-list"] = tiers
		}
		tp := filepath.Join(abs, "torrents", name+".torrent")
		dp := filepath.Join(abs, "data", name)
		must(os.WriteFile(tp, enc(mi), 0o644))
		f, err := os.Create(dp)
		must(err)
		must(f.Truncate(size))
		f.Close()
		man = append(man, entry{name, hex.EncodeToString(ih[:]), tp, dp, size})
	}
	j, _ := json.MarshalIndent(man, "", " ")
	must(os.WriteFile(filepath.Join(abs, "manifest.json"), j, 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
