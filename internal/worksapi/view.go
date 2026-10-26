package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

type workView struct {
	Tree
	Links   []webLink              `json:"links" nullable:"false" doc:"Pages about the Work on other sites, in its Content Type's order"`
	Profile workProfile            `json:"profile" doc:"The Profile the Work wants from"`
	Root    api.Nullable[workRoot] `json:"root" doc:"The root the Work lives in; null while it has none"`
}

type webLink struct {
	Key   string `json:"key" doc:"The link's key, such as imdb"`
	Label string `json:"label" doc:"The site's name as a person reads it"`
	URL   string `json:"url" format:"uri"`
	Kind  string `json:"kind" enum:"page,trailer"`
}

type workProfile struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type workRoot struct {
	ID     string `json:"id" format:"uuid"`
	NodeID string `json:"nodeId" format:"uuid" doc:"The Storage Node whose disk holds it"`
	Path   string `json:"path" doc:"The folder's path on that Node, in that Node's format"`
}

type viewOutput struct{ Body workView }

func (h handlers) getWork(ctx context.Context, by identity.Admin, in *workInput) (*viewOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.shelf).Get(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.viewOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &viewOutput{Body: out}, nil
}

// viewOf is t's view with its Profile, and no links or root until the
// Library's Work read fills them.
func (h handlers) viewOf(ctx context.Context, by identity.Admin, t works.Tree) (workView, error) {
	tree, err := h.treeOf(ctx, by, t)
	if err != nil {
		return workView{}, err
	}
	g, err := goals(h.shelf).Goals(ctx, by, t.ID)
	if err != nil {
		return workView{}, err
	}
	return workView{Tree: tree, Links: []webLink{}, Profile: workProfile{ID: g.Profile.ID.String(), Name: g.Profile.Name}}, nil
}
