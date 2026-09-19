# 02: Playwright end-to-end harness in developer mode

Spec: `docs/admin-native-overhaul-spec.md` (Testing Decisions)

**What to build:** The single test seam the spec proposes: browser-level end-to-end tests with Playwright that run against the app started in developer mode (separate database, mock metadata, indexer, download client and notifications, so no network is needed). Two projects run the same suite: a phone profile (390 × 844, touch, device scale factor 2) and a wide profile (1440 × 900, mouse). A third opt-in run executes the shell tests with reduced motion forced.

After this ticket a developer runs one script from the frontend package and gets a green suite in both profiles locally and in CI. The suite starts small and proves the harness with tests that assert what the admin sees: the Dashboard loads with developer-mode data, navigating to a library list shows items, focusing the search input on the phone profile does not change the viewport scale, and every status pill on the Dashboard has visible text. Every later ticket adds its coverage to this suite.

Conventions this ticket establishes for the rest of the effort:

- Tests start from a URL, locate by role and visible text, and assert visible outcomes. They never assert on class names, DOM structure or animation timing.
- Tests tap on the phone project and click on the wide project through one shared helper; tests that exist in only one shell are scoped to that project.
- Console errors are captured and fail the test.
- Screenshot comparison is available but opt-in, intended for a handful of key screens in both projects and both themes, tolerant enough to catch layout regressions rather than pin pixels.
- Lint and `tsc -b` run before the browser suite in CI.

The spec asks for this seam to be confirmed before implementation begins; it is the only ticket in the effort that adds tooling (a dev dependency, a config, a script and a CI step).

**Blocked by:** 01 (Token layer, platform baseline and unified status rendering)

**Status:** ready-for-agent

- [ ] A Playwright config defines `phone` and `wide` projects with the dimensions, touch and scale settings above, and a reduced-motion variant of the shell tests
- [ ] A frontend script boots the backend in developer mode and the frontend, waits for readiness, and runs the suite; a second script runs it headed for debugging
- [ ] Shared helpers exist for "activate" (tap vs click per project), authenticated session setup, and console-error capture
- [ ] Smoke tests pass in both projects: Dashboard renders developer-mode data; a library list route shows items; on `phone`, focusing the search input leaves the visual viewport scale unchanged; status pills on the Dashboard have accessible text
- [ ] A CI workflow runs lint, `tsc -b` and the suite on pull requests and uploads traces on failure
- [ ] Screenshot comparison is wired but opt-in (skipped unless explicitly enabled) with baseline storage documented in the frontend docs
