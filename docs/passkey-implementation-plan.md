# Passkey Authentication Implementation Plan

## Overview

Add WebAuthn/Passkey support to SlipStream's authentication system for both admin and portal users. Passkeys will be the primary authentication method with PIN remaining as a required fallback.

## Implementation Notes (Deviations from Original Plan)

During implementation, the following changes were made:

1. **Auth Package Consolidation**: The original plan called for `internal/portal/passkey/` as a separate package. Instead, we:
   - Discovered `internal/auth/` was dead legacy code (never imported, no database table)
   - Deleted the dead `internal/auth/` package
   - Moved `internal/portal/auth/` → `internal/auth/` for cleaner structure
   - Added passkey code directly to `internal/auth/` (`passkey.go`, `passkey_handlers.go`)

2. **WebAuthn Library API**: The go-webauthn library's `FinishRegistration` and `FinishDiscoverableLogin` methods expect `*http.Request` rather than parsed credential structs. The handlers pass the raw HTTP request to the service.

3. **Query Parameters for Finish Endpoints**: The finish endpoints use query parameters (`challengeId`, `name`) instead of JSON body fields because the request body contains the raw WebAuthn credential response that the library parses directly.

## Current State

### Admin Authentication
- Single admin user stored in `portal_users` with `is_admin = 1`
- Username always "Administrator"
- 4-digit PIN stored as bcrypt hash
- JWT tokens with `aud: "admin"`, 24-hour expiry
- Setup via localhost-only endpoint

### Portal User Authentication
- Users stored in `portal_users` with `is_admin = 0`
- Invitation-based registration (admin creates invite → user signs up)
- Password stored as bcrypt hash
- JWT tokens with `aud: "portal"`, 30-day expiry

### Shared Infrastructure
- JWT secret stored in `settings` table (`portal_jwt_secret`)
- Common auth middleware validates tokens by audience
- Both user types share `portal_users` table

---

## Design Decisions

### 1. PIN is Mandatory, Passkey is Additive
- **PIN/password is always required** - set during initial setup (admin) or signup (portal users)
- **Passkey registration requires PIN verification first** - proves identity before adding new auth method
- Login UI shows passkey option first (if user has passkeys registered)
- PIN/password always available as "Use PIN instead"
- User can have multiple passkeys registered
- Passkeys can always be deleted (PIN remains as guaranteed fallback)

### 2. Single Credentials Table
- One `passkey_credentials` table for both admin and portal users
- `user_id` references `portal_users.id`
- No separate `user_type` column needed (user type determined by `portal_users.is_admin`)

### 3. Challenge Storage
- Challenges stored in-memory with TTL (not database)
- Use Go's `sync.Map` with expiration cleanup
- Challenges expire after 5 minutes
- Prevents replay attacks via sign count tracking

### 4. Library Choice
- Backend: `github.com/go-webauthn/webauthn` (standard, well-maintained)
- Frontend: `@simplewebauthn/browser` (simplifies credential API)

---

## Implementation Phases

## Phase 0: Setup & Cleanup

### 0.1 Remove Dead Code

Remove the `UpdatePortalUserAdmin` query which is never called and poses a security risk if accidentally exposed.

**File:** `internal/database/queries/portal_users.sql`

Delete this query:
```sql
-- name: UpdatePortalUserAdmin :one
UPDATE portal_users SET
    is_admin = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;
```

Then regenerate sqlc:
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

### 0.2 Add Backend Dependencies

```bash
go get github.com/go-webauthn/webauthn
```

### 0.3 Add Frontend Dependencies

```bash
cd web && bun add @simplewebauthn/browser
```

---

## Phase 1: Database & Backend Foundation

### 1.1 Database Migration

**File:** `internal/database/migrations/041_passkey_credentials.sql`

```sql
-- +goose Up
CREATE TABLE passkey_credentials (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    credential_id BLOB NOT NULL UNIQUE,
    public_key BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    transport TEXT,  -- JSON array of transports: ["internal", "usb", "ble", "nfc"]
    flags_user_present BOOLEAN NOT NULL DEFAULT FALSE,
    flags_user_verified BOOLEAN NOT NULL DEFAULT FALSE,
    flags_backup_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    flags_backup_state BOOLEAN NOT NULL DEFAULT FALSE,
    sign_count INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,  -- User-friendly label like "MacBook Pro Touch ID"
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES portal_users(id) ON DELETE CASCADE
);

CREATE INDEX idx_passkey_credentials_user_id ON passkey_credentials(user_id);
CREATE INDEX idx_passkey_credentials_credential_id ON passkey_credentials(credential_id);

-- +goose Down
DROP INDEX IF EXISTS idx_passkey_credentials_credential_id;
DROP INDEX IF EXISTS idx_passkey_credentials_user_id;
DROP TABLE IF EXISTS passkey_credentials;
```

### 1.2 SQLC Queries

**File:** `internal/database/queries/passkey_credentials.sql`

```sql
-- name: CreatePasskeyCredential :exec
INSERT INTO passkey_credentials (
    id, user_id, credential_id, public_key, attestation_type,
    transport, flags_user_present, flags_user_verified,
    flags_backup_eligible, flags_backup_state, sign_count, name
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPasskeyCredentialsByUserID :many
SELECT * FROM passkey_credentials WHERE user_id = ? ORDER BY created_at DESC;

-- name: GetPasskeyCredentialByCredentialID :one
SELECT * FROM passkey_credentials WHERE credential_id = ?;

-- name: GetPasskeyCredentialByID :one
SELECT * FROM passkey_credentials WHERE id = ?;

-- name: UpdatePasskeyCredentialSignCount :exec
UPDATE passkey_credentials
SET sign_count = ?, last_used_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdatePasskeyCredentialName :exec
UPDATE passkey_credentials SET name = ? WHERE id = ? AND user_id = ?;

-- name: DeletePasskeyCredential :exec
DELETE FROM passkey_credentials WHERE id = ? AND user_id = ?;

-- name: GetAllPasskeyCredentialsForLogin :many
SELECT pc.*, pu.username, pu.is_admin
FROM passkey_credentials pc
JOIN portal_users pu ON pc.user_id = pu.id
WHERE pu.enabled = 1;
```

### 1.3 WebAuthn Service

**File:** `internal/portal/passkey/service.go`

```go
package passkey

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/go-webauthn/webauthn/webauthn"
    "github.com/google/uuid"

    "slipstream/internal/database/sqlc"
)

type Service struct {
    webAuthn    *webauthn.WebAuthn
    queries     *sqlc.Queries
    challenges  sync.Map  // map[string]*ChallengeData
    config      Config
}

type Config struct {
    RPDisplayName string  // "SlipStream"
    RPID          string  // Domain: "localhost" or "slipstream.local"
    RPOrigins     []string // ["http://localhost:3000", "https://slipstream.local"]
}

type ChallengeData struct {
    SessionData *webauthn.SessionData
    UserID      int64
    ExpiresAt   time.Time
}

func NewService(queries *sqlc.Queries, config Config) (*Service, error) {
    wconfig := &webauthn.Config{
        RPDisplayName: config.RPDisplayName,
        RPID:          config.RPID,
        RPOrigins:     config.RPOrigins,
    }

    webAuthn, err := webauthn.New(wconfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create webauthn: %w", err)
    }

    s := &Service{
        webAuthn: webAuthn,
        queries:  queries,
        config:   config,
    }

    // Start challenge cleanup goroutine
    go s.cleanupExpiredChallenges()

    return s, nil
}

// Challenge management
func (s *Service) storeChallenge(challengeID string, data *ChallengeData) {
    data.ExpiresAt = time.Now().Add(5 * time.Minute)
    s.challenges.Store(challengeID, data)
}

func (s *Service) getChallenge(challengeID string) (*ChallengeData, bool) {
    val, ok := s.challenges.Load(challengeID)
    if !ok {
        return nil, false
    }
    data := val.(*ChallengeData)
    if time.Now().After(data.ExpiresAt) {
        s.challenges.Delete(challengeID)
        return nil, false
    }
    return data, true
}

func (s *Service) deleteChallenge(challengeID string) {
    s.challenges.Delete(challengeID)
}

func (s *Service) cleanupExpiredChallenges() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        now := time.Now()
        s.challenges.Range(func(key, value interface{}) bool {
            if data, ok := value.(*ChallengeData); ok {
                if now.After(data.ExpiresAt) {
                    s.challenges.Delete(key)
                }
            }
            return true
        })
    }
}
```

### 1.4 WebAuthn User Adapter

**File:** `internal/portal/passkey/user.go`

```go
package passkey

import (
    "slipstream/internal/database/sqlc"
    "github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnUser adapts portal_users for webauthn.User interface
type WebAuthnUser struct {
    ID          int64
    Username    string
    DisplayName string
    Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
    // Use stable user ID as bytes
    return []byte(fmt.Sprintf("%d", u.ID))
}

func (u *WebAuthnUser) WebAuthnName() string {
    return u.Username
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
    if u.DisplayName != "" {
        return u.DisplayName
    }
    return u.Username
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
    return u.Credentials
}

func (u *WebAuthnUser) WebAuthnIcon() string {
    return "" // Deprecated in WebAuthn spec
}

// Convert DB credentials to WebAuthn credentials
func credentialsFromDB(dbCreds []sqlc.PasskeyCredential) []webauthn.Credential {
    creds := make([]webauthn.Credential, len(dbCreds))
    for i, c := range dbCreds {
        var transport []protocol.AuthenticatorTransport
        if c.Transport.Valid {
            json.Unmarshal([]byte(c.Transport.String), &transport)
        }

        creds[i] = webauthn.Credential{
            ID:              c.CredentialID,
            PublicKey:       c.PublicKey,
            AttestationType: c.AttestationType,
            Transport:       transport,
            Flags: webauthn.CredentialFlags{
                UserPresent:    c.FlagsUserPresent,
                UserVerified:   c.FlagsUserVerified,
                BackupEligible: c.FlagsBackupEligible,
                BackupState:    c.FlagsBackupState,
            },
            Authenticator: webauthn.Authenticator{
                SignCount: uint32(c.SignCount),
            },
        }
    }
    return creds
}
```

### 1.5 Registration Flow

**File:** `internal/portal/passkey/registration.go`

```go
package passkey

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/google/uuid"

    "slipstream/internal/database/sqlc"
)

type BeginRegistrationResponse struct {
    ChallengeID string                                      `json:"challengeId"`
    Options     *protocol.CredentialCreation                `json:"options"`
}

type BeginRegistrationRequest struct {
    PIN string `json:"pin"`  // Required: verify identity before adding passkey
}

type FinishRegistrationRequest struct {
    ChallengeID string                            `json:"challengeId"`
    Name        string                            `json:"name"`
    Credential  *protocol.CredentialCreationResponse `json:"credential"`
}

func (s *Service) BeginRegistration(ctx context.Context, user *sqlc.PortalUser) (*BeginRegistrationResponse, error) {
    // Load existing credentials to exclude
    dbCreds, err := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get existing credentials: %w", err)
    }

    webAuthnUser := &WebAuthnUser{
        ID:          user.ID,
        Username:    user.Username,
        DisplayName: user.DisplayName.String,
        Credentials: credentialsFromDB(dbCreds),
    }

    options, session, err := s.webAuthn.BeginRegistration(webAuthnUser,
        webauthn.WithExclusions(webAuthnUser.WebAuthnCredentials()),
        webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
        webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
            UserVerification: protocol.VerificationPreferred,
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to begin registration: %w", err)
    }

    challengeID := uuid.NewString()
    s.storeChallenge(challengeID, &ChallengeData{
        SessionData: session,
        UserID:      user.ID,
    })

    return &BeginRegistrationResponse{
        ChallengeID: challengeID,
        Options:     options,
    }, nil
}

func (s *Service) FinishRegistration(ctx context.Context, user *sqlc.PortalUser, req *FinishRegistrationRequest) error {
    challenge, ok := s.getChallenge(req.ChallengeID)
    if !ok {
        return fmt.Errorf("challenge expired or not found")
    }
    defer s.deleteChallenge(req.ChallengeID)

    if challenge.UserID != user.ID {
        return fmt.Errorf("challenge user mismatch")
    }

    // Load existing credentials
    dbCreds, _ := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)
    webAuthnUser := &WebAuthnUser{
        ID:          user.ID,
        Username:    user.Username,
        DisplayName: user.DisplayName.String,
        Credentials: credentialsFromDB(dbCreds),
    }

    credential, err := s.webAuthn.FinishRegistration(webAuthnUser, *challenge.SessionData, req.Credential)
    if err != nil {
        return fmt.Errorf("failed to finish registration: %w", err)
    }

    // Serialize transports
    var transportJSON string
    if len(credential.Transport) > 0 {
        b, _ := json.Marshal(credential.Transport)
        transportJSON = string(b)
    }

    // Store credential
    err = s.queries.CreatePasskeyCredential(ctx, sqlc.CreatePasskeyCredentialParams{
        ID:                 uuid.NewString(),
        UserID:             user.ID,
        CredentialID:       credential.ID,
        PublicKey:          credential.PublicKey,
        AttestationType:    credential.AttestationType,
        Transport:          sqlc.NewNullString(transportJSON),
        FlagsUserPresent:   credential.Flags.UserPresent,
        FlagsUserVerified:  credential.Flags.UserVerified,
        FlagsBackupEligible: credential.Flags.BackupEligible,
        FlagsBackupState:   credential.Flags.BackupState,
        SignCount:          int64(credential.Authenticator.SignCount),
        Name:               req.Name,
    })
    if err != nil {
        return fmt.Errorf("failed to store credential: %w", err)
    }

    return nil
}
```

### 1.6 Authentication Flow

**File:** `internal/portal/passkey/authentication.go`

```go
package passkey

import (
    "context"
    "fmt"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/google/uuid"

    "slipstream/internal/database/sqlc"
)

type BeginLoginResponse struct {
    ChallengeID string                             `json:"challengeId"`
    Options     *protocol.CredentialAssertion      `json:"options"`
}

type FinishLoginRequest struct {
    ChallengeID string                            `json:"challengeId"`
    Credential  *protocol.CredentialAssertionResponse `json:"credential"`
}

type FinishLoginResponse struct {
    UserID   int64  `json:"userId"`
    Username string `json:"username"`
    IsAdmin  bool   `json:"isAdmin"`
}

// BeginLogin starts passkey authentication (discoverable credentials)
func (s *Service) BeginLogin(ctx context.Context) (*BeginLoginResponse, error) {
    options, session, err := s.webAuthn.BeginDiscoverableLogin(
        webauthn.WithUserVerification(protocol.VerificationPreferred),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to begin login: %w", err)
    }

    challengeID := uuid.NewString()
    s.storeChallenge(challengeID, &ChallengeData{
        SessionData: session,
        UserID:      0, // Unknown until credential is presented
    })

    return &BeginLoginResponse{
        ChallengeID: challengeID,
        Options:     options,
    }, nil
}

// FinishLogin completes passkey authentication
func (s *Service) FinishLogin(ctx context.Context, req *FinishLoginRequest) (*FinishLoginResponse, error) {
    challenge, ok := s.getChallenge(req.ChallengeID)
    if !ok {
        return nil, fmt.Errorf("challenge expired or not found")
    }
    defer s.deleteChallenge(req.ChallengeID)

    // Find user by credential (discoverable login)
    credential, err := s.webAuthn.FinishDiscoverableLogin(
        func(rawID, userHandle []byte) (webauthn.User, error) {
            return s.findUserByCredential(ctx, rawID, userHandle)
        },
        *challenge.SessionData,
        req.Credential,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to finish login: %w", err)
    }

    // Update sign count
    dbCred, err := s.queries.GetPasskeyCredentialByCredentialID(ctx, credential.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get credential: %w", err)
    }

    err = s.queries.UpdatePasskeyCredentialSignCount(ctx, sqlc.UpdatePasskeyCredentialSignCountParams{
        SignCount: int64(credential.Authenticator.SignCount),
        ID:        dbCred.ID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to update sign count: %w", err)
    }

    // Get user details
    user, err := s.queries.GetPortalUserByID(ctx, dbCred.UserID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    if user.Enabled == 0 {
        return nil, fmt.Errorf("user account is disabled")
    }

    // CRITICAL: Mirror PIN login security - both conditions required
    // This prevents privilege escalation if is_admin column is somehow modified
    isAdmin := user.IsAdmin == 1 && user.Username == "Administrator"

    return &FinishLoginResponse{
        UserID:   user.ID,
        Username: user.Username,
        IsAdmin:  isAdmin,
    }, nil
}

func (s *Service) findUserByCredential(ctx context.Context, rawID, userHandle []byte) (webauthn.User, error) {
    // Find credential by ID
    dbCred, err := s.queries.GetPasskeyCredentialByCredentialID(ctx, rawID)
    if err != nil {
        return nil, fmt.Errorf("credential not found")
    }

    // Get user
    user, err := s.queries.GetPortalUserByID(ctx, dbCred.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found")
    }

    // Get all credentials for this user
    dbCreds, _ := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)

    return &WebAuthnUser{
        ID:          user.ID,
        Username:    user.Username,
        DisplayName: user.DisplayName.String,
        Credentials: credentialsFromDB(dbCreds),
    }, nil
}
```

---

## Phase 2: API Endpoints

### 2.1 Passkey Handlers

**File:** `internal/portal/passkey/handlers.go`

```go
package passkey

import (
    "encoding/json"
    "net/http"

    "slipstream/internal/portal/auth"
)

type Handlers struct {
    service     *Service
    authService *auth.Service
}

func NewHandlers(service *Service, authService *auth.Service) *Handlers {
    return &Handlers{service: service, authService: authService}
}

// POST /api/v1/requests/auth/passkey/register/begin
func (h *Handlers) BeginRegistration(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    var req BeginRegistrationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    user, err := h.service.queries.GetPortalUserByID(r.Context(), claims.UserID)
    if err != nil {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }

    // CRITICAL: Verify PIN before allowing passkey registration
    // This proves the user is who they claim to be, not just someone with a stolen session
    if err := auth.ValidatePassword(user.PasswordHash, req.PIN); err != nil {
        http.Error(w, "invalid PIN", http.StatusUnauthorized)
        return
    }

    resp, err := h.service.BeginRegistration(r.Context(), &user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

// POST /api/v1/requests/auth/passkey/register/finish
func (h *Handlers) FinishRegistration(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    var req FinishRegistrationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    user, err := h.service.queries.GetPortalUserByID(r.Context(), claims.UserID)
    if err != nil {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }

    if err := h.service.FinishRegistration(r.Context(), &user, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// POST /api/v1/requests/auth/passkey/login/begin
func (h *Handlers) BeginLogin(w http.ResponseWriter, r *http.Request) {
    resp, err := h.service.BeginLogin(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

// POST /api/v1/requests/auth/passkey/login/finish
func (h *Handlers) FinishLogin(w http.ResponseWriter, r *http.Request) {
    var req FinishLoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    result, err := h.service.FinishLogin(r.Context(), &req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    // Generate JWT token based on user type
    var token string
    if result.IsAdmin {
        token, err = h.authService.GenerateAdminToken(result.UserID, result.Username)
    } else {
        user, _ := h.service.queries.GetPortalUserByID(r.Context(), result.UserID)
        token, err = h.authService.GeneratePortalToken(&user)
    }
    if err != nil {
        http.Error(w, "failed to generate token", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "token":    token,
        "user":     result,
    })
}

// GET /api/v1/requests/auth/passkey/credentials
func (h *Handlers) ListCredentials(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    creds, err := h.service.queries.GetPasskeyCredentialsByUserID(r.Context(), claims.UserID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return safe subset of credential data
    type CredentialInfo struct {
        ID         string  `json:"id"`
        Name       string  `json:"name"`
        CreatedAt  string  `json:"createdAt"`
        LastUsedAt *string `json:"lastUsedAt"`
    }

    result := make([]CredentialInfo, len(creds))
    for i, c := range creds {
        result[i] = CredentialInfo{
            ID:        c.ID,
            Name:      c.Name,
            CreatedAt: c.CreatedAt.Format(time.RFC3339),
        }
        if c.LastUsedAt.Valid {
            t := c.LastUsedAt.Time.Format(time.RFC3339)
            result[i].LastUsedAt = &t
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

// DELETE /api/v1/requests/auth/passkey/credentials/{id}
func (h *Handlers) DeleteCredential(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    credID := chi.URLParam(r, "id")

    // No need to check for "last auth method" - PIN/password is always required
    // and serves as the guaranteed fallback. Passkeys can always be deleted.

    err := h.service.queries.DeletePasskeyCredential(r.Context(), sqlc.DeletePasskeyCredentialParams{
        ID:     credID,
        UserID: claims.UserID,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// PUT /api/v1/requests/auth/passkey/credentials/{id}
func (h *Handlers) UpdateCredential(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    credID := chi.URLParam(r, "id")

    var req struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    err := h.service.queries.UpdatePasskeyCredentialName(r.Context(), sqlc.UpdatePasskeyCredentialNameParams{
        Name:   req.Name,
        ID:     credID,
        UserID: claims.UserID,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

### 2.2 Route Registration

**Add to:** `internal/api/server.go`

```go
// In setupRoutes() or equivalent

// Passkey routes (public - for login)
r.Post("/api/v1/requests/auth/passkey/login/begin", passkeyHandlers.BeginLogin)
r.Post("/api/v1/requests/auth/passkey/login/finish", passkeyHandlers.FinishLogin)

// Passkey routes (authenticated - for registration and management)
r.Group(func(r chi.Router) {
    r.Use(authMiddleware.AnyAuth())

    r.Post("/api/v1/requests/auth/passkey/register/begin", passkeyHandlers.BeginRegistration)
    r.Post("/api/v1/requests/auth/passkey/register/finish", passkeyHandlers.FinishRegistration)
    r.Get("/api/v1/requests/auth/passkey/credentials", passkeyHandlers.ListCredentials)
    r.Put("/api/v1/requests/auth/passkey/credentials/{id}", passkeyHandlers.UpdateCredential)
    r.Delete("/api/v1/requests/auth/passkey/credentials/{id}", passkeyHandlers.DeleteCredential)
})
```

---

## Phase 3: Frontend Implementation

### 3.1 Passkey API Client

**File:** `web/src/api/passkey.ts`

```typescript
import { startRegistration, startAuthentication } from '@simplewebauthn/browser'
import type {
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'
import { apiClient } from './client'

interface BeginRegistrationResponse {
  challengeId: string
  options: PublicKeyCredentialCreationOptionsJSON
}

interface BeginLoginResponse {
  challengeId: string
  options: PublicKeyCredentialRequestOptionsJSON
}

interface PasskeyCredential {
  id: string
  name: string
  createdAt: string
  lastUsedAt: string | null
}

interface LoginResponse {
  token: string
  user: {
    userId: number
    username: string
    isAdmin: boolean
  }
}

export const passkeyApi = {
  // Registration (requires authentication + PIN verification)
  async beginRegistration(pin: string): Promise<BeginRegistrationResponse> {
    const response = await apiClient.post('/requests/auth/passkey/register/begin', { pin })
    return response.data
  },

  async finishRegistration(challengeId: string, name: string, credential: Credential): Promise<void> {
    await apiClient.post('/requests/auth/passkey/register/finish', {
      challengeId,
      name,
      credential,
    })
  },

  async registerPasskey(pin: string, name: string): Promise<void> {
    const { challengeId, options } = await this.beginRegistration(pin)
    const credential = await startRegistration(options)
    await this.finishRegistration(challengeId, name, credential)
  },

  // Authentication (public)
  async beginLogin(): Promise<BeginLoginResponse> {
    const response = await apiClient.post('/requests/auth/passkey/login/begin')
    return response.data
  },

  async finishLogin(challengeId: string, credential: Credential): Promise<LoginResponse> {
    const response = await apiClient.post('/requests/auth/passkey/login/finish', {
      challengeId,
      credential,
    })
    return response.data
  },

  async loginWithPasskey(): Promise<LoginResponse> {
    const { challengeId, options } = await this.beginLogin()
    const credential = await startAuthentication(options)
    return this.finishLogin(challengeId, credential)
  },

  // Credential management (requires authentication)
  async listCredentials(): Promise<PasskeyCredential[]> {
    const response = await apiClient.get('/requests/auth/passkey/credentials')
    return response.data
  },

  async updateCredential(id: string, name: string): Promise<void> {
    await apiClient.put(`/requests/auth/passkey/credentials/${id}`, { name })
  },

  async deleteCredential(id: string): Promise<void> {
    await apiClient.delete(`/requests/auth/passkey/credentials/${id}`)
  },

  // Check if passkeys are supported
  isSupported(): boolean {
    return (
      window.PublicKeyCredential !== undefined &&
      typeof window.PublicKeyCredential === 'function'
    )
  },

  // Check if conditional UI (autofill) is supported
  async isConditionalUISupported(): Promise<boolean> {
    if (!this.isSupported()) return false
    return PublicKeyCredential.isConditionalMediationAvailable?.() ?? false
  },
}
```

### 3.2 React Hook

**File:** `web/src/hooks/usePasskey.ts`

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { passkeyApi } from '@/api/passkey'
import { usePortalAuthStore } from '@/stores/portalAuth'
import { toast } from 'sonner'

export function usePasskeySupport() {
  return {
    isSupported: passkeyApi.isSupported(),
  }
}

export function usePasskeyCredentials() {
  return useQuery({
    queryKey: ['passkey-credentials'],
    queryFn: () => passkeyApi.listCredentials(),
  })
}

export function useRegisterPasskey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ pin, name }: { pin: string; name: string }) =>
      passkeyApi.registerPasskey(pin, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['passkey-credentials'] })
      toast.success('Passkey registered successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to register passkey: ${error.message}`)
    },
  })
}

export function useDeletePasskey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => passkeyApi.deleteCredential(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['passkey-credentials'] })
      toast.success('Passkey deleted')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete passkey: ${error.message}`)
    },
  })
}

export function usePasskeyLogin() {
  const { setAuth } = usePortalAuthStore()

  return useMutation({
    mutationFn: () => passkeyApi.loginWithPasskey(),
    onSuccess: (data) => {
      setAuth(data.token, {
        id: data.user.userId,
        username: data.user.username,
        isAdmin: data.user.isAdmin,
      })
    },
    onError: (error: Error) => {
      toast.error(`Passkey login failed: ${error.message}`)
    },
  })
}

export function useUpdatePasskeyName() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      passkeyApi.updateCredential(id, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['passkey-credentials'] })
      toast.success('Passkey renamed')
    },
  })
}
```

### 3.3 Login Component Update

**File:** `web/src/routes/auth/Login.tsx` (modifications)

```tsx
import { usePasskeyLogin, usePasskeySupport } from '@/hooks/usePasskey'
import { KeyRound } from 'lucide-react'

function LoginPage() {
  const [showPinForm, setShowPinForm] = useState(false)
  const { isSupported } = usePasskeySupport()
  const passkeyLogin = usePasskeyLogin()

  const handlePasskeyLogin = async () => {
    try {
      await passkeyLogin.mutateAsync()
      // Navigation handled by auth store
    } catch (error) {
      // Error handled by mutation
    }
  }

  return (
    <div className="login-container">
      <h1>Sign In</h1>

      {isSupported && !showPinForm && (
        <>
          <Button
            onClick={handlePasskeyLogin}
            disabled={passkeyLogin.isPending}
            className="w-full"
          >
            <KeyRound className="mr-2 h-4 w-4" />
            {passkeyLogin.isPending ? 'Authenticating...' : 'Sign in with Passkey'}
          </Button>

          <button
            onClick={() => setShowPinForm(true)}
            className="text-sm text-muted-foreground hover:underline"
          >
            Use PIN instead
          </button>
        </>
      )}

      {(!isSupported || showPinForm) && (
        <>
          <PinLoginForm />

          {isSupported && (
            <button
              onClick={() => setShowPinForm(false)}
              className="text-sm text-muted-foreground hover:underline"
            >
              Use Passkey instead
            </button>
          )}
        </>
      )}
    </div>
  )
}
```

### 3.4 Passkey Management Component

**File:** `web/src/components/settings/PasskeyManager.tsx`

```tsx
import { useState } from 'react'
import { usePasskeyCredentials, useRegisterPasskey, useDeletePasskey, usePasskeySupport } from '@/hooks/usePasskey'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { KeyRound, Plus, Trash2, Pencil } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

export function PasskeyManager() {
  const [newPasskeyName, setNewPasskeyName] = useState('')
  const [pin, setPin] = useState('')
  const [isRegistering, setIsRegistering] = useState(false)

  const { isSupported } = usePasskeySupport()
  const { data: credentials, isLoading } = usePasskeyCredentials()
  const registerPasskey = useRegisterPasskey()
  const deletePasskey = useDeletePasskey()

  if (!isSupported) {
    return (
      <div className="text-muted-foreground">
        Passkeys are not supported in this browser.
      </div>
    )
  }

  const handleRegister = async () => {
    if (!newPasskeyName.trim() || !pin.trim()) return

    try {
      await registerPasskey.mutateAsync({ pin, name: newPasskeyName })
      setNewPasskeyName('')
      setPin('')
      setIsRegistering(false)
    } catch (error) {
      // Error handled by mutation
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Passkeys</h3>
        {!isRegistering && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsRegistering(true)}
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Passkey
          </Button>
        )}
      </div>

      {isRegistering && (
        <div className="space-y-3">
          <Input
            placeholder="Passkey name (e.g., MacBook Touch ID)"
            value={newPasskeyName}
            onChange={(e) => setNewPasskeyName(e.target.value)}
          />
          <Input
            type="password"
            placeholder="Enter your PIN to confirm"
            value={pin}
            onChange={(e) => setPin(e.target.value)}
          />
          <div className="flex gap-2">
            <Button
              onClick={handleRegister}
              disabled={registerPasskey.isPending || !newPasskeyName.trim() || !pin.trim()}
            >
              {registerPasskey.isPending ? 'Registering...' : 'Register Passkey'}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setIsRegistering(false)
                setNewPasskeyName('')
                setPin('')
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div>Loading...</div>
      ) : credentials?.length === 0 ? (
        <div className="text-muted-foreground">
          No passkeys registered. Add one for faster, more secure login.
        </div>
      ) : (
        <div className="space-y-2">
          {credentials?.map((cred) => (
            <div
              key={cred.id}
              className="flex items-center justify-between p-3 border rounded-lg"
            >
              <div className="flex items-center gap-3">
                <KeyRound className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="font-medium">{cred.name}</div>
                  <div className="text-sm text-muted-foreground">
                    Created {formatDistanceToNow(new Date(cred.createdAt))} ago
                    {cred.lastUsedAt && (
                      <> · Last used {formatDistanceToNow(new Date(cred.lastUsedAt))} ago</>
                    )}
                  </div>
                </div>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => deletePasskey.mutate(cred.id)}
                disabled={deletePasskey.isPending}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

---

## Phase 4: Configuration & Initialization

### 4.1 WebAuthn Configuration

**Add to:** `internal/config/config.go`

```go
type WebAuthnConfig struct {
    RPDisplayName string   `yaml:"rpDisplayName" env:"WEBAUTHN_RP_DISPLAY_NAME"`
    RPID          string   `yaml:"rpId" env:"WEBAUTHN_RP_ID"`
    RPOrigins     []string `yaml:"rpOrigins" env:"WEBAUTHN_RP_ORIGINS"`
}

// In Config struct
WebAuthn WebAuthnConfig `yaml:"webauthn"`

// Defaults
func DefaultConfig() *Config {
    return &Config{
        // ...
        WebAuthn: WebAuthnConfig{
            RPDisplayName: "SlipStream",
            RPID:          "localhost",
            RPOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
        },
    }
}
```

### 4.2 Service Initialization

**Add to:** `internal/api/server.go` (or service initialization)

```go
import "slipstream/internal/portal/passkey"

// In server setup
passkeyConfig := passkey.Config{
    RPDisplayName: cfg.WebAuthn.RPDisplayName,
    RPID:          cfg.WebAuthn.RPID,
    RPOrigins:     cfg.WebAuthn.RPOrigins,
}

passkeyService, err := passkey.NewService(queries, passkeyConfig)
if err != nil {
    return fmt.Errorf("failed to create passkey service: %w", err)
}

passkeyHandlers := passkey.NewHandlers(passkeyService, authService)
```

---

## Phase 5: Testing

### 5.1 Backend Tests

**File:** `internal/portal/passkey/service_test.go`

```go
func TestChallengeExpiration(t *testing.T) {
    // Test that challenges expire after 5 minutes
}

func TestRegistrationFlow(t *testing.T) {
    // Test begin/finish registration
}

func TestAuthenticationFlow(t *testing.T) {
    // Test begin/finish login
}

func TestSignCountValidation(t *testing.T) {
    // Test replay attack prevention
}

func TestCredentialManagement(t *testing.T) {
    // Test list, update, delete operations
}
```

### 5.2 Frontend Tests

- Test passkey support detection
- Test registration flow with mocked WebAuthn
- Test login flow with mocked WebAuthn
- Test credential management UI

---

## Implementation Checklist

### Phase 0: Setup & Cleanup
- [x] Remove `UpdatePortalUserAdmin` query from `internal/database/queries/portal_users.sql`
- [x] Run `sqlc generate` to remove generated code
- [x] Add backend dependency: `go get github.com/go-webauthn/webauthn`
- [x] Add frontend dependency: `cd web && bun add @simplewebauthn/browser`
- [x] **Bonus**: Deleted dead `internal/auth/` package (legacy unused code)
- [x] **Bonus**: Moved `internal/portal/auth/` → `internal/auth/` for cleaner structure

### Phase 1: Database & Backend Foundation
- [x] Create migration `041_passkey_credentials.sql`
- [x] Create SQLC queries `passkey_credentials.sql`
- [x] Run `sqlc generate`
- [x] Create passkey service in `internal/auth/passkey.go` (note: changed from plan's `internal/portal/passkey/`)
- [x] Implement `PasskeyService` with challenge storage
- [x] Implement `WebAuthnUser` adapter
- [x] Implement registration flow
- [x] Implement authentication flow

### Phase 2: API Endpoints
- [x] Create passkey handlers (`internal/auth/passkey_handlers.go`)
- [x] Register routes in server
- [ ] Test endpoints with curl/Postman

### Phase 3: Frontend Implementation
- [x] Create passkey API client (`web/src/api/portal/passkey.ts`)
- [x] Create React hooks (`web/src/hooks/portal/usePasskey.ts`)
- [x] Update login page with passkey option (`web/src/routes/requests/auth/login.tsx`)
- [x] Create PasskeyManager component (`web/src/components/portal/PasskeyManager.tsx`)
- [x] Add PasskeyManager to user settings page (`web/src/routes/requests/settings.tsx`)

### Phase 4: Configuration
- [x] Add WebAuthn config to config.go (`PortalConfig.WebAuthn`)
- [x] Update default config with WebAuthn defaults
- [x] Initialize PasskeyService in server setup

### Phase 5: Testing
- [ ] Backend unit tests
- [ ] Integration tests
- [ ] Manual testing with real authenticators

---

## Security Considerations

1. **Challenge Expiration**: 5-minute TTL prevents replay attacks
2. **Sign Count**: Track and validate to detect cloned authenticators
3. **User Verification**: Prefer UV to ensure user presence
4. **Credential Exclusion**: Prevent re-registration of same credential
5. **Origin Validation**: Strict origin checking in WebAuthn config
6. **PIN Verification Before Registration**: User must verify PIN before adding a passkey (proves identity)

### Critical: Admin Token Issuance

**The existing PIN login has a security check that passkey login MUST replicate:**

PIN login (secure):
```go
// Only routes to admin login if username is exactly "Administrator"
if req.Username == "Administrator" {
    return h.handleAdminLogin(...)
}
```

This means even if `is_admin = 1` in the database for a portal user, they cannot obtain an admin JWT because their username isn't "Administrator".

**Passkey login MUST maintain this invariant:**

```go
// In passkey FinishLogin - both conditions required
isAdmin := user.IsAdmin == 1 && user.Username == "Administrator"

return &FinishLoginResponse{
    UserID:   user.ID,
    Username: user.Username,
    IsAdmin:  isAdmin,  // Only true if BOTH conditions met
}, nil
```

**Why this matters:**
- The `is_admin` column alone is not sufficient for admin access
- Username "Administrator" is hardcoded as the admin identifier
- This provides defense-in-depth: an attacker would need to modify BOTH the `is_admin` flag AND the username (which has a unique constraint)
- Passkey auth must not bypass this protection by trusting `is_admin` alone

---

## Future Enhancements (Out of Scope)

- Conditional UI (autofill passkey suggestions)
- Cross-device authentication (hybrid transport)
- Admin ability to manage user passkeys
