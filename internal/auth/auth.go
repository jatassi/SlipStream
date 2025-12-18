package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoPasswordSet      = errors.New("no password has been set")
	ErrPasswordRequired   = errors.New("password is required")
)

// Service handles authentication operations.
type Service struct {
	db        *sql.DB
	jwtSecret []byte
}

// Claims represents JWT claims.
type Claims struct {
	jwt.RegisteredClaims
}

// NewService creates a new auth service.
func NewService(db *sql.DB, jwtSecret string) (*Service, error) {
	secret := []byte(jwtSecret)

	// Generate random secret if not provided
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}
	}

	return &Service{
		db:        db,
		jwtSecret: secret,
	}, nil
}

// SetPassword sets or updates the authentication password.
func (s *Service) SetPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Upsert the password
	_, err = s.db.Exec(`
		INSERT INTO auth (id, password_hash, updated_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			password_hash = excluded.password_hash,
			updated_at = CURRENT_TIMESTAMP
	`, string(hash))

	if err != nil {
		return fmt.Errorf("failed to save password: %w", err)
	}

	return nil
}

// ValidatePassword checks if the provided password is correct.
func (s *Service) ValidatePassword(password string) error {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM auth WHERE id = 1").Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoPasswordSet
		}
		return fmt.Errorf("failed to get password: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	return nil
}

// IsPasswordSet returns true if a password has been configured.
func (s *Service) IsPasswordSet() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM auth WHERE id = 1").Scan(&count)
	return err == nil && count > 0
}

// GenerateToken creates a new JWT token.
func (s *Service) GenerateToken() (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "slipstream",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims.
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateAPIKey generates a random API key.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
