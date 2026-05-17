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

// ChallengeData carries the pending WebAuthn session along with the RP context
// (RPID and Origin) selected at Begin time. Storing those on the challenge lets
// FinishLogin/FinishRegistration reconstruct the same WebAuthn instance even
// though the server supports multiple hostnames.
type ChallengeData struct {
	SessionData *webauthn.SessionData
	UserID      int64
	ExpiresAt   time.Time
	RPID        string
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
	// Validate the fallback config eagerly so misconfiguration is surfaced at
	// startup rather than on the first sign-in attempt. The instance itself is
	// constructed per-request to support multi-hostname deployments.
	if _, err := webauthn.New(&webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          config.RPID,
		RPOrigins:     config.RPOrigins,
	}); err != nil {
		return nil, fmt.Errorf("failed to validate webauthn config: %w", err)
	}

	s := &PasskeyService{
		queries: queries,
		config:  config,
	}

	go s.cleanupExpiredChallenges()

	return s, nil
}

// resolveRPForRequest picks the RP ID and Origin for an incoming request.
// It matches the request's Host against the configured RPOrigins and uses
// the matching origin's hostname as the RP ID. When no match is found it
// falls back to the configured RPID and the first configured origin so
// single-host deployments keep their previous behavior.
func (s *PasskeyService) resolveRPForRequest(r *http.Request) (rpID, origin string) {
	reqHost := strings.ToLower(hostnameOnly(r.Host))
	if reqHost != "" {
		for _, o := range s.config.RPOrigins {
			oHost := strings.ToLower(hostnameOnly(originHost(o)))
			if oHost != "" && oHost == reqHost {
				return reqHost, o
			}
		}
	}

	fallbackOrigin := ""
	if len(s.config.RPOrigins) > 0 {
		fallbackOrigin = s.config.RPOrigins[0]
	}
	return s.config.RPID, fallbackOrigin
}

// webAuthnFor builds a fresh WebAuthn instance for the given RP context. The
// origin list always contains exactly the chosen origin, which keeps the
// Finish verification strict and predictable.
func (s *PasskeyService) webAuthnFor(rpID, origin string) (*webauthn.WebAuthn, error) {
	origins := s.config.RPOrigins
	if origin != "" {
		origins = []string{origin}
	}
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: s.config.RPDisplayName,
		RPID:          rpID,
		RPOrigins:     origins,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}
	return w, nil
}

// hostnameOnly strips an optional port from a Host or hostname:port string.
func hostnameOnly(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// originHost extracts the host portion of an origin URL, tolerating values
// stored without a scheme (e.g. just "example.com:443").
func originHost(origin string) string {
	if origin == "" {
		return ""
	}
	if u, err := url.Parse(origin); err == nil && u.Host != "" {
		return u.Host
	}
	return origin
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
func (s *PasskeyService) BeginRegistration(ctx context.Context, user *sqlc.PortalUser, r *http.Request) (*BeginRegistrationResponse, error) {
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

	rpID, origin := s.resolveRPForRequest(r)
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
		RPID:        rpID,
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

	w, err := s.webAuthnFor(challenge.RPID, challenge.Origin)
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
func (s *PasskeyService) BeginLogin(ctx context.Context, r *http.Request) (*BeginLoginResponse, error) {
	rpID, origin := s.resolveRPForRequest(r)
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
		RPID:        rpID,
		Origin:      origin,
	})

	return &BeginLoginResponse{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

// discoverableUserLookup returns a webauthn.DiscoverableUserHandler that resolves
// a presented credential to its owning portal user, recording the resolved
// credential and user via the supplied pointers for the caller to use after
// assertion completes.
func (s *PasskeyService) discoverableUserLookup(
	ctx context.Context,
	foundCredential **sqlc.PasskeyCredential,
	foundUser **sqlc.PortalUser,
) webauthn.DiscoverableUserHandler {
	return func(rawID, _ []byte) (webauthn.User, error) {
		dbCred, err := s.queries.GetPasskeyCredentialByCredentialID(ctx, rawID)
		if err != nil {
			return nil, fmt.Errorf("credential not found")
		}
		*foundCredential = dbCred

		user, err := s.queries.GetPortalUser(ctx, dbCred.UserID)
		if err != nil {
			return nil, fmt.Errorf("user not found")
		}
		*foundUser = user

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

	var foundUser *sqlc.PortalUser
	var foundCredential *sqlc.PasskeyCredential

	w, err := s.webAuthnFor(challenge.RPID, challenge.Origin)
	if err != nil {
		return nil, err
	}

	credential, err := w.FinishDiscoverableLogin(
		s.discoverableUserLookup(ctx, &foundCredential, &foundUser),
		*challenge.SessionData,
		r,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %w", err)
	}

	if foundCredential == nil || foundUser == nil {
		return nil, fmt.Errorf("credential lookup failed")
	}

	err = s.queries.UpdatePasskeyCredentialSignCount(ctx, sqlc.UpdatePasskeyCredentialSignCountParams{
		SignCount: int64(credential.Authenticator.SignCount),
		ID:        foundCredential.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update sign count: %w", err)
	}

	if !foundUser.Enabled {
		return nil, fmt.Errorf("user account is disabled")
	}

	// CRITICAL: Mirror PIN login security - both conditions required
	// This prevents privilege escalation if is_admin column is somehow modified
	isAdmin := foundUser.IsAdmin && foundUser.Username == "Administrator"

	return &FinishLoginResult{
		UserID:   foundUser.ID,
		Username: foundUser.Username,
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
