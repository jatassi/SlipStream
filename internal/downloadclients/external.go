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
	// Saved is which secrets the row holds, for "Saved" and Replace.
	Saved SavedSecrets
	// Version is the settings version, bumped on every edit.
	Version int64
	// Poll is nil until the Node's first poll.
	Poll *Poll

	// checkEvery and importDelay are the stored overrides, nil at the
	// default.
	checkEvery, importDelay *int64
}

type SavedSecrets struct{ Password, APIKey bool }

// Poll is the client's last poll as its Node reported it.
type Poll struct {
	State clients.PollState
	Since time.Time
	// Problem is nil while responding.
	Problem *clients.Problem
	// Version is the product's version, "4.1.1", "" before a sign-in.
	Version string
}

// ClientReach is how an Admin reads an external client now: its Poll, or
// Off, or its Node being offline, which no report says.
type ClientReach string

const (
	ReachResponding     ClientReach = "responding"
	ReachNotResponding  ClientReach = "not_responding"
	ReachSignInRefused  ClientReach = "sign_in_refused"
	ReachNeedsAttention ClientReach = "needs_attention"
	ReachOff            ClientReach = "off"
	ReachNodeOffline    ClientReach = "node_offline"
	// ReachUnchecked is a client no poll has reached yet.
	ReachUnchecked ClientReach = "unchecked"
)

// ReachOf derives a client's reach from the record and the link, in that order: off,
// Node offline, then the poll.
func ReachOf(e External, inStep bool) ClientReach {
	var state *clients.PollState
	if e.Poll != nil {
		state = &e.Poll.State
	}
	return reachOf(e.On, inStep, state)
}

// reachOf is ReachOf for a client on or off, whose Node is in step or
// not, at its last poll's state, nil before its first.
func reachOf(on, inStep bool, state *clients.PollState) ClientReach {
	switch {
	case !on:
		return ReachOff
	case !inStep:
		return ReachNodeOffline
	case state == nil:
		return ReachUnchecked
	}
	return ClientReach(*state)
}

var (
	ErrNoClient  = errors.New("no Download Client has this id")
	ErrNameTaken = errors.New("another Download Client has that name")
)

// AddressTakenError is an address another client already uses: Name on
// Node.
type AddressTakenError struct {
	Name string
	Node role.NodeID
}

func (e *AddressTakenError) Error() string { return e.Name + " already uses that address" }

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
	return externalOf(row, nil)
}

// externalOf is row as a client. sealer opens its secrets to say which it
// holds, and is nil for a reader that never shows them.
func externalOf(row db.DownloadclientsClient, sealer *secret.Sealer) (External, error) {
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
		On: row.IsOn, Version: row.SettingsVersion, checkEvery: row.CheckEverySeconds, importDelay: row.ImportDelaySeconds,
	}
	if sealer != nil {
		s, err := unseal(sealer, row.ID, row.SecretsSealed)
		if err != nil {
			return External{}, err
		}
		e.Saved = SavedSecrets{Password: s.Password != "", APIKey: s.APIKey != ""}
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
	if row.PollState != nil {
		if e.Poll, err = pollOf(row); err != nil {
			return External{}, fmt.Errorf("client %s: the stored poll: %w", row.ID, err)
		}
	}
	return e, nil
}

func pollOf(row db.DownloadclientsClient) (*Poll, error) {
	p := &Poll{State: clients.PollState(*row.PollState), Since: row.PollSince.Time()}
	if row.ClientVersion != nil {
		p.Version = *row.ClientVersion
	}
	if row.PollProblem == nil {
		return p, nil
	}
	var s storedProblem
	if err := json.Unmarshal([]byte(*row.PollProblem), &s); err != nil {
		return nil, fmt.Errorf("decode the problem: %w", err)
	}
	p.Problem = &clients.Problem{Cause: s.Cause, Status: s.Status, Folder: s.Folder, Paired: s.Paired, Host: s.Host, Missing: s.Missing}
	if s.Version != "" {
		v, err := clients.ParseVersion(s.Version)
		if err != nil {
			return nil, err
		}
		p.Problem.Version = v
	}
	return p, nil
}

// storedProblem is how a poll's problem is stored.
type storedProblem struct {
	Cause   clients.Cause `json:"cause"`
	Status  int           `json:"status,omitempty"`
	Version string        `json:"version,omitempty"`
	Folder  string        `json:"folder,omitempty"`
	Paired  string        `json:"paired,omitempty"`
	Host    string        `json:"host,omitempty"`
	Missing int           `json:"missing,omitempty"`
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
// ErrNameTaken or *AddressTakenError.
func Import(ctx context.Context, tx *storage.Tx[file], sealer *secret.Sealer, c Imported, now storage.Time) (ClientID, error) {
	e := External{
		ID: ClientID{UUID: storage.NewUUID()}, Node: c.Node, Kind: c.Kind, Name: c.Name, Address: c.Address,
		Username: c.Username, Label: c.Label, On: true, importDelay: new(int64(c.ImportDelay / time.Second)),
	}
	if err := free(ctx, tx, e); err != nil {
		return ClientID{}, err
	}
	p, err := rowOf(sealer, e, c.Secrets)
	if err != nil {
		return ClientID{}, err
	}
	return e.ID, db.New(tx).InsertClient(ctx, db.InsertClientParams{
		ID: e.ID, NodeID: p.NodeID, Kind: p.Kind, Name: p.Name, Address: p.Address, AddressKey: p.AddressKey,
		Username: p.Username, SecretsSealed: p.SecretsSealed, Label: p.Label, Folder: p.Folder, ClientFolders: p.ClientFolders,
		DaemonHost: p.DaemonHost, DaemonPort: p.DaemonPort, CheckEverySeconds: p.CheckEverySeconds,
		ImportDelaySeconds: p.ImportDelaySeconds, IsOn: p.IsOn, CreatedAt: now,
	})
}

// free is nil unless another client than e has e's name or address.
func free(ctx context.Context, r storage.Reader[file], e External) error {
	q := db.New(r)
	taken, err := q.NameTaken(ctx, db.NameTakenParams{Name: e.Name, Except: e.ID})
	switch {
	case err != nil:
		return err
	case taken != 0:
		return ErrNameTaken
	}
	holder, err := q.AddressHolder(ctx, db.AddressHolderParams{AddressKey: e.Address.Key(), Except: e.ID})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	}
	return &AddressTakenError{Name: holder.Name, Node: holder.NodeID}
}

// rowOf is e as UpdateClient stores it, with s sealed.
func rowOf(sealer *secret.Sealer, e External, s clients.Secrets) (db.UpdateClientParams, error) {
	sealed, err := seal(sealer, e.ID, s)
	if err != nil {
		return db.UpdateClientParams{}, err
	}
	folders, err := json.Marshal(e.Folders.Pairs())
	if err != nil {
		return db.UpdateClientParams{}, fmt.Errorf("encode the Client folders: %w", err)
	}
	p := db.UpdateClientParams{
		ID: e.ID, NodeID: e.Node, Kind: e.Kind, Name: e.Name, Address: e.Address.String(), AddressKey: e.Address.Key(),
		Username: text(e.Username), SecretsSealed: sealed, Label: e.Label.String(), Folder: text(e.Folder),
		ClientFolders: string(folders), CheckEverySeconds: e.checkEvery, ImportDelaySeconds: e.importDelay,
		IsOn: e.On, SettingsVersion: e.Version,
	}
	if d := e.Daemon; d != nil {
		p.DaemonHost, p.DaemonPort = &d.Host, new(int64(d.Port))
	}
	return p, nil
}

func text(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

// unseal is client id's secrets, none when its row holds none.
func unseal(sealer *secret.Sealer, id ClientID, sealed secret.Sealed) (clients.Secrets, error) {
	if !sealed.Set() {
		return clients.Secrets{}, nil
	}
	b, err := sealer.Open(sealed, secretsColumn.Row(id.String()))
	if err != nil {
		return clients.Secrets{}, fmt.Errorf("client %s: open the secrets: %w", id, err)
	}
	var s sealedSecrets
	if err := json.Unmarshal(b, &s); err != nil {
		return clients.Secrets{}, fmt.Errorf("client %s: decode the secrets: %w", id, err)
	}
	return clients.Secrets(s), nil
}
