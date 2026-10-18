# Shell

Every signed-in Admin page sits in one of two shells. In a desktop browser the desktop shell has a sidebar of every destination and a top bar with the ⌘K capsule and the page's main action. Below 768px the phone shell has a large title, a bottom tab bar and a More sheet. Both carry the command palette, toasts, route titles, errors that keep the shell standing, and the startup sequence that plays once per new tab. A Member, or an Admin who opens `/portal`, sees the portal shell instead.

## Sub-features

- `shell-sidebar` the desktop sidebar lists Library with one child per enabled Content Type (`Movies` in `home`), Activity, a separator, Nodes and Tasks, with the brand link `SlipStream` above and the `Account` button below (P-001, P-002).
- `shell-current-page` the current row is the only link with `aria-current="page"` (the brand link `SlipStream` never is), and the page's title names it; on phone a Work and Add go back to `Movies` (P-001).
- `shell-collapse` `Collapse sidebar` shrinks the sidebar to a 60px rail whose icons name themselves in a tooltip on hover; the choice survives a reload, and a window below 1100px starts collapsed (P-001). The toggle and the brand mark share the top row and keep their places in both widths, the toggle at x 14 to 46 inside the rail and the mark at x 74 to 98 just past its edge, while the wordmark wipes in and out with the sidebar.
- `shell-activity-count` a moving Grab shows `Activity 1 live` in the sidebar and `Activity 1` on the phone's tab, and the count leaves when the Grab is imported.
- `shell-page-action` a page's main action sits at the top bar's right: `Add` opens the menu `Add a Work` on the Library, and `Add a Movie` is a link on the Movies library.
- `shell-top-bar-slots` the top bar's right side runs the connection line, the running-tasks capsule and the page's action, in that order; the capsule takes no room while no run is shown, and [Tasks](./tasks.md) drives it (P-007).
- `shell-account-menu` `Account` at the sidebar's foot opens a menu with `Open the portal`, the `Startup sequence` checkbox, `Restart SlipStream…` and `Sign out`; `Restart SlipStream…` asks `Restart SlipStream?` (P-010).
- `shell-tab-bar` the phone's tab bar `Tabs` holds `Library`, `Activity`, `Search` and `More` (P-004).
- `shell-more-sheet` `More` opens the sheet `More` with each enabled Content Type's library (`Movies`), `Nodes`, `Tasks`, the Admin's name, `Open the portal`, `Restart SlipStream…`, `Sign out`, the `Startup sequence` switch, `SlipStream <version>` and `Cancel` (P-005).
- `shell-phone-pages` a top-level page opens on its large title with `Search` and `Account` beside it; a Work page is pushed, with a back link labelled after its library (`Movies`) that pops back to it; a menu opens as an action sheet (P-016).
- `shell-palette` Ctrl+K (⌘K on macOS) or the capsule opens the palette `Search Works or run a command`; typing part of a Work's name lists it under Works, Enter opens it, and Esc returns focus where it was.
- `shell-palette-reopen` in the desktop shell, the shortcut pressed again straight after Esc, while the palette is still closing, reopens it with focus in the search field, and every key typed straight after lands in the query.
- `shell-palette-commands` with no query the palette lists Go to each nav row and Content Type, Go to each section of the browser's own Node, `Add a Movie`, `Toggle sidebar` (desktop shell only), `Restart SlipStream…` and `Sign out`; `Restart SlipStream…` asks `Restart SlipStream?` in both shells.
- `shell-palette-empty` an unmatched query reads `Nothing matches "volcano". Try a Work's name, or a command such as Add a Movie.`
- `shell-palette-phone` the `Search` tab opens the palette full screen with `Cancel`.
- `shell-titles` the tab is titled `<page> · SlipStream`, as `Activity · SlipStream` and `The Quiet Orchard · SlipStream` (P-014).
- `shell-toast` adding a Work shows the toast `Added The Quiet Orchard`, top right in the desktop shell and top centre on phone (P-013).
- `shell-route-error` with the Controller down, a page that fails to load says `SlipStream isn't answering. Check that it's running, then try again.` inside the standing shell, `Details` shows the message, and `Try again` loads the page once the Controller is back (P-015).
- `shell-not-found` a dead address such as `/nowhere` reads `There's nothing at this address.` with `Go to Library`, inside the shell.
- `shell-brand` the sidebar carries the Two Nodes mark and wordmark, and the page links the mark as its favicon with `theme-color` `#0c0d0e` (P-017).
- `shell-portal` `/portal` shows the portal shell: in a desktop browser a top nav `Portal` with Home, Search, Requests and Account and an `Account` menu; on phone a tab bar `Tabs` with the same four. An Admin opens it from `Open the portal` and returns through `Back to Library` in the portal's `Account` menu or on its Account page (story 40).
- `startup-plays` a new tab plays the startup sequence, an image named `SlipStream`, over the page while it loads.
- `startup-skip` any key, click or tap ends it at once, and a reload in the same tab doesn't replay it.
- `startup-off` unchecking `Startup sequence` stores `slipstream.startupOff` in this browser, and the next new tab opens with no sequence.

## How to get to it (user POV)

- Desktop shell: the sidebar's rows (`Library`, `Movies`, `Activity`, `Nodes`, `Tasks`), the brand link `SlipStream`, `Collapse sidebar` or `Expand sidebar`, and the `Account` button at its foot, which opens the menu with `Open the portal`, `Startup sequence`, `Restart SlipStream…` and `Sign out`.
- Desktop shell: the top bar's capsule `Search Works or run a command` showing `⌘K` on macOS and `Ctrl K` elsewhere, and at the right the connection line while reconnecting, the running-tasks capsule while a run is shown, and the page's action (`Add` on the Library).
- Desktop shell: Ctrl+K, or ⌘K on macOS, from any page opens the palette; its commands are `Go to Library`, `Go to Movies`, `Go to Activity`, `Go to Nodes`, `Go to Tasks`, `Go to Overview`, `Go to Web and API`, `Go to Logs`, `Go to Log settings`, `Add a Movie`, `Toggle sidebar`, `Restart SlipStream…` and `Sign out`.
- Phone shell: the tab bar's `Library` and `Activity` tabs, its `Search` tab (the palette, full screen) and its `More` tab (the More sheet, holding `Movies`, `Nodes`, `Tasks`, `Open the portal`, `Restart SlipStream…`, `Sign out` and `Startup sequence`).
- Phone shell: the large title's `Search` button opens the palette and its `Account` button opens the More sheet; a pushed page's back link names the page above it (a Work's library, or the tab it came from).
- Both shells: `/` redirects to `/library`, and any address the app doesn't know shows the not-found page.
- Both shells: the portal at `/portal`, through `Open the portal` in the desktop `Account` menu or the More sheet; the portal leads back through `Back to Library`.
- The startup sequence: any new tab or window on the Controller, signed in or not. Verify sessions open past it; the recipe below opens one that plays it.
- API: `GET /api/v1/counts` gives the Activity count, `GET /api/v1/content-types` the Library's children, and `GET /api/v1/system` the More sheet's version.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in on the Library, which lists nothing yet.
- `{runId}-admin-phone` open the same way with `phone` in place of `desktop` (the skill's Drive).
- The route-error steps kill and restart the Controller, so run them last.

Steps:

- **See the desktop shell.** The sidebar lists `Library`, `Movies`, `Activity`, `Nodes` and `Tasks`, the Library row is current, and the title reads `Library · SlipStream`.

  ```sh
  $FW exec $RUN --proof shell/shell-sidebar -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/shell/shell-sidebar/library.aria.yml
  $FW exec $RUN --proof shell/shell-sidebar -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-sidebar/library.png
  $FW exec $RUN --proof shell/shell-current-page -- $PW cli -s={runId}-admin-desktop eval 'document.title + " " + document.querySelector("nav[aria-label=Main] [aria-current=page]").textContent'
  ```

- **Walk the sidebar.** Choose `Activity`, then `Nodes`. Each title follows: `Activity · SlipStream`, then `home · SlipStream`, the Node's name.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity" })'
  $FW exec $RUN --proof shell/shell-titles -- $PW cli -s={runId}-admin-desktop eval 'document.title'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Nodes" })'
  $FW exec $RUN --proof shell/shell-titles -- $PW cli -s={runId}-admin-desktop eval 'document.title'
  ```

- **Collapse to the rail.** Choose `Collapse sidebar`. The rows lose their words, hovering `Activity` shows its tooltip, and after a reload the button reads `Expand sidebar`. A window resized to 1000px wide and reloaded starts collapsed too.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Collapse sidebar" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity" }).hover(); await page.locator("[data-side=right]").getByText("Activity").waitFor({ timeout: 5000 }); }'
  $FW exec $RUN --proof shell/shell-collapse -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-collapse/rail.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN --proof shell/shell-collapse -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("button", { name: "Expand sidebar" })' --filename={runDir}/proof/shell/shell-collapse/after-reload.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Expand sidebar" })'
  ```

- **Open the palette.** Press Ctrl+K. The dialog `Search Works or run a command` lists the Commands group. Type `volcano` for the empty sentence, then press Escape: the dialog closes and focus is back on the element that had it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Control+k
  $FW exec $RUN --proof shell/shell-palette-commands -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/shell/shell-palette-commands/palette.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type volcano
  $FW exec $RUN --proof shell/shell-palette-empty -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/shell/shell-palette-empty/palette.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN --proof shell/shell-palette -- $PW cli -s={runId}-admin-desktop eval 'document.activeElement.getAttribute("aria-label")'
  ```

- **Reopen the palette while it closes.** Open the palette, press Escape and at once Ctrl+K again, then type `go to act`: the field holds the whole query and Enter opens Activity. One `run-code` presses the keys back to back, since separate `press` commands can land after the 220ms close ends.

  ```sh
  $FW exec $RUN --proof shell/shell-palette-reopen -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.keyboard.press("Control+k"); await page.getByRole("dialog").getByRole("combobox").waitFor(); await page.keyboard.press("Escape"); await page.keyboard.press("Control+k"); await page.keyboard.type("go to act"); return page.getByRole("dialog").getByRole("combobox").inputValue(); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Enter
  $FW exec $RUN --proof shell/shell-palette-reopen -- $PW cli -s={runId}-admin-desktop eval 'location.pathname'
  ```

- **Add a Work from the palette.** Choose the capsule, type `add a` and press Enter. `Add a Movie` opens at `/add?contentType=movie`; search and add The Quiet Orchard as in [add-a-movie.md](add-a-movie.md). The toast `Added The Quiet Orchard` shows top right, and the Work page is titled `The Quiet Orchard · SlipStream`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search Works or run a command" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type "add a"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Enter
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN --proof shell/shell-toast -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Added The Quiet Orchard").waitFor({ timeout: 5000 }); }'
  $FW exec $RUN --proof shell/shell-toast -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-toast/added.png
  $FW exec $RUN --proof shell/shell-titles -- $PW cli -s={runId}-admin-desktop eval 'document.title'
  ```

- **Open a Work from the palette.** From Activity, press Ctrl+K and type `orch`. The Works group lists `The Quiet Orchard Movie`; Enter opens its Work page.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Control+k
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type orch
  $FW exec $RUN --proof shell/shell-palette -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-palette/works.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Enter
  $FW exec $RUN --proof shell/shell-palette -- $PW cli -s={runId}-admin-desktop eval 'location.pathname'
  ```

- **Use the page action and the account menu.** On the Library, `Add` opens the menu `Add a Work` with `Add a Movie`. On `Movies` the action is the link `Add a Movie`. `Account` opens `Open the portal`, `Startup sequence`, `Restart SlipStream…` and `Sign out`; `Restart SlipStream…` asks `Restart SlipStream?`, and `Cancel` leaves the Node running.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN --proof shell/shell-page-action -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("menu")' --filename={runDir}/proof/shell/shell-page-action/add-menu.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Movies" })'
  $FW exec $RUN --proof shell/shell-page-action -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("banner")' --filename={runDir}/proof/shell/shell-page-action/movies.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN --proof shell/shell-account-menu -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("menu")' --filename={runDir}/proof/shell/shell-account-menu/menu.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Restart SlipStream…" })'
  $FW exec $RUN --proof shell/shell-account-menu -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/shell/shell-account-menu/confirm.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Restart SlipStream?" }).getByRole("button", { name: "Cancel" })'
  ```

- **Follow a dead link.** Go to `/nowhere`. The content reads `There's nothing at this address.` with `Go to Library`, and the sidebar stays.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/nowhere'
  $FW exec $RUN --proof shell/shell-not-found -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-not-found/nowhere.png
  ```

- **Read the brand.** The page links the mark as its icon and sets `theme-color`.

  ```sh
  $FW exec $RUN --proof shell/shell-brand -- curl -sS '{nodes.home.url}/library'
  ```

  The output holds `<meta name="theme-color" content="#0c0d0e" />`, `<link rel="icon" type="image/svg+xml" href="/assets/mark-….svg" />`, the apple-touch-icon and the manifest.

- **See the phone shell.** In `{runId}-admin-phone`, the Library opens on its large title with `Search`, `Account` and `Add`, and the tab bar `Tabs` lists `Library`, `Activity`, `Search` and `More`. `Add` opens the action sheet `Add a Work`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/'
  $FW exec $RUN --proof shell/shell-tab-bar -- $PW cli -s={runId}-admin-phone snapshot --filename={runDir}/proof/shell/shell-tab-bar/library.aria.yml
  $FW exec $RUN --proof shell/shell-tab-bar -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/shell/shell-tab-bar/library.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("main").getByRole("button", { name: "Add" })'
  $FW exec $RUN --proof shell/shell-phone-pages -- $PW cli -s={runId}-admin-phone snapshot 'getByRole("dialog", { name: "Add a Work" })' --filename={runDir}/proof/shell/shell-phone-pages/add-sheet.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone press Escape
  ```

- **Open More.** Choose `More`. The sheet lists `Movies`, `Nodes`, `Tasks`, `Ada Okafor`, `Open the portal`, `Restart SlipStream…`, `Sign out`, the switch `Startup sequence`, `SlipStream <version>` and `Cancel`. Choosing `Nodes` closes it and opens `/nodes/<id>`, titled `Nodes`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" })'
  $FW exec $RUN --proof shell/shell-more-sheet -- $PW cli -s={runId}-admin-phone snapshot 'getByRole("dialog", { name: "More" })' --filename={runDir}/proof/shell/shell-more-sheet/more.aria.yml
  $FW exec $RUN --proof shell/shell-more-sheet -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/shell/shell-more-sheet/more.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("dialog", { name: "More" }).getByRole("button", { name: "Nodes" })'
  ```

- **Search on phone.** The `Search` tab opens the palette full screen with `Cancel`; `Cancel` closes it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "Search" })'
  $FW exec $RUN --proof shell/shell-palette-phone -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/shell/shell-palette-phone/palette.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("dialog").getByRole("button", { name: "Cancel" })'
  ```

- **Push a page.** From the `Library` tab, open The Quiet Orchard. The nav bar shows the back link `Movies` and the heading `The Quiet Orchard`; the back link returns to `/library/movie`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: "Library" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("main").getByRole("link", { name: "The Quiet Orchard" })'
  $FW exec $RUN --proof shell/shell-phone-pages -- $PW cli -s={runId}-admin-phone snapshot 'getByRole("banner")' --filename={runDir}/proof/shell/shell-phone-pages/pushed.aria.yml
  $FW exec $RUN --proof shell/shell-phone-pages -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/shell/shell-phone-pages/pushed.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("banner").getByRole("link", { name: "Movies" })'
  ```

- **Watch the Activity count.** Hold the swarm and search the Work, as in [search-and-grab.md](search-and-grab.md). The sidebar row reads `Activity 1 live` and the phone tab `Activity 1`; once the throttle clears and the Grab imports, the row reads `Activity` again.

  ```sh
  $FW fault set $RUN swarm-hold seeders@seedbox throttle=16KiB/s
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("link", { name: "The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN --proof shell/shell-activity-count -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity 1 live" }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof shell/shell-activity-count -- $PW cli -s={runId}-admin-phone run-code 'async page => { await page.getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: "Activity 1" }).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof shell/shell-activity-count -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW fault clear $RUN swarm-hold
  $FW exec $RUN --proof shell/shell-activity-count -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity", exact: true }).waitFor({ timeout: 120000 }); }'
  ```

- **Open the portal.** On the desktop, `Account` › `Open the portal`; on the phone, `More` › `Open the portal`. The desktop shows the top nav `Portal` and the heading `Home`; the phone shows the tab bar `Tabs` with `Home`, `Search`, `Requests` and `Account`. `Back to Library` in the desktop's `Account` menu, and on the phone's `Account` tab, returns to `/library`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Open the portal" })'
  $FW exec $RUN --proof shell/shell-portal -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-portal/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("dialog", { name: "More" }).getByRole("button", { name: "Open the portal" })'
  $FW exec $RUN --proof shell/shell-portal -- $PW cli -s={runId}-admin-phone snapshot --filename={runDir}/proof/shell/shell-portal/phone.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Back to Library" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("main").getByRole("button", { name: "Back to Library" })'
  ```

- **Play the startup sequence.** Write a config without the init script, open a session from it, and go to the Library. The image `SlipStream` covers the page; Shift ends it, a reload doesn't replay it, and a new tab plays it again. Unchecking `Startup sequence` in `Account` stops the next new tab from playing it.

  ```sh
  jq 'del(.browser.initScript)' $RUN/browser/desktop/playwright-cli.json > $RUN/browser/desktop/startup-on.json
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop open --config={runDir}/browser/desktop/startup-on.json
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop state-load {runDir}/auth/admin-desktop.json
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN --proof shell/startup-plays -- $PW cli -s={runId}-startup-desktop run-code 'async page => { await page.getByRole("img", { name: "SlipStream" }).waitFor({ timeout: 5000 }); }'
  $FW exec $RUN --proof shell/startup-plays -- $PW cli -s={runId}-startup-desktop screenshot --filename={runDir}/proof/shell/startup-plays/sequence.png
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop press Shift
  $FW exec $RUN --proof shell/startup-skip -- $PW cli -s={runId}-startup-desktop run-code 'async page => { await page.getByRole("img", { name: "SlipStream" }).waitFor({ state: "detached", timeout: 5000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop reload
  $FW exec $RUN --proof shell/startup-skip -- $PW cli -s={runId}-startup-desktop eval 'document.querySelectorAll("[role=img][aria-label=SlipStream]").length'
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop click 'getByRole("menuitemcheckbox", { name: "Startup sequence" })'
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop tab-new '{nodes.home.browserUrl}/library'
  $FW exec $RUN --proof shell/startup-off -- $PW cli -s={runId}-startup-desktop run-code 'async page => { await page.getByRole("heading", { name: "Library" }).waitFor({ timeout: 10000 }); return page.getByRole("img", { name: "SlipStream" }).count(); }'
  $FW exec $RUN -- $PW cli -s={runId}-startup-desktop close
  ```

  The reload's count and the new tab's count are both `0`. Record the sequence as video under [references/browser.md](../references/browser.md) when the proof needs its motion.

- **Break a page.** Kill the Controller. The top bar shows `Reconnecting to SlipStream…`. Choose `Activity`: the content reads the unreachable sentence with `Try again` and `Details` while the sidebar stays. Start the Controller, wait for the line to go, and choose `Try again`: Activity loads.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW node kill $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Reconnecting to SlipStream…").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("alert").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof shell/shell-route-error -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/shell/shell-route-error/error.aria.yml
  $FW exec $RUN --proof shell/shell-route-error -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/shell/shell-route-error/error.png
  $FW node start $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Reconnecting to SlipStream…").waitFor({ state: "hidden", timeout: 30000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Try again" })'
  $FW exec $RUN --proof shell/shell-route-error -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("heading", { name: "Activity" }).waitFor({ timeout: 10000 }); }'
  ```

## Gotchas

- `getByRole("button", { name: "Search" })` also matches the capsule `Search Works or run a command` on desktop and the large title's and tab bar's `Search` on phone. A page's own Search, such as the Work page's, is its main action: scope it to `getByRole("banner")` with `exact: true` on desktop, where the top bar holds it, and to `getByRole("main")` on phone. The Add page's catalog Search sits in `getByRole("main")` in both shells.
- On phone, the back link and a tab can share a name (`Library` above `/add` with no Content Type). Scope them to `getByRole("banner")` or `getByRole("navigation", { name: "Tabs" })`.
- A nav row's accessible name carries its count, as in `Activity 1 live`. `name: "Activity"` matches either; `exact: true` matches only the row with no count.
- The rail's tooltips have no `tooltip` role; wait for the label's text in the `[data-side=right]` popup.
- Ctrl+K opens the palette on Linux and Windows; on macOS it's `Meta+k`, and the capsule shows `⌘K`.
- Verify sessions start past the startup sequence through the config's init script, so `startup-*` needs the separate config above. The init script runs on every navigation and every new tab of its session, so neither a reload nor `sessionstorage-delete` brings the sequence back there.
- The phone's large title collapses into the nav bar only once the page scrolls; a page shorter than the screen never collapses.
- Killing the Controller drops the seedbox's link; doctor's check 3 reads `node seedbox's link is down, not up` until it reconnects.
