// Package nodeapi is the Node API as the world speaks it: hand-written
// requests and responses, the way a script calls the API.
package nodeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Admin struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PIN         string `json:"pin"`
}

// ClaimController creates the first Admin through the Setup code and
// returns its API key. It sends one request and never retries.
func ClaimController(ctx context.Context, baseURL, setupCode string, admin Admin) (string, error) {
	body, err := json.Marshal(admin)
	if err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/setup/controller", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Setup-Code", setupCode)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	reply, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	if res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("claim the Controller: %d %s", res.StatusCode, bytes.TrimSpace(reply))
	}
	var claimed struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(reply, &claimed); err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	return claimed.APIKey, nil
}
