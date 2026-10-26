package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

func (h handlers) getWork(ctx context.Context, by identity.Admin, in *workInput) (*treeOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.shelf).Get(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.treeOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &treeOutput{Body: out}, nil
}
