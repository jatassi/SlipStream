# External Requests Implementation Plan

## Overview

This plan implements the External Requests feature for SlipStream as specified in `/docs/external-requests-spec.md`. The implementation is organized into 7 phases with tasks ordered by dependency.

---

## Progress Tracker

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Database Foundation | ✅ Complete | All migrations and sqlc queries created |
| Phase 2: Authentication & User Management | ✅ Complete | Auth service, users service, invitations service, middleware, handlers |
| Phase 3: Core Request Services | ✅ Complete | Requests service, library checker, quota service, watchers service |
| Phase 4: Auto-Approve & Integration | ✅ Complete | Auto-approve service, request searcher, status tracker, notifications, WebSocket events, quota reset scheduler |
| Phase 5: API Layer | ✅ Complete | All handlers created, routes registered, rate limiting added |
| Phase 6: Frontend - Portal | ✅ Complete | Portal routes, API client, hooks, auth store, layout, all pages created |
| Phase 7: Frontend - Admin | ✅ Complete | Admin API client, hooks, user/invitation management, request queue, settings pages |
| Phase 8: Polish & Finalization | ✅ Complete | Quality profile UI updated, CLAUDE.md/AGENTS.md documented. Tasks 8.2-8.5 are manual testing. |

---

## Phase 1: Database Foundation ✅

**Goal:** Create all required database tables and schema modifications.

### Task 1.1: Create Portal Users Migration
**Requirements:** 3.1.1, 3.1.2, 3.1.3, 3.3.1, 3.3.3, 3.3.4, 3.4.1, 7.1.1, 12.1.1

Create migration `027_portal_users.sql`:
- `portal_users` table with columns:
  - `id` INTEGER PRIMARY KEY AUTOINCREMENT
  - `email` TEXT UNIQUE NOT NULL
  - `password_hash` TEXT NOT NULL
  - `display_name` TEXT
  - `quality_profile_id` INTEGER REFERENCES quality_profiles(id)
  - `auto_approve` INTEGER DEFAULT 0
  - `enabled` INTEGER DEFAULT 1
  - `created_at`, `updated_at` DATETIME

### Task 1.2: Create Portal Invitations Migration
**Requirements:** 3.2.1, 3.2.2, 3.2.3, 3.2.4, 3.2.5, 12.1.2

Create migration `028_portal_invitations.sql`:
- `portal_invitations` table:
  - `id` INTEGER PRIMARY KEY AUTOINCREMENT
  - `email` TEXT NOT NULL
  - `token` TEXT UNIQUE NOT NULL
  - `expires_at` DATETIME NOT NULL
  - `used_at` DATETIME (NULL until used)
  - `created_at` DATETIME

### Task 1.3: Create Requests Migration
**Requirements:** 4.1.1-4.1.4, 4.3.1-4.3.5, 4.6.2, 4.7.1, 4.7.2, 4.7.3, 12.1.3

Create migration `029_requests.sql`:
- `requests` table:
  - `id` INTEGER PRIMARY KEY AUTOINCREMENT
  - `user_id` INTEGER NOT NULL REFERENCES portal_users(id)
  - `media_type` TEXT NOT NULL (movie/series/season/episode)
  - `tmdb_id` INTEGER
  - `tvdb_id` INTEGER
  - `title` TEXT NOT NULL
  - `year` INTEGER
  - `season_number` INTEGER (for season/episode requests)
  - `episode_number` INTEGER (for episode requests)
  - `status` TEXT NOT NULL DEFAULT 'pending'
  - `monitor_future` INTEGER DEFAULT 0 (for series)
  - `denied_reason` TEXT
  - `approved_at` DATETIME
  - `approved_by` INTEGER REFERENCES portal_users(id)
  - `media_id` INTEGER (links to movie/series/episode after fulfillment)
  - `target_slot_id` INTEGER REFERENCES version_slots(id)
  - `created_at`, `updated_at` DATETIME
- Indexes on `user_id`, `status`, `tmdb_id`, `tvdb_id`

### Task 1.4: Create Request Watchers Migration
**Requirements:** 9.3.1, 9.3.2, 9.3.3, 12.1.4

Create migration `030_request_watchers.sql`:
- `request_watchers` table:
  - `request_id` INTEGER REFERENCES requests(id) ON DELETE CASCADE
  - `user_id` INTEGER REFERENCES portal_users(id) ON DELETE CASCADE
  - PRIMARY KEY (request_id, user_id)
  - `created_at` DATETIME

### Task 1.5: Create User Quotas Migration
**Requirements:** 6.1.1, 6.1.2, 6.2.1, 6.2.2, 6.2.3, 12.1.5

Create migration `031_user_quotas.sql`:
- `user_quotas` table:
  - `user_id` INTEGER PRIMARY KEY REFERENCES portal_users(id) ON DELETE CASCADE
  - `movies_limit` INTEGER
  - `seasons_limit` INTEGER
  - `episodes_limit` INTEGER
  - `movies_used` INTEGER DEFAULT 0
  - `seasons_used` INTEGER DEFAULT 0
  - `episodes_used` INTEGER DEFAULT 0
  - `period_start` DATETIME
  - `updated_at` DATETIME

### Task 1.6: Create User Notifications Migration
**Requirements:** 9.1.1, 9.1.2, 9.1.3, 12.1.6

Create migration `032_user_notifications.sql`:
- `user_notifications` table:
  - `id` INTEGER PRIMARY KEY AUTOINCREMENT
  - `user_id` INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE
  - `type` TEXT NOT NULL (discord/email/telegram/etc.)
  - `name` TEXT NOT NULL
  - `settings` TEXT NOT NULL (JSON)
  - `on_available` INTEGER DEFAULT 1
  - `enabled` INTEGER DEFAULT 1
  - `created_at`, `updated_at` DATETIME

### Task 1.7: Modify Quality Profiles Migration
**Requirements:** 5.1.1, 12.2.1

Create migration `033_quality_profile_auto_approve.sql`:
- Add `allow_auto_approve` INTEGER DEFAULT 0 to `quality_profiles`

### Task 1.8: Create Global Request Settings Migration
**Requirements:** 12.4.1-12.4.6

Create migration `034_request_settings.sql`:
- Insert default settings into `settings` table:
  - `requests_default_movie_quota`
  - `requests_default_season_quota`
  - `requests_default_episode_quota`
  - `requests_default_root_folder_id`
  - `requests_admin_notify_new`
  - `requests_search_rate_limit`

### Task 1.9: Generate sqlc Queries
**Requirements:** All 12.1.x

Create query files:
- `internal/database/queries/portal_users.sql`
- `internal/database/queries/portal_invitations.sql`
- `internal/database/queries/requests.sql`
- `internal/database/queries/request_watchers.sql`
- `internal/database/queries/user_quotas.sql`
- `internal/database/queries/user_notifications.sql`

Run `sqlc generate` after creating all queries.

---

## Phase 2: Authentication & User Management ✅

**Goal:** Implement multi-user authentication system for portal users.

### Task 2.1: Create Portal Auth Service
**Requirements:** 2.2.1, 3.4.1, 3.4.2

Create `internal/portal/auth/service.go`:
- Extended JWT claims with `UserID`, `Email`, `Role`, `Audience`
- `GeneratePortalToken(user)` - 30-day expiration, portal audience
- `ValidatePortalToken(token)` - Validates portal-specific tokens
- `HashPassword(password)` / `ValidatePassword(hash, password)`
- Separate from main SlipStream auth (different audience claim)

### Task 2.2: Create Portal Users Service
**Requirements:** 3.1.1-3.1.3, 3.3.1-3.3.5, 7.1.1, 7.1.2

Create `internal/portal/users/service.go`:
- `Create(email, password, displayName)` - Creates new user
- `Get(id)` / `GetByEmail(email)` - Retrieve user
- `List()` - List all users (admin)
- `Update(id, input)` - Update profile (email, password, displayName)
- `SetEnabled(id, enabled)` - Enable/disable user
- `SetQualityProfile(id, profileID)` - Assign quality profile
- `SetAutoApprove(id, enabled)` - Per-user auto-approve setting
- `Delete(id)` - Delete user (preserve requests per 3.3.5)

### Task 2.3: Create Invitations Service
**Requirements:** 3.2.1-3.2.5

Create `internal/portal/invitations/service.go`:
- `Create(email)` - Generate invitation with secure token
- `Get(token)` - Retrieve invitation by token
- `Validate(token)` - Check if token valid and unused
- `MarkUsed(token)` - Mark invitation as used
- `ResendLink(email)` - Generate new token for existing invitation
- `ListPending()` - List unused invitations (admin)
- `Delete(id)` - Delete invitation
- Token generation using `crypto/rand`
- Default expiration: 7 days (configurable)

### Task 2.4: Create Portal Auth Middleware
**Requirements:** 2.2.1, 2.2.2, 2.2.4

Create `internal/portal/middleware/auth.go`:
- `PortalAuthMiddleware` - Validates portal JWT tokens
- Extracts user claims into Echo context
- Returns 401 for missing/invalid tokens
- Sets `c.Set("portalUser", claims)` for handlers

### Task 2.5: Create Portal Auth Handlers
**Requirements:** 3.2.2, 3.2.3, 3.2.4, 3.3.2, 12.3.2

Create `internal/portal/auth/handlers.go`:
- `POST /api/v1/requests/auth/login` - Email/password login
- `POST /api/v1/requests/auth/signup` - Complete signup via invitation token
- `POST /api/v1/requests/auth/resend` - Resend invitation link
- `GET /api/v1/requests/auth/profile` - Get current user profile
- `PUT /api/v1/requests/auth/profile` - Update profile
- `POST /api/v1/requests/auth/logout` - Logout (client-side token removal)

---

## Phase 3: Core Request Services ✅

**Goal:** Implement request management business logic.

### Task 3.1: Create Requests Service
**Requirements:** 4.1.1-4.1.4, 4.2.1-4.2.6, 4.3.1-4.3.5, 4.5.1, 4.5.2, 4.6.1-4.6.3, 4.7.1-4.7.3

Create `internal/portal/requests/service.go`:
- `Create(userID, input)` - Submit new request
  - Check library existence (4.2.2)
  - Check duplicate requests (4.2.3)
  - Set monitoring options for series (4.2.4)
- `Get(id)` / `GetByTmdbID(tmdbID)` / `GetByTvdbID(tvdbID)`
- `List(filters)` - List with status/type filtering
- `ListByUser(userID)` - User's requests
- `Cancel(id, userID)` - Cancel pending request (4.5.1)
- `Approve(id, approverID, action)` - Approve with action type (4.4.1-4.4.3)
- `Deny(id, approverID, reason)` - Deny with optional reason
- `UpdateStatus(id, status)` - Update request status
- `LinkMedia(id, mediaID)` - Link to fulfilled media (4.7.2)
- `BatchApprove(ids, approverID, action)` / `BatchDeny(ids, approverID, reason)` (4.4.4)

### Task 3.2: Create Library Check Service
**Requirements:** 4.2.2, 4.2.3, 8.1.1, 8.1.2, 8.1.3

Create `internal/portal/requests/library_check.go`:
- `CheckMovieAvailability(tmdbID, userQualityProfileID)` - Returns availability info
- `CheckSeriesAvailability(tvdbID, ...)` - Series availability
- `CheckSeasonAvailability(tvdbID, seasonNum, ...)` - Season availability
- `CheckEpisodeAvailability(tvdbID, seasonNum, episodeNum, ...)` - Episode availability
- Returns: `{inLibrary: bool, existingSlots: []SlotInfo, canRequest: bool, existingRequestID: *int64}`
- Multi-version aware: checks target slot vs existing slots (8.1.1, 8.1.2)

### Task 3.3: Create Quota Service
**Requirements:** 5.2.1, 5.2.2, 5.2.3, 6.1.1-6.1.3, 6.2.1-6.2.3, 6.3.1, 6.3.2

Create `internal/portal/quota/service.go`:
- `GetUserQuota(userID)` - Get current quota status
- `CheckQuota(userID, mediaType)` - Check if user can auto-approve
- `ConsumeQuota(userID, mediaType)` - Decrement available quota
- `ResetQuotas()` - Reset all user quotas (called by scheduler)
- `GetGlobalDefaults()` - Get default quota limits
- `SetUserOverride(userID, limits)` - Set per-user limits
- Weekly reset logic: Monday midnight local time (6.3.2)
- Separate pools for movies, seasons, episodes (6.1.1)

### Task 3.4: Create Request Watchers Service
**Requirements:** 9.3.1, 9.3.2, 9.3.3

Create `internal/portal/requests/watchers.go`:
- `Watch(requestID, userID)` - Add watcher to request
- `Unwatch(requestID, userID)` - Remove watcher
- `GetWatchers(requestID)` - List all watchers
- `IsWatching(requestID, userID)` - Check if user is watching
- `GetWatchedRequests(userID)` - Get requests user is watching

---

## Phase 4: Auto-Approve & Integration

**Goal:** Implement auto-approve logic and integrate with existing services.

### Task 4.1: Create Auto-Approve Service
**Requirements:** 5.1.1, 5.1.2, 5.1.3, 5.2.1, 5.2.2, 5.2.3, 5.3.1, 5.3.2

Create `internal/portal/autoapprove/service.go`:
- `ShouldAutoApprove(userID, qualityProfileID)` - OR logic (5.1.3)
  - Check user's `auto_approve` flag
  - Check quality profile's `allow_auto_approve` flag
- `ProcessAutoApprove(request)` - Full auto-approve flow:
  1. Check if auto-approve enabled
  2. Check quota availability
  3. If quota OK: approve + trigger autosearch
  4. If quota exceeded: leave as pending (5.2.3)
- Integration point: Called after request creation

### Task 4.2: Integrate with AutoSearch Service
**Requirements:** 4.4.2, 5.3.1, 5.3.2, 8.2.1, 8.2.2

Modify `internal/autosearch/service.go`:
- Add `SearchForRequest(requestID)` method
- Uses existing `SearchMovie` / `SearchEpisode` / `SearchSeason` / `SearchSeries`
- On success: Update request status to 'downloading'
- On completion: Update request status to 'available', link media ID
- Multi-version: Uses request's `target_slot_id` (8.2.1, 8.2.2)

### Task 4.3: Create Request Status Tracker
**Requirements:** 4.3.1-4.3.5, 4.7.2

Create `internal/portal/requests/status_tracker.go`:
- Subscribe to import completion events
- Update request status when media becomes available
- Link request to media ID on fulfillment
- Handles both single items and season packs

### Task 4.4: Integrate with Notification Service
**Requirements:** 9.1.1-9.1.3, 9.2.1, 9.2.2, 9.3.2

Modify notification system:
- Add new event types: `EventRequestAvailable`
- Create `internal/portal/notifications/service.go`:
  - `NotifyRequestAvailable(request)` - Notify requester + watchers
  - `NotifyAdminNewRequest(request)` - Notify admin if enabled
- Uses user's configured notification channels
- Watchers receive notifications through their own channels (9.3.2)

### Task 4.5: Add WebSocket Events
**Requirements:** 10.3.1, 10.3.2

Modify `internal/websocket/hub.go`:
- Add event types:
  - `request:created` - New request submitted
  - `request:updated` - Request status changed
  - `request:deleted` - Request cancelled/removed
- Broadcast on all request state changes

### Task 4.6: Create Quota Reset Scheduler Task
**Requirements:** 6.3.1, 6.3.2

Create `internal/scheduler/tasks/quotareset.go`:
- Scheduled task for Monday midnight local time
- Calls `quotaService.ResetQuotas()`
- Register in scheduler during server startup

---

## Phase 5: API Layer

**Goal:** Implement all HTTP API endpoints.

### Task 5.1: Create Portal Request Handlers
**Requirements:** 12.3.1

Create `internal/portal/requests/handlers.go`:
- `GET /api/v1/requests` - List all requests (with filters)
- `POST /api/v1/requests` - Submit new request
- `GET /api/v1/requests/:id` - Get request details
- `DELETE /api/v1/requests/:id` - Cancel request (owner only, pending only)
- `POST /api/v1/requests/:id/watch` - Watch request
- `DELETE /api/v1/requests/:id/watch` - Unwatch request

### Task 5.2: Create Portal Search Handlers
**Requirements:** 10.1.1, 10.1.2, 10.1.3

Create `internal/portal/search/handlers.go`:
- `GET /api/v1/requests/search` - Search with availability enrichment
- Wraps existing metadata search
- Enriches results with:
  - `inLibrary` flag + slot info (10.1.2)
  - `existingRequest` info (10.1.2)
- Apply global rate limiting (10.1.3)

### Task 5.3: Create Portal User Notification Handlers
**Requirements:** 9.1.1, 9.1.2

Create `internal/portal/notifications/handlers.go`:
- `GET /api/v1/requests/auth/notifications` - List user's notifications
- `POST /api/v1/requests/auth/notifications` - Add notification channel
- `PUT /api/v1/requests/auth/notifications/:id` - Update channel
- `DELETE /api/v1/requests/auth/notifications/:id` - Remove channel
- `POST /api/v1/requests/auth/notifications/:id/test` - Test channel
- `GET /api/v1/requests/auth/notifications/schema` - Get available providers

### Task 5.4: Create Admin User Handlers
**Requirements:** 11.2.1-11.2.6, 12.3.3

Create `internal/portal/admin/users_handlers.go`:
- `GET /api/v1/admin/requests/users` - List all users with quota status
- `GET /api/v1/admin/requests/users/:id` - Get user details
- `PUT /api/v1/admin/requests/users/:id` - Update user settings
- `POST /api/v1/admin/requests/users/:id/enable` - Enable user
- `POST /api/v1/admin/requests/users/:id/disable` - Disable user
- `DELETE /api/v1/admin/requests/users/:id` - Delete user

### Task 5.5: Create Admin Invitation Handlers
**Requirements:** 11.2.3

Create `internal/portal/admin/invitations_handlers.go`:
- `GET /api/v1/admin/requests/invitations` - List invitations
- `POST /api/v1/admin/requests/invitations` - Create invitation
- `DELETE /api/v1/admin/requests/invitations/:id` - Delete invitation

### Task 5.6: Create Admin Request Handlers
**Requirements:** 11.3.1-11.3.4, 12.3.3

Create `internal/portal/admin/requests_handlers.go`:
- `GET /api/v1/admin/requests` - List requests with admin filtering
- `POST /api/v1/admin/requests/:id/approve` - Approve with action
- `POST /api/v1/admin/requests/:id/deny` - Deny with reason
- `POST /api/v1/admin/requests/batch/approve` - Batch approve
- `POST /api/v1/admin/requests/batch/deny` - Batch deny

### Task 5.7: Create Admin Settings Handlers
**Requirements:** 12.4.1-12.4.6

Create `internal/portal/admin/settings_handlers.go`:
- `GET /api/v1/admin/requests/settings` - Get global settings
- `PUT /api/v1/admin/requests/settings` - Update global settings
- Settings: default quotas, root folder, admin notifications, rate limit

### Task 5.8: Create Search Rate Limiter
**Requirements:** 2.2.3, 10.1.3

Create `internal/portal/ratelimit/search_limiter.go`:
- Global rate limiter for portal search requests
- Configurable limit (requests per minute)
- Uses existing `ratelimit.Limiter` pattern from indexer
- Middleware integration for portal search endpoints

### Task 5.9: Register All Routes
**Requirements:** 2.1.2, 2.2.2, 12.3.1-12.3.3

Modify `internal/api/server.go`:
- Create `/api/v1/requests` group with portal auth middleware
- Create `/api/v1/admin/requests` group with admin auth check
- Register all handlers from Tasks 5.1-5.7
- Apply rate limiting middleware to search endpoints

---

## Phase 6: Frontend - Portal

**Goal:** Implement portal user interface.

### Task 6.1: Create Portal Route Structure
**Requirements:** 2.1.2, 2.3.1, 2.3.2

Create frontend route structure:
```
/web/src/routes/requests/
  ├── index.tsx          (request list)
  ├── search.tsx         (search + request)
  ├── $id.tsx            (request detail)
  └── profile.tsx        (user profile + notifications)
/web/src/routes/requests/auth/
  ├── login.tsx
  └── signup.tsx
```

Update `router.tsx` with new routes.

### Task 6.2: Create Portal API Client
**Requirements:** All portal endpoints

Create `/web/src/api/portal/`:
- `auth.ts` - Login, signup, profile, logout
- `requests.ts` - Request CRUD, watch/unwatch
- `search.ts` - Search with availability
- `notifications.ts` - User notification management

### Task 6.3: Create Portal Hooks
**Requirements:** All portal functionality

Create `/web/src/hooks/portal/`:
- `usePortalAuth.ts` - Auth state, login/logout mutations
- `useRequests.ts` - Request queries and mutations
- `usePortalSearch.ts` - Search with availability enrichment
- `useUserNotifications.ts` - User notification management
- `usePortalUser.ts` - Current user profile

### Task 6.4: Create Portal Auth Store
**Requirements:** 3.4.1, 3.4.2, 3.4.3, 3.4.4, 3.4.5

Create `/web/src/stores/portalAuth.ts`:
- `token` - JWT token storage
- `user` - Current user info
- `isAuthenticated` - Auth status
- `redirectUrl` - Original URL user tried to access (admin only)
- `login(email, password)` - Authenticate
- `logout()` - Clear auth state
- `updateProfile(data)` - Update user info
- `setRedirectUrl(url)` - Store URL for post-login redirect (admin only)
- `getPostLoginRedirect()` - Returns redirect URL for admin, main portal for portal users
- Persist to localStorage

### Task 6.5: Create Portal Layout
**Requirements:** 2.3.1, 10.4.1, 10.4.2

Create `/web/src/components/portal/`:
- `PortalLayout.tsx` - Main layout wrapper
- `PortalHeader.tsx` - Header with user menu
- `PortalSidebar.tsx` - Navigation (if needed)
- Use same styling/components as main SlipStream (2.3.1)
- Responsive design (10.4.1)

### Task 6.6: Create Portal Auth Pages
**Requirements:** 3.2.2, 3.2.3, 3.3.1, 3.4.3, 3.4.4, 3.4.5

Create auth pages:
- `login.tsx` - Email/password login form
  - On successful admin login: redirect to stored URL or default admin page
  - On successful portal user login: always redirect to main portal page (`/requests`)
- `signup.tsx` - Signup via invitation token
  - Set password
  - Optional display name (defaults to email prefix)
  - After signup: redirect to main portal page

Create auth guard component:
- `PortalAuthGuard.tsx` - Wraps protected portal routes
  - If unauthenticated: redirect to login, store original URL (for admin detection)
  - If authenticated: render children

### Task 6.7: Create Portal Search Page
**Requirements:** 10.1.1, 10.1.2, 10.1.3

Create `/web/src/routes/requests/search.tsx`:
- Search interface identical to main SlipStream (10.1.1)
- Display availability badges:
  - "Already in Library" with slot info
  - "Already Requested" with link
- Request button for available items
- Request dialog with monitoring options (for series)

### Task 6.8: Create Request List Page
**Requirements:** 10.2.1-10.2.4, 10.3.1, 10.3.2

Create `/web/src/routes/requests/index.tsx`:
- Tab navigation: Pending, Approved, Downloading, Available, Denied (10.2.1)
- Sorting options within tabs (10.2.2)
- All users' requests visible (10.2.3)
- Edit/cancel only own requests (10.2.4)
- WebSocket subscription for real-time updates (10.3.1, 10.3.2)

### Task 6.9: Create Request Detail Page
**Requirements:** 4.7.1, 9.3.1

Create `/web/src/routes/requests/$id.tsx`:
- Full request details
- Status history
- Watch/unwatch button
- Cancel button (if pending and owner)
- Link to media detail (if fulfilled)

### Task 6.10: Create User Profile Page
**Requirements:** 3.3.2, 9.1.1, 9.1.2

Create `/web/src/routes/requests/profile.tsx`:
- Profile editing: email, password, display name
- Notification channels management
- Same notification provider UI as main app

### Task 6.11: Add Portal WebSocket Handler
**Requirements:** 10.3.1, 10.3.2

Modify `/web/src/stores/websocket.ts`:
- Add handlers for `request:created`, `request:updated`, `request:deleted`
- Invalidate request queries on updates
- Update request list in real-time

---

## Phase 7: Frontend - Admin

**Goal:** Implement admin management interface in SlipStream.

### Task 7.1: Create Admin API Client
**Requirements:** All admin endpoints

Create `/web/src/api/admin/`:
- `users.ts` - User management
- `invitations.ts` - Invitation management
- `requests.ts` - Request approval/denial
- `settings.ts` - Global request settings

### Task 7.2: Create Admin Hooks
**Requirements:** All admin functionality

Create `/web/src/hooks/admin/`:
- `useAdminUsers.ts` - User list, enable/disable, delete
- `useAdminInvitations.ts` - Invitation CRUD
- `useAdminRequests.ts` - Request approval/denial, batch operations
- `useRequestSettings.ts` - Global settings management

### Task 7.3: Create User Management Page
**Requirements:** 11.1.1, 11.1.2, 11.2.1-11.2.6

Create `/web/src/routes/settings/requests/users.tsx`:
- User list with columns: name, email, status, quota usage, quality profile
- Enable/disable toggle
- Edit user dialog: quality profile, auto-approve, quota override
- Generate invitation button
- Delete user button

### Task 7.4: Create Invitation Management UI
**Requirements:** 11.2.3

Add to user management page:
- "Invite User" button
- Invitation list with status (pending/used/expired)
- Copy link button
- Delete invitation button

### Task 7.5: Create Request Management Page
**Requirements:** 11.3.1-11.3.4

Create `/web/src/routes/settings/requests/index.tsx`:
- Request queue with filtering/sorting (11.3.1)
- Checkbox selection for batch operations
- Approve button with action dropdown (approve only, auto-search, manual search)
- Deny button with optional reason dialog
- Batch approve/deny buttons (11.3.2)
- Root folder override at approval (11.3.3)

### Task 7.6: Create Request Settings Page
**Requirements:** 12.4.1-12.4.6

Create `/web/src/routes/settings/requests/settings.tsx`:
- Default quotas: movies, seasons, episodes per week
- Default root folder selection
- Admin notification toggle for new requests
- Global search rate limit setting

### Task 7.7: Add Settings Navigation
**Requirements:** 11.1.1

Modify `/web/src/routes/settings/index.tsx`:
- Add "External Requests" section with links to:
  - Users & Invitations
  - Request Queue
  - Request Settings

### Task 7.8: Developer Mode Admin Requests
**Requirements:** 11.4.1, 11.4.2

Modify request management:
- Add "Submit Test Request" button (visible in dev mode only)
- Uses existing `useDeveloperMode()` hook
- Quota exempt for test requests

---

## Phase 8: Polish & Finalization

**Goal:** Final integration, testing, and documentation.

### Task 8.1: Quality Profile UI Update
**Requirements:** 5.1.1

Modify quality profile edit dialog:
- Add "Allow Auto-Approve" toggle
- Display in profile list

### Task 8.2: Multi-Version Mode Integration Testing
**Requirements:** 8.1.1-8.1.3, 8.2.1-8.2.3

Test scenarios:
- Request when different slot exists
- Request fulfillment to correct slot
- Slot info display in search results

### Task 8.3: Notification Testing
**Requirements:** 9.1.1-9.1.3, 9.2.1, 9.2.2, 9.3.2

Test scenarios:
- User notifications on request available
- Admin notifications on new request
- Watcher notifications

### Task 8.4: Quota System Testing
**Requirements:** 6.1.1-6.3.2

Test scenarios:
- Auto-approve within quota
- Auto-approve exceeds quota → pending
- Weekly reset on Monday midnight
- Per-user override

### Task 8.5: WebSocket Real-Time Testing
**Requirements:** 10.3.1, 10.3.2

Test scenarios:
- Request status updates reflect immediately
- Multiple browser sessions sync correctly

### Task 8.6: Update CLAUDE.md/AGENTS.md
Document new patterns and endpoints for future development.

---

## Requirement Coverage Audit

| Spec Section | Requirements | Covered In Tasks |
|--------------|--------------|------------------|
| 1.1 | 1.1.1-1.1.7 | Throughout all phases |
| 1.2 | 1.2.1-1.2.6 | Uses existing services |
| 2.1 | 2.1.1-2.1.5 | 5.9, Architecture |
| 2.2 | 2.2.1-2.2.4 | 2.1, 2.4, 5.8 |
| 2.3 | 2.3.1-2.3.2 | 6.1, 6.5 |
| 3.1 | 3.1.1-3.1.3 | 1.1, 2.2 |
| 3.2 | 3.2.1-3.2.5 | 1.2, 2.3, 2.5, 6.6 |
| 3.3 | 3.3.1-3.3.5 | 1.1, 2.2, 6.6, 6.10 |
| 3.4 | 3.4.1-3.4.5 | 2.1, 6.4, 6.6 |
| 4.1 | 4.1.1-4.1.4 | 1.3, 3.1 |
| 4.2 | 4.2.1-4.2.6 | 3.1, 3.2 |
| 4.3 | 4.3.1-4.3.5 | 1.3, 3.1, 4.3 |
| 4.4 | 4.4.1-4.4.4 | 3.1, 5.6, 7.5 |
| 4.5 | 4.5.1-4.5.2 | 3.1, 5.1 |
| 4.6 | 4.6.1-4.6.3 | 1.3, 3.1, 5.6 |
| 4.7 | 4.7.1-4.7.3 | 1.3, 3.1, 4.3, 6.9 |
| 5.1 | 5.1.1-5.1.3 | 1.7, 4.1 |
| 5.2 | 5.2.1-5.2.3 | 4.1 |
| 5.3 | 5.3.1-5.3.2 | 4.1, 4.2 |
| 6.1 | 6.1.1-6.1.3 | 1.5, 3.3 |
| 6.2 | 6.2.1-6.2.3 | 1.5, 3.3, 7.3 |
| 6.3 | 6.3.1-6.3.2 | 3.3, 4.6 |
| 7.1 | 7.1.1-7.1.3 | 1.1, 2.2 |
| 7.2 | 7.2.1-7.2.2 | 3.1, 5.6 |
| 8.1 | 8.1.1-8.1.3 | 3.2, 8.2 |
| 8.2 | 8.2.1-8.2.3 | 4.2, 8.2 |
| 9.1 | 9.1.1-9.1.3 | 1.6, 4.4, 5.3, 6.10 |
| 9.2 | 9.2.1-9.2.2 | 4.4, 5.7 |
| 9.3 | 9.3.1-9.3.3 | 1.4, 3.4, 4.4, 6.9 |
| 10.1 | 10.1.1-10.1.3 | 5.2, 5.8, 6.7 |
| 10.2 | 10.2.1-10.2.4 | 6.8 |
| 10.3 | 10.3.1-10.3.2 | 4.5, 6.11, 8.5 |
| 10.4 | 10.4.1-10.4.2 | 6.5 |
| 11.1 | 11.1.1-11.1.2 | 7.7 |
| 11.2 | 11.2.1-11.2.6 | 5.4, 5.5, 7.3, 7.4 |
| 11.3 | 11.3.1-11.3.4 | 5.6, 7.5 |
| 11.4 | 11.4.1-11.4.2 | 7.8 |
| 12.1 | 12.1.1-12.1.6 | 1.1-1.6, 1.9 |
| 12.2 | 12.2.1 | 1.7, 8.1 |
| 12.3 | 12.3.1-12.3.3 | 5.1-5.6, 5.9 |
| 12.4 | 12.4.1-12.4.6 | 1.8, 5.7, 7.6 |
| 12.5 | 12.5.1-12.5.2 | N/A (out of scope) |
| 13 | 13.1-13.5 | N/A (out of scope) |

**All requirements from sections 1-12 are covered.** Section 13 items are explicitly out of scope.
