# 18: Legacy removal, motion audit and prototype decision

Spec: `docs/admin-native-overhaul-spec.md` (Removal of legacy, stories 89–94, Further Notes)

**What to build:** With every page ported, the admin app has no trace of the old presentation layer: the old page header, the card-based settings sections, the header search bar's mobile fallback, hover-glow and card-glow utilities and their usages, and every `transition-all` are gone; glow remains only on active download progress and the completion flash. The Cinematic and Console prototype variants are deleted. Motion across the app is audited against the policy (entrances on first mount only, under 300 ms, exits faster than entrances, only transform and opacity except the progress bar, reduced motion and reduced transparency honoured, light theme intact). Screenshot baselines are recorded for the key screens (Dashboard, Library grid, Detail, Activity, Settings list) in both projects and both themes. The real-hardware checklist (sticky hover, tap delay, safe areas, keyboard, sheet drag) is run and recorded.

This ticket also makes the decision the spec defers: whether the Native prototype and its harness ship to `main` or are dropped now that the production shell is complete. Record the decision and its reasoning under the spec's Comments heading.

**Blocked by:** 08 (Library actions), 10 (Search tab), 12 (Settings forms), 13 (Long-form settings), 14 (System and session actions), 15 (Requests admin), 16 (Missing and History), 17 (Calendar and Manual Import)

**Status:** ready-for-agent

- [ ] No references remain to the old page header, settings list section, add placeholder card, media status badge, module card components or the header search mobile fallback
- [ ] `transition-all` does not appear in the frontend source; hover-glow and card-glow utilities are removed and glow is used only by download progress and the completion flash
- [ ] The Cinematic and Console prototype variants and any code only they used are deleted; the Native prototype's fate is decided and recorded in the spec's Comments
- [ ] Reduced-motion run of the full suite passes; reduced-transparency renders solid bars on the key screens; the light theme renders every key screen with correct status colours
- [ ] Returning to a tab or navigating back does not replay list entrance animation
- [ ] Screenshot baselines exist for the five key screens in both projects and both themes and the comparison passes
- [ ] The real-hardware checklist is completed on at least one iOS and one Android device and the results appended to the spec's Comments
- [ ] `tsc -b`, ESLint and the full Playwright suite pass in CI
