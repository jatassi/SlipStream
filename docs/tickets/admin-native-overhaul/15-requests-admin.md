# 15: Requests admin: segmented queue, row actions, users and settings

Spec: `docs/admin-native-overhaul-spec.md` (stories 73–75)

**What to build:** The requests queue uses a segmented control across Pending, Approved, Downloading, Available and Denied, rendering each request as a row with poster, requester, media type and status pill. Approve and deny are trailing actions on the row; deny confirms in an action sheet on phones and a dialog on wide screens. Portal Users and Request Settings are grouped lists under Requests, and their forms (invite, user edit, request search) open through the sheet/dialog presenter. The portal itself under `/requests` is untouched.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter), 09 (Movie and series detail: hero, pill actions, grouped metadata, sheet presenter)

**Status:** ready-for-agent

- [ ] Both projects: the queue shows the five segments; choosing Pending lists developer-mode pending requests as rows with poster, requester, media type and status pill
- [ ] Both projects: approving a pending request moves it out of Pending and into Approved (or Downloading once auto search begins)
- [ ] Phone project: deny opens an action sheet; confirming moves the request to Denied; wide project: the same through a dialog
- [ ] Both projects: Users lists portal users as rows; invite and edit open through the presenter and a created user appears as a row
- [ ] Both projects: Request Settings renders as grouped controls and saves
- [ ] The portal under `/requests` renders exactly as before
