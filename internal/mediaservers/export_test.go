package mediaservers

import (
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/secret"
)

// SealCredential seals c for server id's row, as Save stores it.
func SealCredential(sealer *secret.Sealer, id link.ServerID, c link.Credential) (secret.Sealed, error) {
	return sealCredential(sealer, id, c)
}
