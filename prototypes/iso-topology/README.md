# Isometric Nodes topology (prototype)

This is a throwaway prototype. It never merges. Jackson picked **variant B · Sites as dioramas** as the reference for the Nodes canvas in [Nodes, Pairing and Roles](https://github.com/jatassi/SlipStream/issues/140).

Open `index.html` in a browser. The yellow bar switches between the two variants and four states (calm, busy, relay warning, media server offline). `b/index.html` is variant B on its own. It is a standalone page that also reads `?state=` or `#state`.

## What B establishes

- Each Node is its own site tile. Each resource is a small machine standing on that tile, and its motion is the metric.
  - Storage is a four-drawer cabinet, one drawer per drive. A drawer slides out while its drive is read, and its gauge shows used/size.
  - The transcoder is a roller press. A wide 4K strip goes in and a narrow 1080p strip comes out, at the session's fps, with one lane per session.
  - The torrent engine is a piece board. Pieces fly from peers down a chute into the active download's panel. Seeding sends pieces back out.
  - The Relay Node is a switchboard tower with one socket per fronted domain, a TLS lock and status lamps. It feeds the viewer devices.
- A glass tube carries the tunnel traffic between tiles. Capsule rate and speed come from throughput and latency. A's open cable tray reads traffic more clearly and is worth grafting on.
- Every number and motion comes from one pure model (`model.mjs`, `sample(state, t)`). State changes ease in place.
- Lit Console palette (`theme.css`): signal colours appear only on status lamps, and live traffic is white with the blue glow.

Known rough edges are listed in [the thread](https://claude.ai/code/project/chan_011rigPaLozCw4zXgXuGjkRX?thread=cmsg_011rigPaLozCw4zXgXuGjkRX8Yw73B582cQJ1Y98ptsW8B). Flying pieces ignore selection dimming, the top-right of the stage is empty, and each page also has its own state buttons.

## Rebuilding

The figures are drawn with the [anatomy](https://github.com/wheresryan22/anatomy) skill's kit at commit `689dcf9`. `a/build.mjs` and `b/build.mjs` import it from `../../anatomy/src/`. Point those imports at the vendored copy before rebuilding. `BRIEF.md` is the brief the two builders worked from.
