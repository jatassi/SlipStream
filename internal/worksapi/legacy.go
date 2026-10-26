package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

type listInput struct {
	ContentType string `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type listedWork struct {
	Work
	Fields  []api.OneOf[fieldValue] `json:"fields" nullable:"false" doc:"The Work's own fields"`
	Artwork []art                   `json:"artwork" nullable:"false" doc:"The Work's own images, by kind"`
}

type workList struct {
	Works []listedWork `json:"works" nullable:"false" doc:"By sort label, then id"`
}

type workListOutput struct{ Body workList }

type treeOutput struct{ Body tree }

func (h handlers) listWorks(ctx context.Context, by identity.Admin, in *listInput) (*workListOutput, error) {
	var only *works.TypeID
	if in.ContentType != "" {
		t := works.TypeID(in.ContentType)
		only = &t
	}
	list, err := open(h.shelf).List(ctx, by, only)
	if err != nil {
		return nil, answer(err)
	}
	m := minter(h.shelf)
	out := workList{Works: []listedWork{}}
	for _, l := range list {
		out.Works = append(out.Works, listedWork{Work: workOf(l.Summary), Fields: fieldsOf(m, l.Fields), Artwork: artOf(m, l.Artwork)})
	}
	return &workListOutput{Body: out}, nil
}

func (h handlers) refreshWork(ctx context.Context, by identity.Admin, in *workInput) (*treeOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.shelf).Refresh(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.treeOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &treeOutput{Body: out}, nil
}
