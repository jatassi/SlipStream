# Testing Checklist: Lint Fixes (setState-in-effect)

This checklist covers components modified during the lint fix session that converted `useEffect` state sync patterns to render-time state adjustment.

## Movies & Series

- [ ] **Add Movie** (`/movies/add`) - Search for a movie, select it, verify form populates with defaults
- [ ] **Add Movie via URL** (`/movies/add?tmdbId=xxx`) - Direct link auto-selects movie and shows configure step
- [ ] **Add Series** (`/series/add`) - Search for a series, select it, verify form populates with defaults
- [ ] **Add Series via URL** (`/series/add?tmdbId=xxx`) - Direct link auto-selects series and shows configure step
- [ ] **Poster images** - Images load correctly on movie/series grids and detail pages
- [ ] **Backdrop images** - Backdrop images load on movie/series detail pages

## Search

- [ ] **Header search bar** - Type to search, debounce works (500ms), navigates to results
- [ ] **Header search bar** - Press Enter to search immediately
- [ ] **Header search bar** - Clear button works
- [ ] **Search results page** - Expandable grids show "Show more" card when collapsed
- [ ] **Search results page** - Grids expand/collapse properly
- [ ] **Search results page** - Grid resets to collapsed when search query changes significantly
- [ ] **Manual search modal** - Open from movie/series detail page
- [ ] **Manual search modal** - Sort columns work (click to toggle direction)
- [ ] **Manual search modal** - Modal state resets when closed and reopened

## Settings

- [ ] **Quality Profiles** - Create new profile dialog opens with defaults
- [ ] **Quality Profiles** - Edit existing profile loads saved values
- [ ] **Quality Profiles** - Dialog resets properly when closed
- [ ] **Download Clients** - Add/edit dialog works
- [ ] **Download Clients** - Test connection button shows success/error toast
- [ ] **Indexers > Prowlarr** - Configure Prowlarr form loads saved values
- [ ] **Indexers > Prowlarr** - Form detects changes and enables save button
- [ ] **Media Management > File Naming** - Token builder dialog opens and closes properly
- [ ] **Media Management > File Naming** - Token builder resets when reopened
- [ ] **Media Management > File Naming** - Form saves correctly
- [ ] **Media Management > Auto Search** - Toggle switch works
- [ ] **Media Management > Auto Search** - Slider works
- [ ] **Media Management > Auto Search** - Form loads saved values on page load
- [ ] **Requests > Settings** - Form loads saved portal settings
- [ ] **Requests > Settings** - Form saves correctly

## Portal (if enabled)

- [ ] **Portal downloads** - Active downloads display correctly
- [ ] **Portal downloads** - Downloads maintain stable order (don't jump around)
- [ ] **Passkey management** - "Add Passkey" button shows registration form
- [ ] **Passkey management** - PIN input auto-submits when 4 digits entered
- [ ] **Passkey support check** - Page loads without hanging (sync check)

## Slots (if using multi-version)

- [ ] **Dry Run Modal** - Open assign slot dialog
- [ ] **Dry Run Modal** - Select a slot from dropdown
- [ ] **Dry Run Modal** - Assign works, dialog closes
- [ ] **Dry Run Modal** - Dialog resets selection when reopened
- [ ] **Migration Preview** - Same tests as Dry Run Modal

## Mobile / Responsive

- [ ] **Responsive layout** - Resize browser below 768px, layout adapts
- [ ] **Responsive layout** - Resize back above 768px, layout returns to desktop
