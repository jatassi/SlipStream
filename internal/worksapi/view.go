package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
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
	out, err := h.view(ctx, by, id)
	if err != nil {
		return nil, err
	}
	return &viewOutput{Body: out}, nil
}

func (h handlers) view(ctx context.Context, by identity.Admin, id works.WorkID) (workView, error) {
	w, err := composer(h.shelf).Work(ctx, by, id)
	if err != nil {
		return workView{}, answer(err)
	}
	return viewOf(minter(h.shelf), w), nil
}

func viewOf(m artwork.Minter, w library.Work) workView {
	out := workView{Tree: treeOf(m, w.Tree, w.Timings), Links: make([]webLink, len(w.Links)), Profile: workProfile{ID: w.Profile.ID.String(), Name: w.Profile.Name}}
	for i, l := range w.Links {
		out.Links[i] = webLink{Key: l.Key, Label: l.Label, URL: l.URL, Kind: l.Kind.String()}
	}
	if r := w.Root; r != nil {
		out.Root = api.Some(workRoot{ID: r.ID.String(), NodeID: r.Node.String(), Path: r.Path})
	}
	return out
}
