package notifications

import (
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/secret"
)

var secretsColumn = secret.NewColumn("notifications_connection.secrets_sealed")

func (s *Service) seal(id ConnectionID, v providers.Values) (secret.Sealed, error) {
	if len(v) == 0 {
		return secret.Sealed{}, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return secret.Sealed{}, fmt.Errorf("encode the secrets: %w", err)
	}
	return s.cfg.Sealer.Seal(b, secretsColumn.Row(id.String())), nil
}

func (s *Service) open(id ConnectionID, sealed secret.Sealed) (providers.Values, error) {
	if !sealed.Set() {
		return providers.Values{}, nil
	}
	b, err := s.cfg.Sealer.Open(sealed, secretsColumn.Row(id.String()))
	if err != nil {
		return nil, fmt.Errorf("open the secrets of connection %s: %w", id, err)
	}
	var v providers.Values
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("decode the secrets of connection %s: %w", id, err)
	}
	return v, nil
}

