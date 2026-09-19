# 11: Settings index and list pages as grouped rows

Spec: `docs/admin-native-overhaul-spec.md` (stories 63–65, 68)

**What to build:** Media, Download Pipeline and General settings each open as a grouped list of their sub-sections with a trailing count and warning summary (for example "Indexers 4 · 1 warning"). Root Folders, Quality Profiles, Version Slots, Indexers, Download Clients and Notifications list their items as rows with an icon tile, a primary line and a trailing detail, replacing today's card grid, with the Add action as a trailing plus in the top bar. Boolean settings on these pages are switches in rows with the label leading and the switch trailing. Every settings page adopts `Screen` with the correct back label. The card-based settings list section and the add placeholder card are deleted.

The forms behind Add and row taps keep opening exactly as they do today in this ticket; moving them to the sheet/dialog presenter is the next ticket.

**Blocked by:** 05 (Dashboard as grouped sections with Group and Row)

**Status:** ready-for-agent

- [ ] Both projects: `/settings/media`, `/settings/download-pipeline` and `/settings/general` show grouped rows per sub-section with counts drawn from developer-mode data; a sub-section with a warning shows it in the trailing detail
- [ ] Both projects: each of the six list pages renders its items as rows with tile, primary line and trailing detail; no card grid remains
- [ ] Both projects: the Add action is a trailing plus in the top bar on every list page and opens the existing create flow
- [ ] Both projects: switch rows toggle and the value persists after reload
- [ ] Phone project: back from a list page reads the settings section name; back from a section reads "More"
- [ ] The card-based list section and add placeholder components no longer exist
