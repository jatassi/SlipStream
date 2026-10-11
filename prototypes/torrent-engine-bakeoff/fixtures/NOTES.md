# fixtures

Build: `cd fixtures && /usr/local/go/bin/go build -o fixtures .` (stdlib only, hand-written bencode).

Run: `fixtures/fixtures gen --out DIR --count 3 --piece-len 4194304 --pieces 1024 --announce http://127.0.0.1:7070/PK/announce [--announce-list 'A1,A2;B1'] [--name-prefix fx]`

- Names are `<prefix>-NNNNNN` (default prefix `fx`). Each torrent is one 4 GiB sparse file (`ls -ls` shows 0 blocks). Generating 3 takes ~15 ms.
- `private=1`; `announce` is the first announce-list URL when `--announce-list` is given.
- `manifest.json` paths are absolute.
