# 13: Long-form settings as grouped control lists

Spec: `docs/admin-native-overhaul-spec.md` (stories 69, 72)

**What to build:** File Naming (import and naming), Auto Search, RSS Sync, Server and Authentication settings are laid out as grouped lists of controls with section footers carrying the explanatory text that today sits inline. Switches are trailing in their rows, selects and inputs are 44 px rows with the label leading, and save behaviour is unchanged. The Migrate from *arr flow remains reachable as a settings sub-screen under Media and renders inside `Screen` with its steps intact.

**Blocked by:** 11 (Settings index and list pages as grouped rows)

**Status:** ready-for-agent

- [ ] Both projects: each of the five pages renders grouped control rows with footers; no free-floating description paragraphs remain between controls
- [ ] Both projects: toggling a switch on Auto Search and saving persists after reload; changing a naming option updates the preview as today
- [ ] Both projects: Server and Authentication forms save with the same validation messages as before
- [ ] Both projects: Migrate from *arr opens from Media settings, shows its first step, and its back label is "Media"
- [ ] Phone project: every input on these pages is 16 px and at least 44 px tall
