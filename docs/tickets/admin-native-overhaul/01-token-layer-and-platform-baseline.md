# 01: Token layer, platform baseline and unified status rendering

Spec: `docs/admin-native-overhaul-spec.md` (stories 58, 67, 80–85, 88, 92–93, 95–96)

**What to build:** The admin app gets the Native design system's token layer as its app-wide foundation, the phone platform baseline, and one way to paint status everywhere. After this ticket, an admin opening any existing page on a phone sees no page zoom when focusing an input, no grey tap flash, a browser status bar that matches the app in both colour schemes, and every media status (Available, Missing, Downloading, Upgradable, Unreleased, Failed) rendered as the same tinted dot-plus-text pill wherever it appears. Nothing else about the layout changes yet.

Concretely:

- The mobile token file from the prototype becomes the production token layer imported by the global stylesheet, with the `m-` prefixes dropped: motion easings (strong ease-out, strong ease-in-out, drawer) and durations (press 120, fast 160, base 220, sheet 320); the six-step type scale (display, heading, title, body, footnote, caption) with paired leading and tracking; touch geometry (44 px tap, 52 px large tap, 16 px screen gutter, 56 px tab bar, sheet/card/thumb radii); the six status hues; bar, heavy and chip materials; safe-area variables resolving from `env()`; the press, press-row, press-dim, scroll and one-shot entrance utilities; a tabular-numbers utility.
- Light-theme values are added for the status hues and materials so the light theme keeps working; reduced-transparency makes materials solid; reduced-motion removes press scale and swaps entrance motion for short fades.
- The platform baseline moves into the production HTML entry and global stylesheet: `viewport-fit=cover`, `interactive-widget=resizes-content`, one `theme-color` per colour scheme, transparent tap highlight, `text-size-adjust`, overscroll containment on the root and on inner scrollers, 16 px minimum input font, `touch-action: manipulation` and no text selection on controls (never on content), hover styles gated behind a fine pointer.
- `StatusDot` and `StatusPill` are added as the only components that paint status hues. `MediaStatusBadge` is replaced by `StatusPill` at every call site and deleted. The media-type colours (movie orange, series blue) are untouched.
- A single viewport hook exposes the shell decision (phone below 768 px, wide at and above) for later tickets; nothing consumes it yet beyond a smoke usage.

No routes, hooks, stores or backend change. Existing pages keep their current layout; only the tokens they can reach and the status rendering change.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] The token layer is imported globally and the design-system doc's tables (motion, type, touch, status, materials) map one-to-one to defined CSS custom properties and utilities without the `m-` prefix
- [ ] Light theme defines every status hue and material token; switching theme in the app shows correct status colours and legible bars in both schemes
- [ ] On a phone-sized viewport, focusing any text input in the app does not change the page scale; tapping a button shows no grey highlight; long-pressing a control does not select text
- [x] The document declares a theme colour per colour scheme and `viewport-fit=cover`
- [x] `prefers-reduced-transparency` renders material surfaces solid; `prefers-reduced-motion` removes press scale and entrance slides
- [x] `MediaStatusBadge` no longer exists; every former call site renders `StatusPill` (prominent) or `StatusDot` (dense) and each status uses exactly one hue across the app
- [x] A viewport hook returns the current shell (`phone` or `wide`) and updates on resize across the 768 px boundary
- [x] `tsc -b` and ESLint pass

## Comments

Light-theme status hues and materials are defined on `:root` (dark overrides on `.dark`), and the reduced-motion / reduced-transparency rules are in `web/src/tokens.css`. I did not run the authenticated app to watch a live theme switch, and I did not exercise iOS Safari input-zoom, tap-highlight, or control text-select on a phone. Those two boxes stay open for hardware / running-app checks.

`useViewport` is smoked on `RootLayout` via `data-shell` only. The prototype under `web/src/prototypes/mobile/` was not modified.
