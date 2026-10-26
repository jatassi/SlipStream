package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
)

type deleteInput struct {
	Body deleting
}

type deleting struct {
	WorkIDs []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
	Files   string   `json:"files" enum:"delete,leave" doc:"Delete the Works' library files from disk, or leave them where they are"`
	Preview bool     `json:"preview,omitempty" doc:"Return the plan and the outcomes it predicts, and delete nothing"`
}

type deletePlan struct {
	Works       int   `json:"works" minimum:"0" doc:"The Works it removes"`
	Files       int   `json:"files" minimum:"0" doc:"Their library files, Kept files included"`
	Kept        int   `json:"kept" minimum:"0" doc:"Their Kept files"`
	Bytes       int64 `json:"bytes" minimum:"0" doc:"The size of their library files"`
	Downloading int   `json:"downloading" minimum:"0" doc:"Their unfinished downloads, whose torrents keep downloading and seeding in the client"`
}

type deleted struct {
	Plan     deletePlan           `json:"plan"`
	Outcomes []api.OneOf[outcome] `json:"outcomes" nullable:"false" doc:"One per Work, in request order"`
}

type deletedOutput struct{ Body deleted }

func (h handlers) deleteWorks(ctx context.Context, by identity.Admin, in *deleteInput) (*deletedOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	d := library.Deleting{Files: library.FilesChoice(in.Body.Files), Preview: in.Body.Preview}
	plan, os, err := composer(h.shelf).Delete(ctx, by, ws, d)
	if err != nil {
		return nil, answer(err)
	}
	out, err := outcomesOf(os)
	if err != nil {
		return nil, err
	}
	return &deletedOutput{Body: deleted{Plan: deletePlan(plan), Outcomes: out}}, nil
}
