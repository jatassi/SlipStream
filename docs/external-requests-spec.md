# External Requests

## 1. Feature Overview

The external requests feature allows friends and family to request movies and TV shows not currently in the library. This is inspired by Overseerr but integrated directly into SlipStream.

### 1.1 Core Capabilities
- 1.1.1 User-friendly request interface for movies, series, seasons, and episodes
- 1.1.2 Request management with approval/denial workflow
- 1.1.3 Auto-approve system with quota controls
- 1.1.4 Multi-user support with magic link invitations
- 1.1.5 Notification support via existing SlipStream notification channels
- 1.1.6 Mobile-friendly responsive design
- 1.1.7 Real-time updates via WebSocket

### 1.2 Existing Foundations
- 1.2.1 Search feature and search results page
- 1.2.2 Notification system (Discord, Email, Slack, Telegram, Pushover, Webhook)
- 1.2.3 Quality profiles and root folders
- 1.2.4 Auto-search logic
- 1.2.5 Multi-version/slot system
- 1.2.6 WebSocket hub for real-time updates

## 2. Architecture

### 2.1 Deployment Model
- 2.1.1 Same binary as SlipStream with separate route groups
- 2.1.2 Portal accessible at `/requests`
- 2.1.3 Admin management integrated into main SlipStream UI
- 2.1.4 Shared database between portal and main application
- 2.1.5 Middleware-based permission enforcement

### 2.2 Security Model
- 2.2.1 Separate JWT audience/claims for portal users vs admin
- 2.2.2 Portal routes explicitly allowlisted (cannot accidentally expose admin endpoints)
- 2.2.3 Rate limiting on portal search endpoints
- 2.2.4 Login required for all portal access (no public/guest access)

### 2.3 UI Consistency
- 2.3.1 Portal uses identical styling and components as SlipStream
- 2.3.2 Search interface identical to main SlipStream search

## 3. User Management

### 3.1 User Roles
- 3.1.1 Two-tier system: Admin and User
- 3.1.2 Admin: Full SlipStream access plus request approval capabilities
- 3.1.3 User: Portal access only (search, request, view status)

### 3.2 Magic Link Signup
- 3.2.1 Admin generates invitation link for new users
- 3.2.2 One-click account creation via magic link
- 3.2.3 User sets password on first visit
- 3.2.4 Expired links persist - user can request resend via email without admin intervention
- 3.2.5 No limit on number of invitations admin can generate

### 3.3 User Profile
- 3.3.1 Display name optional at signup (defaults to email prefix before @)
- 3.3.2 Self-service profile editing: email, password, display name
- 3.3.3 No auto-disable of inactive accounts
- 3.3.4 Admin can manually enable/disable user accounts
- 3.3.5 Disabled users' pending requests remain in queue

### 3.4 Sessions
- 3.4.1 Long-lived session tokens (30 days)
- 3.4.2 Separate token handling from main SlipStream admin auth
- 3.4.3 Unauthenticated users redirected to login page
- 3.4.4 After admin login: redirect to originally requested URL
- 3.4.5 After portal user login: always redirect to main portal page (ignore original URL)

## 4. Request Workflow

### 4.1 Request Types
- 4.1.1 Movies: Request entire movie
- 4.1.2 Series: Request entire series with monitoring options
- 4.1.3 Seasons: Request specific season(s)
- 4.1.4 Episodes: Request individual episode(s)

### 4.2 Request Submission
- 4.2.1 Users can submit unlimited requests (quotas apply to auto-approve only)
- 4.2.2 Block submission if item already in library (show message)
- 4.2.3 Block submission if item already requested by another user (show link to existing request)
- 4.2.4 For series: User chooses monitoring options (same as current add series flow)
- 4.2.5 Quality determined by user's assigned quality profile (not user-selectable)
- 4.2.6 Root folder uses default (admin can override at approval)

### 4.3 Request Status Flow
- 4.3.1 **Pending**: Awaiting approval
- 4.3.2 **Approved**: Approved by admin or auto-approved
- 4.3.3 **Denied**: Rejected by admin (optional reason)
- 4.3.4 **Downloading**: Media is being downloaded
- 4.3.5 **Available**: Media is in library and ready to watch

### 4.4 Approval Actions
- 4.4.1 **Approve only**: Mark as approved, no immediate action
- 4.4.2 **Approve + auto-search**: Approve and immediately trigger search/download
- 4.4.3 **Approve + manual search**: Approve and flag for admin to manually search later
- 4.4.4 Batch operations: Admin can select multiple requests and approve/deny in bulk

### 4.5 Request Cancellation
- 4.5.1 Users can cancel their own pending requests (before approval)
- 4.5.2 Once approved, only admin can modify request status

### 4.6 Denial
- 4.6.1 Admin can deny requests
- 4.6.2 Denial reason is optional
- 4.6.3 If provided, reason is visible to user

### 4.7 Request History
- 4.7.1 Full history maintained (all statuses visible)
- 4.7.2 Fulfilled requests link to actual media item in library
- 4.7.3 Internal tracking of which user originally requested each item (not publicly displayed)

## 5. Auto-Approve System

### 5.1 Configuration
- 5.1.1 Control at quality profile level: New `allow_auto_approve` boolean field
- 5.1.2 Control at per-user level: User-specific auto-approve setting
- 5.1.3 **OR logic**: Auto-approve enabled if EITHER quality profile OR user setting allows it

### 5.2 Quota Integration
- 5.2.1 Quotas apply to auto-approved requests only
- 5.2.2 Users can submit unlimited requests regardless of quota
- 5.2.3 When quota exceeded: Request queued as pending (requires manual approval)

### 5.3 Search Behavior
- 5.3.1 Auto-approved requests use existing auto-search logic
- 5.3.2 No special handling for quality fallback (uses quality profile cutoff)

## 6. Quota System

### 6.1 Quota Structure
- 6.1.1 Three separate pools: Movies, Seasons, Episodes
- 6.1.2 Quota kicks in when ANY individual limit is reached
- 6.1.3 Example: 5 movies/week, 3 seasons/week, 10 episodes/week

### 6.2 Configuration
- 6.2.1 Global default quotas apply to all users
- 6.2.2 Per-user override available for specific users
- 6.2.3 Admin can set different limits for different users

### 6.3 Reset Schedule
- 6.3.1 Weekly reset (fixed schedule)
- 6.3.2 Reset occurs every Monday at midnight local server time

## 7. Quality & Storage

### 7.1 Quality Profiles
- 7.1.1 Each user is assigned a quality profile by admin
- 7.1.2 User's requests use their assigned profile
- 7.1.3 Users cannot select or change quality preference

### 7.2 Root Folders
- 7.2.1 Requests use a default root folder
- 7.2.2 Admin can override root folder at approval time

## 8. Multi-Version Mode

### 8.1 Availability Checks
- 8.1.1 Allow requests if user's quality profile targets a different slot than what exists
- 8.1.2 Block requests only if the target slot already has content
- 8.1.3 Search results show all existing slots with indicators (which quality tiers are available)

### 8.2 Request Fulfillment
- 8.2.1 Approved requests link to existing library item
- 8.2.2 Downloads go to a new slot on the existing item
- 8.2.3 No separate media item created for version requests

## 9. Notifications

### 9.1 User Notifications
- 9.1.1 Users configure their own notification channels
- 9.1.2 Same connectors available as main application (Discord, Email, Slack, Telegram, Pushover, Webhook)
- 9.1.3 Users notified only when media is **available** (not on approval/denial)

### 9.2 Admin Notifications
- 9.2.1 Admin can optionally receive notifications for new requests
- 9.2.2 Configurable via admin's notification settings

### 9.3 Watch Feature
- 9.3.1 Users can "watch" any request (including others' requests)
- 9.3.2 Watchers receive full notifications through their configured channels when request is fulfilled
- 9.3.3 Enables users to follow requests for items they also want

## 10. Portal UI

### 10.1 Search
- 10.1.1 Search interface identical to SlipStream main search
- 10.1.2 Results show availability indicators:
  - "Already in Library" badge (with slot info in multi-version mode)
  - "Already Requested" badge (with link to existing request)
  - Absence of badges indicates available to request
- 10.1.3 Global rate limiting on search queries to prevent abuse

### 10.2 Request List
- 10.2.1 Tab-based navigation by status (Pending, Approved, Downloading, Available, Denied)
- 10.2.2 Sorting options within each tab
- 10.2.3 All requests visible to all users (not just own requests)
- 10.2.4 Users can only edit/cancel their own requests

### 10.3 Real-Time Updates
- 10.3.1 WebSocket connection for live status updates
- 10.3.2 Request status changes reflected immediately without page refresh

### 10.4 Responsive Design
- 10.4.1 Mobile-friendly responsive layout
- 10.4.2 No PWA features (standard responsive web app)

## 11. Admin UI (in SlipStream)

### 11.1 Location
- 11.1.1 Separate admin area within SlipStream (not part of portal)
- 11.1.2 Accessible to admin users only

### 11.2 User Management
- 11.2.1 User list with status, quota usage, assigned quality profile
- 11.2.2 Enable/disable user accounts
- 11.2.3 Generate invitation links
- 11.2.4 Assign/change quality profile per user
- 11.2.5 Override quota limits per user
- 11.2.6 Configure per-user auto-approve setting

### 11.3 Request Management
- 11.3.1 Request queue with filtering and sorting
- 11.3.2 Batch approve/deny operations
- 11.3.3 Override root folder at approval
- 11.3.4 View request history and audit trail

### 11.4 Developer Mode
- 11.4.1 Admin can submit requests as if a regular user (for testing)
- 11.4.2 Only available when developer mode is enabled

## 12. Technical Requirements

### 12.1 Database Schema (New Tables)
- 12.1.1 `portal_users`: User accounts (id, email, password_hash, display_name, quality_profile_id, auto_approve, enabled, created_at)
- 12.1.2 `portal_invitations`: Magic link invitations (id, email, token, expires_at, used_at)
- 12.1.3 `requests`: Request records (id, user_id, media_type, tmdb_id/tvdb_id, status, created_at, approved_at, approved_by, denied_reason, media_id)
- 12.1.4 `request_watchers`: Watch relationships (request_id, user_id)
- 12.1.5 `user_quotas`: Per-user quota overrides (user_id, movies_limit, seasons_limit, episodes_limit)
- 12.1.6 `user_notifications`: User notification configurations (user_id, provider, settings, on_available)

### 12.2 Database Schema (Modifications)
- 12.2.1 `quality_profiles`: Add `allow_auto_approve` boolean field

### 12.3 API Routes
- 12.3.1 Portal: `/api/v1/requests/*` (search, submit, list, cancel, watch)
- 12.3.2 Portal Auth: `/api/v1/requests/auth/*` (login, signup, profile, notifications)
- 12.3.3 Admin: `/api/v1/admin/requests/*` (approve, deny, batch, users, invitations)

### 12.4 Global Settings
- 12.4.1 Default movie quota (per week)
- 12.4.2 Default season quota (per week)
- 12.4.3 Default episode quota (per week)
- 12.4.4 Default root folder for requests
- 12.4.5 Admin notification on new request (boolean)
- 12.4.6 Global search rate limit

### 12.5 External API
- 12.5.1 Not included in initial implementation
- 12.5.2 Can be added in future phase for Discord bots, etc.

## 13. Out of Scope (Initial Release)

- 13.1 Progressive Web App (PWA) features
- 13.2 External API for third-party integrations
- 13.3 Plex/Jellyfin authentication integration
- 13.4 Request comments/discussion threads
- 13.5 Request voting/priority system
