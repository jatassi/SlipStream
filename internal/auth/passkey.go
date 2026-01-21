package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

type PasskeyService struct {
	webAuthn   *webauthn.WebAuthn
	queries    *sqlc.Queries
	challenges sync.Map // map[string]*ChallengeData
	config     PasskeyConfig
}

type PasskeyConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

type ChallengeData struct {
	SessionData *webauthn.SessionData
	UserID      int64
	ExpiresAt   time.Time
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
	wconfig := &webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          config.RPID,
		RPOrigins:     config.RPOrigins,
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	s := &PasskeyService{
		webAuthn: webAuthn,
		queries:  queries,
		config:   config,
	}

	go s.cleanupExpiredChallenges()

	return s, nil
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
	data := val.(*ChallengeData)
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
func (s *PasskeyService) BeginRegistration(ctx context.Context, user *sqlc.PortalUser) (*BeginRegistrationResponse, error) {
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

	exclusions := credentialDescriptorsFromDB(dbCreds)

	options, session, err := s.webAuthn.BeginRegistration(webAuthnUser,
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
	})

	return &BeginRegistrationResponse{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

// FinishRegistration completes passkey registration using the HTTP request
func (s *PasskeyService) FinishRegistration(ctx context.Context, user *sqlc.PortalUser, challengeID string, name string, r *http.Request) error {
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
		DisplayName: user.DisplayName.String,
		Credentials: credentialsFromDB(dbCreds),
	}

	credential, err := s.webAuthn.FinishRegistration(webAuthnUser, *challenge.SessionData, r)
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
func (s *PasskeyService) BeginLogin(ctx context.Context) (*BeginLoginResponse, error) {
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

// FinishLogin completes passkey authentication using the HTTP request
func (s *PasskeyService) FinishLogin(ctx context.Context, challengeID string, r *http.Request) (*FinishLoginResult, error) {
	challenge, ok := s.getChallenge(challengeID)
	if !ok {
		return nil, fmt.Errorf("challenge expired or not found")
	}
	defer s.deleteChallenge(challengeID)

	var foundUser *sqlc.PortalUser
	var foundCredential *sqlc.PasskeyCredential

	credential, err := s.webAuthn.FinishDiscoverableLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			dbCred, err := s.queries.GetPasskeyCredentialByCredentialID(ctx, rawID)
			if err != nil {
				return nil, fmt.Errorf("credential not found")
			}
			foundCredential = dbCred

			user, err := s.queries.GetPortalUser(ctx, dbCred.UserID)
			if err != nil {
				return nil, fmt.Errorf("user not found")
			}
			foundUser = user

			dbCreds, _ := s.queries.GetPasskeyCredentialsByUserID(ctx, user.ID)

			return &WebAuthnUser{
				ID:          user.ID,
				Username:    user.Username,
				DisplayName: user.DisplayName.String,
				Credentials: credentialsFromDB(dbCreds),
			}, nil
		},
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

	if foundUser.Enabled == 0 {
		return nil, fmt.Errorf("user account is disabled")
	}

	// CRITICAL: Mirror PIN login security - both conditions required
	// This prevents privilege escalation if is_admin column is somehow modified
	isAdmin := foundUser.IsAdmin == 1 && foundUser.Username == "Administrator"

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
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
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

func credentialsFromDB(dbCreds []*sqlc.PasskeyCredential) []webauthn.Credential {
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

func credentialDescriptorsFromDB(dbCreds []*sqlc.PasskeyCredential) []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, len(dbCreds))
	for i, c := range dbCreds {
		var transport []protocol.AuthenticatorTransport
		if c.Transport.Valid {
			json.Unmarshal([]byte(c.Transport.String), &transport)
		}

		descriptors[i] = protocol.CredentialDescriptor{
			Type:            protocol.PublicKeyCredentialType,
			CredentialID:    c.CredentialID,
			Transport:       transport,
		}
	}
	return descriptors
}
