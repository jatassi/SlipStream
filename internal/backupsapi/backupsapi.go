package backupsapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

type Keeper interface {
	// Backups is the Backups service, false unless the Node is a
	// Controller.
	Backups() (*backups.Backups, bool)
}

func open(k Keeper) *backups.Backups {
	b, ok := k.Backups()
	if !ok {
		panic("backupsapi: an operation ran on a Node that is no Controller")
	}
	return b
}

var (
	destinationNotFound = api.NewCode(http.StatusNotFound, "destination_not_found", "No Backup destination has that id")
	destinationExists   = api.NewCode(http.StatusConflict, "destination_exists", "That folder is already a Backup destination")
	nodeNotFound        = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	nodeOffline         = api.NewCode(http.StatusServiceUnavailable, "node_offline", "The Node is offline")
	destinationUnusable = api.NewCarrying[refusal](http.StatusUnprocessableEntity, "destination_unusable", "The folder can't be a Backup destination", "refusal")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{backups.ErrNoDestination, destinationNotFound},
	{backups.ErrDestinationExists, destinationExists},
	{roots.ErrNoNode, nodeNotFound},
	{roots.ErrNodeAway, nodeOffline},
}

func answer(err error) error {
	if unusable, ok := errors.AsType[*backups.UnusableError](err); ok {
		return destinationUnusable.With(refusal{Reason: string(unusable.Refusal.Reason)})
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type refusal struct {
	Reason string `json:"reason" enum:"not_absolute,not_found,not_a_folder,unwritable,inside_root,contains_root,inside_data,contains_data,inside_downloads,contains_downloads,unresponsive,network_share,outside,node_offline" doc:"Why the folder can't hold Backups"`
}

type destination struct {
	ID       string               `json:"id" pattern:"^[0-9a-f]{32}$"`
	NodeID   string               `json:"nodeId" format:"uuid"`
	Folder   string               `json:"folder" doc:"As the Admin typed it"`
	AddedAt  time.Time            `json:"addedAt"`
	Removing bool                 `json:"removing" doc:"Removed, while a copy already running into its folder finishes"`
	Copies   copies               `json:"copies" doc:"Its copies of the listed Backups, by state"`
	Newest   api.Nullable[newest] `json:"newest" doc:"Its newest copy, null before one is copied"`
}

type copies struct {
	Waiting int `json:"waiting" doc:"Waiting for a copy to start"`
	Copying int `json:"copying"`
	Copied  int `json:"copied"`
	Failed  int `json:"failed"`
}

type newest struct {
	BackupID string    `json:"backupId" pattern:"^[0-9a-f]{32}$"`
	Name     string    `json:"name" doc:"The Backup's file name"`
	CopiedAt time.Time `json:"copiedAt"`
	Check    check     `json:"check" doc:"Its last check"`
}

type check struct {
	Result string     `json:"result" enum:"none,asked,ok,mismatch,unreadable,no_local_file" doc:"none: not checked yet; asked: waiting for its Node; no_local_file: this machine has no file of the Backup to check it against"`
	At     *time.Time `json:"at" doc:"When the result came in; null while none has"`
	Cause  string     `json:"cause,omitempty" doc:"Why the check failed, as a sentence"`
}

type destinationList struct {
	Destinations []destination `json:"destinations" nullable:"false" doc:"In the order they were added"`
}

type destinationListOutput struct{ Body destinationList }

type newDestination struct {
	NodeID string `json:"nodeId" format:"uuid"`
	Folder string `json:"folder" minLength:"1" maxLength:"4096" doc:"An absolute folder in the Node's path style"`
}

type addDestinationInput struct{ Body newDestination }

type added struct {
	Destination destination        `json:"destination"`
	Room        api.Nullable[room] `json:"room" doc:"The room the folder's volume lacks to hold every listed Backup; null when it has enough"`
}

type room struct {
	Need int64 `json:"need" doc:"The bytes of every listed Backup"`
	Free int64 `json:"free" doc:"The bytes SlipStream may write on the folder's volume"`
}

type addedOutput struct{ Body added }

type destinationInput struct {
	ID string `path:"destinationId" pattern:"^[0-9a-f]{32}$"`
}

func Routes(r *api.Router, k Keeper) {
	o := r.Owner("backups", api.Admin)
	h := handlers{k}

	api.Register(o, api.Admin, api.Op{
		ID:      "listBackupDestinations",
		Method:  http.MethodGet,
		Path:    "/backup-destinations",
		Summary: "List the Backup destinations and their copies",
		Status:  http.StatusOK,
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID:      "addBackupDestination",
		Method:  http.MethodPost,
		Path:    "/backup-destinations",
		Summary: "Make a folder on a Node a Backup destination, and copy every Backup there",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{nodeNotFound, nodeOffline, destinationExists, destinationUnusable.Code},
	}, h.add)

	api.Register(o, api.Admin, api.Op{
		ID:      "removeBackupDestination",
		Method:  http.MethodDelete,
		Path:    "/backup-destinations/{destinationId}",
		Summary: "Stop copying Backups to a destination, leaving its copies in its folder",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{destinationNotFound},
	}, h.remove)
}

type handlers struct{ k Keeper }

func (h handlers) list(context.Context, identity.Admin, *struct{}) (*destinationListOutput, error) {
	l := open(h.k).Ledger()
	out := destinationList{Destinations: []destination{}}
	for _, d := range l.Destinations {
		out.Destinations = append(out.Destinations, destinationOf(l, d))
	}
	return &destinationListOutput{Body: out}, nil
}

func (h handlers) add(ctx context.Context, by identity.Admin, in *addDestinationInput) (*addedOutput, error) {
	u, err := storage.ParseUUID(in.Body.NodeID)
	if err != nil {
		return nil, nodeNotFound
	}
	b := open(h.k)
	a, err := b.Add(ctx, by, role.NodeID{UUID: u}, in.Body.Folder)
	if err != nil {
		return nil, answer(err)
	}
	out := added{Destination: destinationOf(b.Ledger(), a.Destination)}
	if a.Need > 0 {
		out.Room = api.Some(room{Need: a.Need, Free: a.Free})
	}
	return &addedOutput{Body: out}, nil
}

func (h handlers) remove(ctx context.Context, _ identity.Admin, in *destinationInput) (*struct{}, error) {
	id, err := backups.ParseDestinationID(in.ID)
	if err != nil {
		return nil, destinationNotFound
	}
	return nil, answer(open(h.k).Remove(ctx, id))
}

func destinationOf(l backups.Ledger, d backups.Destination) destination {
	out := destination{ID: d.ID.String(), NodeID: d.Node.String(), Folder: d.Folder, AddedAt: d.AddedAt, Removing: !d.RemovedAt.IsZero()}
	found := false
	for _, e := range l.Backups {
		c, ok := e.Copy(d.ID)
		if !ok || e.State != backups.Verified {
			continue
		}
		switch c.State {
		case backups.CopyWaiting:
			out.Copies.Waiting++
		case backups.Copying:
			out.Copies.Copying++
		case backups.Copied:
			out.Copies.Copied++
		case backups.CopyFailed:
			out.Copies.Failed++
		}
		if c.State == backups.Copied && !found {
			found = true
			text, _ := e.ID.MarshalText()
			out.Newest = api.Some(newest{BackupID: string(text), Name: e.Name.String(), CopiedAt: c.CopiedAt, Check: checkOf(c.Check)})
		}
	}
	return out
}

func checkOf(c backups.Check) check {
	out := check{Result: string(c.Result), Cause: c.Cause}
	if out.Result == "" {
		out.Result = "none"
	}
	if !c.At.IsZero() {
		out.At = &c.At
	}
	return out
}
