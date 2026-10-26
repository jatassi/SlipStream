package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

// FilesChoice is what a delete does with the Works' library files.
type FilesChoice string

const (
	DeleteFiles FilesChoice = "delete"
	LeaveFiles  FilesChoice = "leave"
)

type Deleting struct {
	Files   FilesChoice
	Preview bool
}

// DeletePlan is what a delete does to the Works it would remove.
// Downloading counts unfinished Grabs, whose torrents keep downloading and
// seeding in the client.
type DeletePlan struct {
	Works       int
	Files, Kept int
	Bytes       int64
	Downloading int
}

func (l *Library) Delete(context.Context, identity.Admin, []works.WorkID, Deleting) (DeletePlan, []Outcome, error) {
	return DeletePlan{}, nil, ErrNotBuilt
}
