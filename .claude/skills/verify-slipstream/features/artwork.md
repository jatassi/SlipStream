# Artwork

A movie's poster, backdrop, title treatment and studio logo, and a person's photo, are served by SlipStream at an artwork URL the Work API hands out. The Controller fetches each image once per size through the Metadata Role, keeps it in its artwork store, prefetches the poster and backdrop when a Work is added, and refuses any token it didn't issue.

## Sub-features

- `artwork-urls` `POST /api/v1/works` and `GET /api/v1/works/{id}` answer an `artwork` list of `{kind, url}`, one per kind the movie has, each URL `/api/v1/artwork/<token>` without a size (P-059).
- `artwork-search` `GET /api/v1/content-types/movie/catalog?query=quiet+orchard` answers each result's `posterUrl`, and `GET /api/v1/content-types/movie/catalog/9000001` previews the Work with its `artwork`, before anything is added; the URLs are the ones the add answers.
- `artwork-sizes` a signed-in Account's GET of an artwork URL at `size=small`, `medium`, `large` and `original` answers 200 `image/jpeg` with `Cache-Control: private, max-age=31536000, immutable`, a larger image at each size (P-059).
- `artwork-prefetch` with every image-cdn request failing after the add, the backdrop at `large` still answers 200 from the store, and at `small` answers 404 `artwork_unavailable` with `Cache-Control: no-store`.
- `artwork-missing` once image-cdn answers again, the backdrop at `small` still answers 404 until `timings.metadata.artwork_retry` (5 minutes) passes, and a size never asked for answers 200.
- `artwork-forged` a token SlipStream didn't issue answers 404 `artwork_unavailable` with `detail` `SlipStream didn't issue that artwork token` and `Cache-Control: no-store`.
- `artwork-worker` with Metadata moved to seedbox alone and home's image-cdn front reset, an image never asked for answers 200, and `events.jsonl` holds a `front.open` on seedbox's `image-cdn` front: the Controller fetches through the Metadata holder.
- `artwork-account` the API key alone gets 403 `needs_account`: the route serves signed-in Accounts.
- `artwork-work-page` the Work page requests its backdrop at `size=large`, its poster and title treatment at `size=medium`, and the studio logo and credit photos at `size=small`, each answering 200 (P-059).

## How to get to it (user POV)

- Desktop and phone shells: the Work page shows the backdrop at `large`, the poster and title treatment at `medium`, and the studio logo and people's photos at `small` (`work-page.md`). Opening an artwork URL with `?size=` in a signed-in tab shows the image.
- API: `GET /api/v1/works/{id}`, `GET /api/v1/library` (`posterUrl`), `GET /api/v1/content-types/{contentType}/catalog?query=` (`posterUrl`), then `GET /api/v1/artwork/{token}?size=small|medium|large|original` with the session cookie.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline on a fresh `home` run, with `{runId}-admin-desktop` signed in.
- `one.js`, a `run-code` function that fetches `URL_HERE` from inside the signed-in page, so the session cookie goes with it, and prints the size, status, type, cache header and the byte count or problem:

  ```js
  async page => page.evaluate(async (url) => {
    const r = await fetch(url);
    const b = await r.arrayBuffer();
    return [url.split("?")[1], r.status, r.headers.get("content-type"), r.headers.get("cache-control"), r.status === 200 ? b.byteLength : new TextDecoder().decode(b)].join(" | ");
  }, URL_HERE)
  ```

  `one() { sed "s#URL_HERE#\"$1\"#" one.js; }` fills it in.

Steps:

- **Search and preview.** The result's `posterUrl` and the preview's `work.artwork` carry artwork URLs.

  ```sh
  $FW exec $RUN --proof artwork/artwork-search -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=quiet+orchard'
  $FW exec $RUN --proof artwork/artwork-search -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000001'
  ```

- **Add the movie and read its URLs.** The reply's `work.artwork` lists `backdrop`, `poster` and `title_treatment`. Set `P` and `B` to the poster's and backdrop's `url`.

  ```sh
  $FW exec $RUN --proof artwork/artwork-urls -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001"}' '{nodes.home.url}/api/v1/works'
  ```

- **Each size.** Each answers 200 `image/jpeg`, immutable, and grows with the size.

  ```sh
  for size in small medium large original; do $FW exec $RUN --proof artwork/artwork-sizes -- $PW cli -s={runId}-admin-desktop run-code "$(one "$P?size=$size")"; done
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}$P?size=large"
  $FW exec $RUN --proof artwork/artwork-sizes -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/artwork/artwork-sizes/poster-large.png
  ```

- **Prefetched before it was asked for.** With image-cdn down, the backdrop at `large`, which the add prefetched, answers 200; at `small` it answers 404 no-store.

  ```sh
  $FW fault set $RUN cdn-down image-cdn status=503
  $FW exec $RUN --proof artwork/artwork-prefetch -- $PW cli -s={runId}-admin-desktop run-code "$(one "$B?size=large")"
  $FW exec $RUN --proof artwork/artwork-prefetch -- $PW cli -s={runId}-admin-desktop run-code "$(one "$B?size=small")"
  ```

- **A miss waits for its retry.** With image-cdn back, the backdrop at `small` still answers 404; at `medium` it answers 200.

  ```sh
  $FW fault clear $RUN cdn-down
  $FW exec $RUN --proof artwork/artwork-missing -- $PW cli -s={runId}-admin-desktop run-code "$(one "$B?size=small")"
  $FW exec $RUN --proof artwork/artwork-missing -- $PW cli -s={runId}-admin-desktop run-code "$(one "$B?size=medium")"
  ```

- **A forged token.** Flip the low bit of the poster token's last character within the base64url alphabet, which base64 leaves unused, and graft its signature onto another path; both answer 404 `artwork_unavailable` no-store.

  ```sh
  a=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_
  i=$(( $(expr index "$a" "${P: -1}") - 1 ))
  $FW exec $RUN --proof artwork/artwork-forged -- $PW cli -s={runId}-admin-desktop run-code "$(one "${P%?}${a:$(( i ^ 1 )):1}?size=medium")"
  $FW exec $RUN --proof artwork/artwork-forged -- $PW cli -s={runId}-admin-desktop run-code "$(one "/api/v1/artwork/AXRtZGItaW1hZ2VzAHBvc3RlcgAvYWJjZGVmLmpwZw.${P##*.}?size=medium")"
  ```

- **Fetched through a Worker.** Move Metadata to seedbox alone, reset home's image-cdn front, and ask for the backdrop at `original`, which no step asked for yet: it answers 200. The last `front.open` on `image-cdn` names seedbox.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  SET='[{"nodeId":"{nodes.home.nodeId}","roles":["storage","converting","subtitles","media-server-link","notifying","serving"]},{"nodeId":"{nodes.seedbox.nodeId}","roles":["searching","downloading","metadata"]}]'
  $FW exec $RUN --proof artwork/artwork-worker -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":$SET}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW fault set $RUN cdn-home image-cdn@home reset
  $FW exec $RUN --proof artwork/artwork-worker -- $PW cli -s={runId}-admin-desktop run-code "$(one "$B?size=original")"
  jq -c 'select(.kind == "front.open" and .data.target == "image-cdn")' $RUN/events.jsonl | tail -1
  ```

- **The API key alone.**

  ```sh
  $FW exec $RUN --proof artwork/artwork-account -- curl -sS -i -H 'X-Api-Key: {apiKey}' "{nodes.home.url}$P?size=medium"
  ```

- **On the Work page.** Open the Work and list each image the page loaded with its size and status: the backdrop at `large`, the poster and title treatment at `medium`, 200 each.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/<id from the add>"
  $FW exec $RUN --proof artwork/artwork-work-page -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.evaluate(() => performance.getEntriesByType("resource").filter((e) => e.name.includes("/api/v1/artwork/")).map((e) => [new URL(e.name).search, e.responseStatus]))'
  $FW exec $RUN --proof artwork/artwork-work-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/artwork/artwork-work-page/work.png
  ```

## Gotchas

- The route is `api.Account`: curl with the API key gets 403. Fetch from inside the signed-in page; `page.request` resolves `home.localhost` outside the browser and fails.
- An artwork URL carries no size; the route requires `?size=`.
- The add prefetches the poster at `medium` and the backdrop at `large` in the background. Set the image-cdn fault a few seconds after the add, or the prefetch itself fails and is marked missing.
- A miss stays a miss for `timings.metadata.artwork_retry`; a recipe that wants the image afterwards asks for another size or waits it out.
- After the Worker step, doctor fails "the cluster matches the Scenario": Metadata sits on seedbox and the `cdn-home` fault stays set. Drive that step last, or on a run used for nothing else.
- The flipped-character forgery only proves something when the flip stays within the alphabet and touches only the unused bits, as the recipe's index arithmetic does; a token spelled any other way than issued is refused.
