// Package restoreapi holds the first-run page's restore: an uploaded
// Backup, opened with its passphrase, then confirmed, after which the Node
// restarts to finish it.
package restoreapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
)

// Restorer is the Node as the restore reaches it.
type Restorer interface {
	// ReceiveRestore keeps an uploaded Backup and answers its id.
	ReceiveRestore(ctx context.Context, body io.Reader) (string, error)
	// UploadRoom is how many bytes an upload may take.
	UploadRoom(ctx context.Context) (int64, error)
	// OpenRestore stages the upload id with the passphrase typed.
	OpenRestore(ctx context.Context, id, passphrase string) (backups.Summary, error)
	// RestoreController commits the staged upload id and restarts.
	RestoreController(ctx context.Context, id string, c backups.Confirm) error
}

var (
	wrongPassphrase = api.NewCode(http.StatusUnprocessableEntity, string(backups.WrongPassphrase), "Wrong passphrase")
	damaged         = api.NewCode(http.StatusUnprocessableEntity, string(backups.Damaged), "Damaged Backup")
	notABackup      = api.NewCode(http.StatusUnprocessableEntity, string(backups.NotABackup), "Not a Backup")
	tooNew          = api.NewCode(http.StatusUnprocessableEntity, string(backups.TooNew), "Backup from a newer version")
	noSpace         = api.NewCode(http.StatusUnprocessableEntity, string(backups.NoSpace), "Not enough space")
	oldControllerOn = api.NewCode(http.StatusUnprocessableEntity, "old_controller_on", "Confirm that the old Controller stays off")
	noUpload        = api.NewCode(http.StatusNotFound, "restore_unknown", "This upload isn't here anymore")
)

var stageCodes = map[backups.StageCode]*api.Code{
	backups.WrongPassphrase: wrongPassphrase,
	backups.Damaged:         damaged,
	backups.NotABackup:      notABackup,
	backups.TooNew:          tooNew,
	backups.NoSpace:         noSpace,
}

type uploaded struct {
	ID string `json:"id" doc:"Names the upload in the open and confirm steps"`
}

type uploadedOutput struct{ Body uploaded }

type openRestoreInput struct {
	ID   string `path:"id" maxLength:"64"`
	Body struct {
		Passphrase string `json:"passphrase" minLength:"1" maxLength:"1024"`
	}
}

type summaryNode struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type summary struct {
	TakenAt    time.Time     `json:"takenAt"`
	Version    string        `json:"version" doc:"The SlipStream version that took the Backup"`
	Controller string        `json:"controller" doc:"The name of the Controller the Backup is of"`
	Nodes      []summaryNode `json:"nodes" doc:"Every Node the Backup lists, the Controller first"`
	NewMachine bool          `json:"newMachine" doc:"The Backup is of another machine: confirm that its Controller stays off"`
}

type summaryOutput struct{ Body summary }

type confirmRestoreInput struct {
	ID   string `path:"id" maxLength:"64"`
	Body struct {
		OldControllerOff bool     `json:"oldControllerOff" doc:"Required for a Backup of another machine"`
		Remove           []string `json:"remove,omitempty" doc:"The Nodes to leave out of the restored library" maxItems:"10"`
	}
}

func Routes(r *api.Router, n Restorer) {
	o := r.Owner("restore", api.Setup)

	api.RegisterUpload(o, api.Setup, api.Op{
		ID:      "uploadRestore",
		Method:  http.MethodPost,
		Path:    "/setup/restore",
		Summary: "Upload a Backup to restore on this Node",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{noSpace, api.LifecycleBusy},
	}, func(ctx context.Context) (int64, error) { return n.UploadRoom(ctx) }, func(ctx context.Context, _ identity.Claim, body io.Reader) (*uploadedOutput, error) {
		id, err := n.ReceiveRestore(ctx, body)
		if err != nil {
			return nil, refusal(err)
		}
		return &uploadedOutput{Body: uploaded{ID: id}}, nil
	})

	api.Register(o, api.Setup, api.Op{
		ID:      "openRestore",
		Method:  http.MethodPost,
		Path:    "/setup/restore/{id}/open",
		Summary: "Open the uploaded Backup with its passphrase and summarize it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{wrongPassphrase, damaged, notABackup, tooNew, noSpace, noUpload, api.AlreadyClaimed},
	}, func(ctx context.Context, _ identity.Claim, in *openRestoreInput) (*summaryOutput, error) {
		s, err := n.OpenRestore(ctx, in.ID, in.Body.Passphrase)
		if err != nil {
			return nil, refusal(err)
		}
		out := summary{TakenAt: s.TakenAt, Version: s.Version, Controller: s.Controller, NewMachine: s.NewMachine, Nodes: []summaryNode{}}
		for _, node := range s.Nodes {
			out.Nodes = append(out.Nodes, summaryNode{ID: node.ID.String(), Name: node.Name})
		}
		return &summaryOutput{Body: out}, nil
	})

	api.Register(o, api.Setup, api.Op{
		ID:      "confirmRestore",
		Method:  http.MethodPost,
		Path:    "/setup/restore/{id}/confirm",
		Summary: "Restore the opened Backup. The Node restarts to finish it, then opens sign-in",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{oldControllerOn, noUpload, noSpace, api.JoinPending, api.LifecycleBusy},
	}, func(ctx context.Context, _ identity.Claim, in *confirmRestoreInput) (*struct{}, error) {
		c := backups.Confirm{OldControllerOff: in.Body.OldControllerOff}
		for _, s := range in.Body.Remove {
			id, err := storage.ParseUUID(s)
			if err != nil {
				return nil, api.Invalid.On("remove", "Each Node to leave out is a Node id from the summary.")
			}
			c.Remove = append(c.Remove, nodes.ID{UUID: id})
		}
		if err := n.RestoreController(ctx, in.ID, c); err != nil {
			return nil, refusal(err)
		}
		return &struct{}{}, nil
	})
}

// ErrNoUpload is what a Restorer answers for an id it doesn't hold.
var ErrNoUpload = errors.New("no upload has that id")

func refusal(err error) error {
	if stage, ok := errors.AsType[*backups.StageError](err); ok {
		return stageCodes[stage.Code].Logged(stage.Err, stage.Text)
	}
	if big, ok := errors.AsType[*api.TooLargeError](err); ok {
		return noSpace.Because(fmt.Sprintf("Restoring needs %s free on this Node's disk, and it has %s free. Free some space, then try again.", backups.Size(big.Sent), backups.Size(big.Max)))
	}
	switch {
	case errors.Is(err, backups.ErrOldControllerOn):
		return oldControllerOn.Because("This Backup is of another machine. Confirm that its Controller stays off.")
	case errors.Is(err, ErrNoUpload):
		return noUpload.Because("Upload the Backup again.")
	case errors.Is(err, lifecycle.ErrBusy):
		return api.LifecycleBusy
	case errors.Is(err, identity.ErrAlreadyClaimed):
		return api.AlreadyClaimed
	case errors.Is(err, identity.ErrJoinPending):
		return api.JoinPending
	}
	return err
}
