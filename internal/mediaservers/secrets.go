package mediaservers

import (
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/secret"
)

var secretsColumn = secret.NewColumn("mediaservers_server.secrets_sealed")

// sealedCredential is how a server's credential is encoded under the seal.
type sealedCredential struct {
	Token  string `json:"token,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
}

// sealCredential is c sealed for server id's row.
func sealCredential(sealer *secret.Sealer, id link.ServerID, c link.Credential) (secret.Sealed, error) {
	b, err := json.Marshal(sealedCredential(c)) //nolint:gosec // G117: sealed below
	if err != nil {
		return secret.Sealed{}, fmt.Errorf("encode the credential: %w", err)
	}
	return sealer.Seal(b, secretsColumn.Row(id.String())), nil
}

// unsealCredential is server id's credential.
func unsealCredential(sealer *secret.Sealer, id link.ServerID, sealed secret.Sealed) (link.Credential, error) {
	b, err := sealer.Open(sealed, secretsColumn.Row(id.String()))
	if err != nil {
		return link.Credential{}, fmt.Errorf("open the credential of Media Server %s: %w", id, err)
	}
	var c sealedCredential
	if err := json.Unmarshal(b, &c); err != nil {
		return link.Credential{}, fmt.Errorf("decode the credential of Media Server %s: %w", id, err)
	}
	return link.Credential(c), nil
}
