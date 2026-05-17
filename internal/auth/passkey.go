package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

type PasskeyService struct {
	queries    *sqlc.Queries
	challenges sync.Map // map[string]*ChallengeData
	config     PasskeyConfig
}

type PasskeyConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

// ChallengeData carries the pending WebAuthn session along with the Origin
// selected at Begin time. The matching RP ID is read back from
// SessionData.RelyingPartyID at Finish; storing the chosen Origin lets us
// reconstruct an equivalent WebAuthn instance even when the server is reached
// via more than one hostname.
type ChallengeData struct {
	SessionData *webauthn.SessionData
	UserID      int64
	ExpiresAt   time.Time
	Origin      string
}

type BeginRegistrationResponse struct {
	ChallengeID string                       `json:"challengeId"`
	Options     *protocol.CredentialCreation `json:"options"`
}

type BeginLoginResponse struct {
	ChallengeID string                        `json:"challengeId"`
	Options     *protocol.CredentialAssertion `json:"options"`
}

type FinishLoginResult struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
}

type PasskeyCredentialInfo struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"createdAt"`
	LastUsedAt *string `json:"lastUsedAt"`
}

func NewPasskeyService(queries *sqlc.Queries, config PasskeyConfig) (*PasskeyService, error) {
	// Validate eagerly so misconfiguration surfaces at startup rather than on
	// the first sign-in attempt. The runtime instance is constructed per-request
	// to support multi-hostname deployments.
	if _, err := webauthn.New(buildWebAuthnConfig(config, config.RPID, config.RPOrigins)); err != nil {
		return nil, fmt.Errorf("failed to validate webauthn config: %w", err)
	}

	s := &PasskeyService{
		queries: queries,
		config:  config,
	}

	go s.cleanupExpiredChallenges()

	return s, nil
}

// buildWebAuthnConfig assembles a webauthn.Config from the service-wide settings
// and the (rpID, origins) chosen for a specific request. Keeping construction in
// one place prevents skew between startup validation and per-request use.
func buildWebAuthnConfig(cfg PasskeyConfig, rpID string, origins []string) *webauthn.Config {
	return &webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          rpID,
		RPOrigins:     origins,
	}
}

// resolveRPForRequest picks the RP ID and Origin for an incoming request by
// matching the request host against the configured RPOrigins. Returns an error
// when no origin matches: silently falling through would issue a challenge the
// browser is guaranteed to reject at Finish time, producing a confusing failure
// mode and leaving stale challenges in the in-memory map.
func (s *PasskeyService) resolveRPForRequest(host string) (rpID, origin string, err error) {
	reqHost := requestHostname(host)
	if reqHost == "" {
		return "", "", fmt.Errorf("request host is empty")
	}
	for _, o := range s.config.RPOrigins {
		if oHost := normalizedOriginHost(o); oHost != "" && oHost == reqHost {
			return reqHost, o, nil
		}
	}
	return "", "", fmt.Errorf("host %q is not a configured WebAuthn origin", reqHost)
}

// webAuthnFor builds a fresh WebAuthn instance for the given RP context. The
// origin list always contains exactly the chosen origin so Finish verification
// rejects anything that doesn't match the Begin-time decision.
func (s *PasskeyService) webAuthnFor(rpID, origin string) (*webauthn.WebAuthn, error) {
	if rpID == "" || origin == "" {
		return nil, fmt.Errorf("missing RP ID or origin")
	}
	w, err := webauthn.New(buildWebAuthnConfig(s.config, rpID, []string{origin}))
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}
	return w, nil
}

// requestHostname normalizes the host portion of an incoming HTTP request's
// Host header (which may include a port).
func requestHostname(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(host)
}

// normalizedOriginHost extracts the lowercase hostname from a configured
// RPOrigin URL. Entries without a parseable scheme/host return empty and are
// skipped — operators are expected to store full origins (validated upstream
// by webauthn.New at startup).
func normalizedOriginHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func (s *PasskeyService) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *PasskeyService) storeChallenge(challengeID string, data *ChallengeData) {
	data.ExpiresAt = time.Now().Add(5 * time.Minute)
	s.challenges.Store(challengeID, data)
}

func (s *PasskeyService) getChallenge(challengeID string) (*ChallengeData, bool) {
	val, ok := s.challenges.Load(challengeID)
	if !ok {
		return nil, false
	}
	data, ok := val.(*ChallengeData)
	if !ok {
		return nil, false
	}
	if time.Now().After(data.ExpiresAt) {
		s.challenges.Delete(challengeID)
		return nil, false
	}
	return data, true
}

func (s *PasskeyService) deleteChallenge(challengeID string) {
	s.challenges.Delete(challengeID)
}

func (s *PasskeyService) cleanupExpiredChallenges() {
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

// BeginRegistration starts passkey registration for an authenticated user
func (s *PasskeyService) BeginRegistration(ctx context.Context, user *sqlc.PortalUser, host string) (*BeginRegistrationResponse, error) {
	dbCreds, err := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing credentials: %w", err)
	}

	webAuthnUser := &WebAuthnUser{
		ID:          user.ID,
		Username:    user.Username,
		Credentials: credentialsFromDB(dbCreds),
	}

	exclusions := credentialDescriptorsFromDB(dbCreds)

	rpID, origin, err := s.resolveRPForRequest(host)
	if err != nil {
		return nil, err
	}
	w, err := s.webAuthnFor(rpID, origin)
	if err != nil {
		return nil, err
	}

	options, session, err := w.BeginRegistration(webAuthnUser,
		webauthn.WithExclusions(exclusions),
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
		Origin:      origin,
	})

	return &BeginRegistrationResponse{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

// FinishRegistration completes passkey registration using the HTTP request
func (s *PasskeyService) FinishRegistration(ctx context.Context, user *sqlc.PortalUser, challengeID, name string, r *http.Request) error {
	challenge, ok := s.getChallenge(challengeID)
	if !ok {
		return fmt.Errorf("challenge expired or not found")
	}
	defer s.deleteChallenge(challengeID)

	if challenge.UserID != user.ID {
		return fmt.Errorf("challenge user mismatch")
	}

	dbCreds, _ := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)
	webAuthnUser := &WebAuthnUser{
		ID:          user.ID,
		Username:    user.Username,
		Credentials: credentialsFromDB(dbCreds),
	}

	w, err := s.webAuthnFor(challenge.SessionData.RelyingPartyID, challenge.Origin)
	if err != nil {
		return err
	}

	credential, err := w.FinishRegistration(webAuthnUser, *challenge.SessionData, r)
	if err != nil {
		return fmt.Errorf("failed to finish registration: %w", err)
	}

	var transportJSON sql.NullString
	if len(credential.Transport) > 0 {
		b, _ := json.Marshal(credential.Transport)
		transportJSON = sql.NullString{String: string(b), Valid: true}
	}

	err = s.queries.CreatePasskeyCredential(ctx, sqlc.CreatePasskeyCredentialParams{
		ID:                  uuid.NewString(),
		UserID:              user.ID,
		CredentialID:        credential.ID,
		PublicKey:           credential.PublicKey,
		AttestationType:     credential.AttestationType,
		Transport:           transportJSON,
		FlagsUserPresent:    credential.Flags.UserPresent,
		FlagsUserVerified:   credential.Flags.UserVerified,
		FlagsBackupEligible: credential.Flags.BackupEligible,
		FlagsBackupState:    credential.Flags.BackupState,
		SignCount:           int64(credential.Authenticator.SignCount),
		Name:                name,
	})
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	return nil
}

// BeginLogin starts passkey authentication (discoverable credentials)
func (s *PasskeyService) BeginLogin(ctx context.Context, host string) (*BeginLoginResponse, error) {
	rpID, origin, err := s.resolveRPForRequest(host)
	if err != nil {
		return nil, err
	}
	w, err := s.webAuthnFor(rpID, origin)
	if err != nil {
		return nil, err
	}

	options, session, err := w.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationPreferred),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	challengeID := uuid.NewString()
	s.storeChallenge(challengeID, &ChallengeData{
		SessionData: session,
		UserID:      0, // Unknown until credential is presented
		Origin:      origin,
	})

	return &BeginLoginResponse{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

// discoverableUserResult captures the credential and user resolved during a
// discoverable-credential assertion so the caller can inspect them after
// FinishDiscoverableLogin returns.
type discoverableUserResult struct {
	Credential *sqlc.PasskeyCredential
	User       *sqlc.PortalUser
}

// discoverableUserLookup returns a webauthn.DiscoverableUserHandler that
// resolves a presented credential to its owning portal user and records both
// on result.
func (s *PasskeyService) discoverableUserLookup(ctx context.Context, result *discoverableUserResult) webauthn.DiscoverableUserHandler {
	return func(rawID, _ []byte) (webauthn.User, error) {
		dbCred, err := s.queries.GetPasskeyCredentialByCredentialID(ctx, rawID)
		if err != nil {
			return nil, fmt.Errorf("credential not found")
		}
		result.Credential = dbCred

		user, err := s.queries.GetPortalUser(ctx, dbCred.UserID)
		if err != nil {
			return nil, fmt.Errorf("user not found")
		}
		result.User = user

		dbCreds, _ := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)

		return &WebAuthnUser{
			ID:          user.ID,
			Username:    user.Username,
			Credentials: credentialsFromDB(dbCreds),
		}, nil
	}
}

// FinishLogin completes passkey authentication using the HTTP request
func (s *PasskeyService) FinishLogin(ctx context.Context, challengeID string, r *http.Request) (*FinishLoginResult, error) {
	challenge, ok := s.getChallenge(challengeID)
	if !ok {
		return nil, fmt.Errorf("challenge expired or not found")
	}
	defer s.deleteChallenge(challengeID)

	w, err := s.webAuthnFor(challenge.SessionData.RelyingPartyID, challenge.Origin)
	if err != nil {
		return nil, err
	}

	lookup := &discoverableUserResult{}
	credential, err := w.FinishDiscoverableLogin(
		s.discoverableUserLookup(ctx, lookup),
		*challenge.SessionData,
		r,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %w", err)
	}

	if lookup.Credential == nil || lookup.User == nil {
		return nil, fmt.Errorf("credential lookup failed")
	}

	err = s.queries.UpdatePasskeyCredentialSignCount(ctx, sqlc.UpdatePasskeyCredentialSignCountParams{
		SignCount: int64(credential.Authenticator.SignCount),
		ID:        lookup.Credential.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update sign count: %w", err)
	}

	if !lookup.User.Enabled {
		return nil, fmt.Errorf("user account is disabled")
	}

	// CRITICAL: Mirror PIN login security - both conditions required
	// This prevents privilege escalation if is_admin column is somehow modified
	isAdmin := lookup.User.IsAdmin && lookup.User.Username == "Administrator"

	return &FinishLoginResult{
		UserID:   lookup.User.ID,
		Username: lookup.User.Username,
		IsAdmin:  isAdmin,
	}, nil
}

// ListCredentials returns all passkeys for a user
func (s *PasskeyService) ListCredentials(ctx context.Context, userID int64) ([]PasskeyCredentialInfo, error) {
	creds, err := s.queries.GetPasskeyCredentialsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]PasskeyCredentialInfo, len(creds))
	for i, c := range creds {
		result[i] = PasskeyCredentialInfo{
			ID:        c.ID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		}
		if c.LastUsedAt.Valid {
			t := c.LastUsedAt.Time.Format(time.RFC3339)
			result[i].LastUsedAt = &t
		}
	}

	return result, nil
}

// UpdateCredentialName renames a passkey
func (s *PasskeyService) UpdateCredentialName(ctx context.Context, credID string, userID int64, name string) error {
	return s.queries.UpdatePasskeyCredentialName(ctx, sqlc.UpdatePasskeyCredentialNameParams{
		Name:   name,
		ID:     credID,
		UserID: userID,
	})
}

// DeleteCredential removes a passkey
func (s *PasskeyService) DeleteCredential(ctx context.Context, credID string, userID int64) error {
	return s.queries.DeletePasskeyCredential(ctx, sqlc.DeletePasskeyCredentialParams{
		ID:     credID,
		UserID: userID,
	})
}

// WebAuthnUser adapts portal_users for webauthn.User interface
type WebAuthnUser struct {
	ID          int64
	Username    string
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(fmt.Sprintf("%d", u.ID))
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.Username
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.Username
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func (u *WebAuthnUser) WebAuthnIcon() string {
	return "" // Deprecated in WebAuthn spec
}

func credentialsFromDB(dbCreds []*sqlc.PasskeyCredential) []webauthn.Credential {
	creds := make([]webauthn.Credential, len(dbCreds))
	for i, c := range dbCreds {
		var transport []protocol.AuthenticatorTransport
		if c.Transport.Valid {
			_ = json.Unmarshal([]byte(c.Transport.String), &transport)
		}

		// Check for integer overflow before conversion
		if c.SignCount < 0 || c.SignCount > 4294967295 {
			continue // Skip invalid sign count
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

func credentialDescriptorsFromDB(dbCreds []*sqlc.PasskeyCredential) []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, len(dbCreds))
	for i, c := range dbCreds {
		var transport []protocol.AuthenticatorTransport
		if c.Transport.Valid {
			_ = json.Unmarshal([]byte(c.Transport.String), &transport)
		}

		descriptors[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.CredentialID,
			Transport:    transport,
		}
	}
	return descriptors
}
