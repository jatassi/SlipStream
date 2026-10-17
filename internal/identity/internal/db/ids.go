package db

import "github.com/jatassi/slipstream/internal/storage"

type AccountID struct{ storage.UUID }

type SessionID struct{ storage.UUID }

type PasskeyID struct{ storage.UUID }

type LinkID struct{ storage.UUID }

// CredentialID is a WebAuthn credential's id, as the authenticator chose it.
type CredentialID []byte

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// KeyPrompt is whether the passkey prompt may show: "ask" or "never".
type KeyPrompt string
