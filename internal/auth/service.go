package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/portal"
)

const (
	TokenExpiryPortal = 30 * 24 * time.Hour // 30 days for portal users
	TokenExpiryAdmin  = 24 * time.Hour      // 24 hours for admin users
)

var (
	ErrInvalidCredentials = portal.ErrInvalidCredentials
	ErrUserDisabled       = portal.ErrUserDisabled
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrPasswordRequired   = portal.ErrPasswordRequired
)

type Service struct {
	queries   *sqlc.Queries
	jwtSecret []byte
}

//nolint:gosec // variable name, not a credential
const jwtSecretSettingKey = "portal_jwt_secret"

func NewService(queries *sqlc.Queries, jwtSecret string) (*Service, error) {
	secret := []byte(jwtSecret)

	if len(secret) == 0 {
		var err error
		secret, err = loadOrGenerateSecret(queries)
		if err != nil {
			return nil, err
		}
	}

	return &Service{
		queries:   queries,
		jwtSecret: secret,
	}, nil
}

func loadOrGenerateSecret(queries *sqlc.Queries) ([]byte, error) {
	ctx := context.Background()
	setting, err := queries.GetSetting(ctx, jwtSecretSettingKey)

	switch {
	case err == nil && setting.Value != "":
		secret, decErr := hex.DecodeString(setting.Value)
		if decErr != nil {
			return nil, fmt.Errorf("failed to decode stored JWT secret: %w", decErr)
		}
		return secret, nil

	case errors.Is(err, sql.ErrNoRows) || (err == nil && setting.Value == ""):
		return generateAndPersistSecret(queries)

	default:
		return nil, fmt.Errorf("failed to load JWT secret from database: %w", err)
	}
}

func generateAndPersistSecret(queries *sqlc.Queries) ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	_, err := queries.SetSetting(context.Background(), sqlc.SetSettingParams{
		Key:   jwtSecretSettingKey,
		Value: hex.EncodeToString(secret),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist JWT secret: %w", err)
	}
	return secret, nil
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
	// Reload JWT secret from the new database
	ctx := context.Background()
	setting, err := queries.GetSetting(ctx, jwtSecretSettingKey)
	if err == nil && setting.Value != "" {
		secret, err := hex.DecodeString(setting.Value)
		if err == nil {
			s.jwtSecret = secret
			return
		}
	}
	// If no secret in database, generate and store one
	if errors.Is(err, sql.ErrNoRows) || (err == nil && setting.Value == "") {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return
		}
		_, err = queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   jwtSecretSettingKey,
			Value: hex.EncodeToString(secret),
		})
		if err == nil {
			s.jwtSecret = secret
		}
	}
}

func (s *Service) GeneratePortalToken(user *sqlc.PortalUser) (string, error) {
	claims := &portal.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     portal.RoleUser,
		Audience: portal.AudiencePortal,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiryPortal)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "slipstream-portal",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) GenerateAdminToken(userID int64, username string) (string, error) {
	claims := &portal.Claims{
		UserID:   userID,
		Username: username,
		Role:     portal.RoleAdmin,
		Audience: portal.AudienceAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiryAdmin)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "slipstream-portal",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) ValidateToken(tokenString string) (*portal.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &portal.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*portal.Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Service) ValidatePortalToken(tokenString string) (*portal.Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Audience != portal.AudiencePortal {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Service) ValidateAdminToken(tokenString string) (*portal.Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Audience != portal.AudienceAdmin {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

var HashPassword = portal.HashPassword
var ValidatePassword = portal.ValidatePassword
