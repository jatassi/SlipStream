# 04: Push navigation for detail and secondary screens

Spec: `docs/admin-native-overhaul-spec.md` (stories 4–5, 10, 20, 89–91)

**What to build:** On a phone, opening a detail screen, an add flow, a settings sub-page, requests admin, system or any `/more` destination pushes the screen in from the right over the tab root; the layer beneath shifts and dims; going back slides it out faster than it came in. The back button names the screen the admin came from ("Library", "More", "Settings"). On a wide screen the same routes render instantly in the content column with a back button and no motion. The existing movie and series detail pages, unchanged inside, are the first screens to demonstrate the push.

Concretely:

- Pushed routes are identified from the route tree (detail routes, add routes, settings sub-routes, requests admin, system, `/more` children). Push-in is 260 ms and pop is 200 ms on the strong ease-out curve, animating only transform and opacity; the underlying tab root stays mounted.
- The back label derives from the route hierarchy: the parent list for detail routes, "More" for `/more` children, the settings section name for settings leaves. Browser back and the in-app back button behave identically.
- Detail screens use the compact bar variant of `Screen`: transparent over the hero and translucent once the hero scrolls away.
- Keyboard-initiated navigation and tab switches never animate. Reduced motion replaces the slide with a 160 ms cross-fade. The wide shell never animates route changes.
- The movie and series detail pages adopt `Screen` in compact mode with the existing hero and content beneath; their loaders and prefetches are unchanged.

**Blocked by:** 03 (Two-shell layout)

**Status:** done

- [x] Phone project: tapping a library item pushes the detail screen; the back button reads the library's name; activating back returns to the list at the same scroll position
- [x] Phone project: opening a settings sub-page from More shows a back button reading "More"; browser back produces the same result as the button
- [x] Phone project: on a detail screen the top bar is transparent over the hero and shows the compact title once the hero is scrolled out
- [x] Wide project: the same navigations render instantly with a back control and no intermediate animation frame visible
- [x] Reduced-motion run: pushed screens appear and disappear without slide transforms and the flows above still pass
- [x] No route change on either shell exceeds 320 ms of motion; only transform and opacity are animated by the push

## Comments

Back from `/movies/$id` and `/series/$id` (and the add routes) reads **Library**. Back from other pushed admin routes, including today's settings leaves, reads **More**. Settings section index routes still redirect to the first leaf, so a section-name back target would loop; ticket 11 introduces real section pages and the leaf back labels.

In-app back calls `history.back()`, so it matches browser back. Pop keeps a DOM snapshot of the outgoing layer because the router Outlet swaps immediately; the tab root under it stays mounted.

Motion is CSS-only: push-in 260 ms, pop 200 ms, reduced-motion fade 160 ms, behind-shift `--dur-base` (220 ms), all on `--ease-out-strong`, transform and opacity only (dim is a black overlay's opacity, not `filter`). Keyboard input and tab switches set `data-instant` and skip animation. Tests assert settled UI, not durations.

Library scroll after pop allows under 8 px of drift from the behind-layer transform; the list does not jump to the top.
