package downloadclients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

type ClientID = db.ClientID

// External is one external Download Client as the Controller keeps it.
// Its secrets stay sealed in the row.
type External struct {
	ID       ClientID
	Node     role.NodeID
	Kind     clients.Kind
	Name     string
	Address  clients.Address
	Username string
	Label    clients.Label
	// Folder is the download folder as the client sees it, "" for the
	// client's default.
	Folder  string
	Folders clients.Folders
	// Daemon is Deluge's chosen daemon, nil for the only one.
	Daemon                  *clients.Daemon
	CheckEvery, ImportDelay time.Duration
	On                      bool
	// Version is the settings version, bumped on every edit.
	Version int64
}

var (
	ErrNoClient     = errors.New("no Download Client has this id")
	ErrNameTaken    = errors.New("another Download Client has that name")
	ErrAddressTaken = errors.New("another Download Client uses that address")
)

var secretsColumn = secret.NewColumn("downloadclients_client.secrets_sealed")

// ExternalOf reads client id. A deleted client is ErrNoClient.
func ExternalOf(ctx context.Context, r storage.Reader[file], id ClientID) (External, error) {
	row, err := db.New(r).GetClient(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return External{}, ErrNoClient
	}
	if err != nil {
		return External{}, err
	}
	return externalOf(row)
}

func externalOf(row db.DownloadclientsClient) (External, error) {
	address, err := clients.ParseAddress(row.Address, row.Kind)
	if err != nil {
		return External{}, fmt.Errorf("client %s: the stored address: %w", row.ID, err)
	}
	label, err := clients.ParseLabel(row.Label, row.Kind)
	if err != nil {
		return External{}, fmt.Errorf("client %s: the stored label: %w", row.ID, err)
	}
	var pairs []clients.Folder
	if err := json.Unmarshal([]byte(row.ClientFolders), &pairs); err != nil {
		return External{}, fmt.Errorf("client %s: decode the Client folders: %w", row.ID, err)
	}
	folders, err := clients.ParseFolders(pairs)
	if err != nil {
		return External{}, fmt.Errorf("client %s: the stored Client folders: %w", row.ID, err)
	}
	e := External{
		ID: row.ID, Node: row.NodeID, Kind: row.Kind, Name: row.Name, Address: address, Label: label, Folders: folders,
		CheckEvery: CheckEvery.Value(row.CheckEverySeconds), ImportDelay: ImportDelay.Value(row.ImportDelaySeconds),
		On: row.IsOn, Version: row.SettingsVersion,
	}
	if row.Username != nil {
		e.Username = *row.Username
	}
	if row.Folder != nil {
		e.Folder = *row.Folder
	}
	if row.DaemonHost != nil {
		e.Daemon = &clients.Daemon{Host: *row.DaemonHost, Port: int(*row.DaemonPort)}
	}
	return e, nil
}

// Imported is a client S54 (Import engine and import from v1) carries
// over from v1, parsed by the caller.
type Imported struct {
	Node        role.NodeID
	Kind        clients.Kind
	Name        string
	Address     clients.Address
	Username    string
	Secrets     clients.Secrets
	Label       clients.Label
	ImportDelay time.Duration
}

// Import saves c in tx with no Test: on, at settings version 1 and with
// no poll, so its first poll shows whether it works. It fails with
// ErrNameTaken or ErrAddressTaken.
func Import(ctx context.Context, tx *storage.Tx[file], sealer *secret.Sealer, c Imported, now storage.Time) (ClientID, error) {
	q := db.New(tx)
	taken, err := q.ClientTaken(ctx, db.ClientTakenParams{Name: c.Name, AddressKey: c.Address.Key()})
	switch {
	case err != nil:
		return ClientID{}, err
	case taken.NameTaken != 0:
		return ClientID{}, ErrNameTaken
	case taken.AddressTaken != 0:
		return ClientID{}, ErrAddressTaken
	}
	id := ClientID{UUID: storage.NewUUID()}
	sealed, err := seal(sealer, id, c.Secrets)
	if err != nil {
		return ClientID{}, err
	}
	var username *string
	if c.Username != "" {
		username = &c.Username
	}
	delay := int64(c.ImportDelay / time.Second)
	return id, q.InsertClient(ctx, db.InsertClientParams{
		ID: id, NodeID: c.Node, Kind: c.Kind, Name: c.Name, Address: c.Address.String(), AddressKey: c.Address.Key(),
		Username: username, SecretsSealed: sealed, Label: c.Label.String(), ImportDelaySeconds: &delay, CreatedAt: now,
	})
}

// sealedSecrets is how a client's secrets are encoded under the seal.
type sealedSecrets struct {
	Password string `json:"password,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
}

// seal is s sealed for client id's row, unset for a client with none.
func seal(sealer *secret.Sealer, id ClientID, s clients.Secrets) (secret.Sealed, error) {
	if s == (clients.Secrets{}) {
		return secret.Sealed{}, nil
	}
	b, err := json.Marshal(sealedSecrets(s)) //nolint:gosec // G117: sealed below
	if err != nil {
		return secret.Sealed{}, fmt.Errorf("encode the secrets: %w", err)
	}
	return sealer.Seal(b, secretsColumn.Row(id.String())), nil
}
