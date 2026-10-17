# Browser

The browser is `playwright cli` from web/'s pinned Playwright (`web/node_modules/.bin/playwright cli`, `$PW cli` below), run through `fakeworld exec` so each action is a journaled step. It keeps a long-lived browser per named session, so a recipe fills a form in one step and submits it in the next.

## Sessions

- One session per identity and shell, named `<runId>-<identity>-<shell>`: `-s={runId}-admin-desktop`, `-s={runId}-admin-phone`, `-s={runId}-member-phone`. The run id keeps two agents in one worktree apart. An identity is `admin`, the Scenario's Admin; `member`, its first Member who is no Admin (Grace in `home`); or any Member's email from the manifest's `members`.
- `fakeworld browser $RUN` writes `browser/desktop/playwright-cli.json` and `browser/phone/playwright-cli.json` from the manifest's `browser` block: Chromium with host-resolver rules that map every `<node>.localhost` name to its address, the shell's viewport and device, and `browser/output/` as the output dir. Open each session with `open --config={runDir}/browser/<shell>/playwright-cli.json`. Each config's `initScript`, `browser/startup-played.js`, opens every tab past the startup sequence; the skill's Drive says how to see it.
- Sign in once per identity and shell through the real page: `bun web/harness/bin/sign-in.ts {runDir} admin desktop` writes `auth/admin-desktop.json`, and `state-load {runDir}/auth/admin-desktop.json` loads it into the open session. It declines the passkey prompt, and waits for the Library for an Admin and the portal's Home for a Member. Reuse the session afterwards.
- `member-phone` is the Member on the phone preset, the portal's tab bar and the keypad:

  ```sh
  $FW exec $RUN -- bun web/harness/bin/sign-in.ts {runDir} member phone
  $FW exec $RUN -- $PW cli -s={runId}-member-phone open --config={runDir}/browser/phone/playwright-cli.json
  $FW exec $RUN -- $PW cli -s={runId}-member-phone state-load {runDir}/auth/member-phone.json
  $FW exec $RUN -- $PW cli -s={runId}-member-phone goto {nodes.home.browserUrl}/portal
  ```
- Use `{nodes.<n>.browserUrl}` (`http://home.localhost:<port>`), never the bare IP: Chromium treats `*.localhost` as a secure context, which passkeys need.
- `$PW cli list` shows the open sessions. Close yours with `-s=<session> close`. A headless session closes itself after an hour idle; `open` it again and `state-load` again.

## Locators and waits

- Target by role and accessible name: `click 'getByRole("link", { name: "Activity" })'`, `fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"`. Write the locator with double quotes inside single quotes, which keeps the journal and `summary.md` readable.
- Each command prints a snapshot of the page; `--raw snapshot 'getByRole("main")'` prints only that subtree.
- Wait for an outcome with `run-code`, which runs a Playwright function against the page: `run-code 'async page => { await page.getByRole("list", { name: "Grabs" }).getByText("Imported").waitFor({ timeout: 60000 }); }'`. Give every wait an explicit timeout.
- A PIN field takes `type` after a `click` on it, so its input events fire per digit as a user's would. On the phone preset a new PIN sits above the `Keypad` group; tap its digit buttons in a `run-code` loop, as [signup links](../features/signup-links.md) does.
- Passkeys need Chromium's virtual authenticator, added to the session's page over CDP with `run-code`, as [passkeys](../features/passkeys.md) does. It lives as long as the session's browser.

## Proof files

- Before and after each proof action: `snapshot --filename={runDir}/proof/<f>/<s>/<name>.aria.yml` and `screenshot --filename={runDir}/proof/<f>/<s>/<name>.png`. A relative `--filename` lands in the current directory, the worktree root, so always pass `{runDir}/proof/...`.
- Run those steps with `--proof <f>/<s>` too, so their output sits beside the files.

## Traces

`tracing-start` begins a trace and `tracing-stop` ends it. Each pair writes its own `trace-<ms>.trace` and `.network` under `$RUN/browser/output/traces/`, sharing one `resources/` dir, so wrap each recipe in one pair. Inspect one with `$PW trace open <file>` and then `$PW trace actions`.

## Video

Only for a recipe that proves motion. `video-start {runDir}/proof/<f>/<s>/<name>.webm` before the motion and `video-stop` after, then encode H.264 with the run's ffmpeg, which Safari and iOS play:

```sh
$FW exec $RUN --proof <f>/<s> -- {runDir}/bin/ffmpeg -loglevel error -i {runDir}/proof/<f>/<s>/<name>.webm -c:v libx264 -pix_fmt yuv420p -movflags +faststart {runDir}/proof/<f>/<s>/<name>.mp4
```

Recording needs Playwright's own ffmpeg, which `bun run --cwd web e2e:install` installs; the run's ffmpeg doesn't stand in for it. A `video-start` that fails leaves the session reporting `Video recording has already been started`, so close the session and open it again.

Playwright's bundled skill tells you to attach media to PRs with `gh --attach`. That doesn't apply here: media goes to the maintainer in chat or on one private Artifact, never into the PR.
