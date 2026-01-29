package plex

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPINExpired  = errors.New("PIN has expired")
	ErrPINNotReady = errors.New("PIN not yet authorized")
)

// OAuthFlow manages the PIN-based OAuth flow
type OAuthFlow struct {
	client *Client
}

// NewOAuthFlow creates a new OAuth flow manager
func NewOAuthFlow(client *Client) *OAuthFlow {
	return &OAuthFlow{client: client}
}

// OAuthStartResult contains the internal data from starting OAuth flow
type OAuthStartResult struct {
	PinID     int
	PinCode   string
	AuthURL   string
	ExpiresAt time.Time
}

// OAuthCheckResult contains the result of checking OAuth status
type OAuthCheckResult struct {
	AuthToken string
	Complete  bool
}

// StartAuth initiates the OAuth flow by creating a PIN
func (o *OAuthFlow) StartAuth(ctx context.Context) (*OAuthStartResult, error) {
	pin, err := o.client.CreatePIN(ctx)
	if err != nil {
		return nil, err
	}

	authURL := o.client.GetAuthURL(pin.Code)

	return &OAuthStartResult{
		PinID:     pin.ID,
		PinCode:   pin.Code,
		AuthURL:   authURL,
		ExpiresAt: pin.ExpiresAt,
	}, nil
}

// CheckAuth checks if the user has completed authentication
func (o *OAuthFlow) CheckAuth(ctx context.Context, pinID int) (*OAuthCheckResult, error) {
	status, err := o.client.CheckPIN(ctx, pinID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(status.ExpiresAt) {
		return nil, ErrPINExpired
	}

	if status.AuthToken == "" {
		return &OAuthCheckResult{
			Complete: false,
		}, nil
	}

	return &OAuthCheckResult{
		AuthToken: status.AuthToken,
		Complete:  true,
	}, nil
}

// WaitForAuth polls the PIN status until the user authenticates or the PIN expires
func (o *OAuthFlow) WaitForAuth(ctx context.Context, pinID int, pollInterval time.Duration) (string, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			result, err := o.CheckAuth(ctx, pinID)
			if err != nil {
				if errors.Is(err, ErrPINExpired) {
					return "", err
				}
				continue
			}

			if result.Complete {
				return result.AuthToken, nil
			}
		}
	}
}
