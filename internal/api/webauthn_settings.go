package api

import (
	"context"
	"strings"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	settingWebAuthnRPID          = "webauthn_rp_id"
	settingWebAuthnRPOrigins     = "webauthn_rp_origins"
	settingWebAuthnRPDisplayName = "webauthn_rp_display_name"
)

// loadWebAuthnRPID returns the configured RP ID, preferring the DB-stored
// override and falling back to the static config default.
func loadWebAuthnRPID(ctx context.Context, queries *sqlc.Queries, cfg *config.WebAuthnConfig) string {
	if setting, err := queries.GetSetting(ctx, settingWebAuthnRPID); err == nil && setting.Value != "" {
		return setting.Value
	}
	return cfg.RPID
}

// loadWebAuthnRPOrigins returns the configured origin list, preferring the
// DB-stored override and falling back to the static config default.
func loadWebAuthnRPOrigins(ctx context.Context, queries *sqlc.Queries, cfg *config.WebAuthnConfig) []string {
	if setting, err := queries.GetSetting(ctx, settingWebAuthnRPOrigins); err == nil && setting.Value != "" {
		return parseOriginList(setting.Value)
	}
	return cfg.RPOrigins
}

// loadWebAuthnRPDisplayName returns the configured display name, preferring
// the DB-stored override and falling back to the static config default.
func loadWebAuthnRPDisplayName(ctx context.Context, queries *sqlc.Queries, cfg *config.WebAuthnConfig) string {
	if setting, err := queries.GetSetting(ctx, settingWebAuthnRPDisplayName); err == nil && setting.Value != "" {
		return setting.Value
	}
	return cfg.RPDisplayName
}

// parseOriginList splits the newline-delimited origin storage format into
// a clean string slice with empties trimmed.
func parseOriginList(raw string) []string {
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// serializeOriginList joins origins with newlines for DB storage.
func serializeOriginList(origins []string) string {
	cleaned := make([]string, 0, len(origins))
	for _, o := range origins {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "\n")
}
