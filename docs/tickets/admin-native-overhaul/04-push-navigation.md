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

**Status:** ready-for-agent

- [ ] Phone project: tapping a library item pushes the detail screen; the back button reads the library's name; activating back returns to the list at the same scroll position
- [ ] Phone project: opening a settings sub-page from More shows a back button reading "More"; browser back produces the same result as the button
- [ ] Phone project: on a detail screen the top bar is transparent over the hero and shows the compact title once the hero is scrolled out
- [ ] Wide project: the same navigations render instantly with a back control and no intermediate animation frame visible
- [ ] Reduced-motion run: pushed screens appear and disappear without slide transforms and the flows above still pass
- [ ] No route change on either shell exceeds 320 ms of motion; only transform and opacity are animated by the push
