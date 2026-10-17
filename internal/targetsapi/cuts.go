package targetsapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

var (
	cutNotFound     = api.NewCode(http.StatusUnprocessableEntity, "cut_not_found", "No cut has that key")
	noSuchCut       = api.NewCode(http.StatusNotFound, "cut_not_found", "No cut has that key")
	cutNameTaken    = api.NewCode(http.StatusConflict, "cut_name_taken", "A cut already has that name, alias or key")
	cutInUse        = api.NewCode(http.StatusConflict, "cut_in_use", "Edition lists or files hold the cut")
	standardIsFixed = api.NewCode(http.StatusConflict, "standard_is_fixed", "Standard can't be renamed or removed")
	cutNotCustom    = api.NewCode(http.StatusConflict, "cut_not_custom", "Only a custom cut can be changed or removed")
)

var origins = map[targets.Origin]string{targets.BuiltIn: "builtin", targets.FromType: "type", targets.Custom: "custom"}

type cut struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases" nullable:"false" doc:"The phrases that name it in a title or file name, a custom cut's name first; Standard is read where none is named"`
	Origin      string   `json:"origin" enum:"builtin,type,custom" doc:"Built into SlipStream, added by a Content Type, or added by an Admin"`
	ContentType *string  `json:"contentType,omitempty" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"The Content Type that adds it, for origin type"`
}

func cutOf(c targets.ListedCut) cut {
	out := cut{Key: string(c.Key), Name: c.Name, Aliases: append([]string{}, c.Aliases...), Origin: origins[c.Origin]}
	if c.Origin == targets.FromType {
		out.ContentType = new(string(c.Type))
	}
	return out
}

type cutList struct {
	Cuts []cut `json:"cuts" nullable:"false" doc:"The built-in cuts, Standard first, then each Content Type's by type, then the custom ones by name"`
}

type cutListOutput struct{ Body cutList }

func (h handlers) listCuts(ctx context.Context, by identity.Admin, _ *struct{}) (*cutListOutput, error) {
	known, err := open(h.goals).ListCuts(ctx, by)
	if err != nil {
		return nil, err
	}
	out := cutList{Cuts: []cut{}}
	for _, c := range known.All() {
		out.Cuts = append(out.Cuts, cutOf(c))
	}
	return &cutListOutput{Body: out}, nil
}

type newCut struct {
	Name    string   `json:"name" minLength:"1" maxLength:"100" doc:"Its key is the name's words in lower case joined by hyphens, fixed once made"`
	Aliases []string `json:"aliases,omitempty" nullable:"false" maxItems:"20" doc:"Other phrases that name it in a title; its name always does"`
}

func (n newCut) spec() targets.CutSpec { return targets.CutSpec{Name: n.Name, Aliases: n.Aliases} }

type createCutInput struct{ Body newCut }

type cutOutput struct{ Body cut }

// cutRefused answers err from a cut's own operation, which names the cut
// by its path.
func cutRefused(err error) error {
	switch {
	case errors.Is(err, targets.ErrNoCut):
		return noSuchCut
	case errors.Is(err, targets.ErrCutNameTaken):
		return cutNameTaken
	case errors.Is(err, targets.ErrBadCutName):
		return api.Invalid.On("name", err.Error())
	case errors.Is(err, targets.ErrStandardIsFixed):
		return standardIsFixed
	case errors.Is(err, targets.ErrCutNotCustom):
		return cutNotCustom
	}
	if inUse, ok := errors.AsType[*targets.InUseError](err); ok {
		return cutInUse.Because(inUse.Error())
	}
	return changed(err)
}

func (h handlers) createCut(ctx context.Context, by identity.Admin, in *createCutInput) (*cutOutput, error) {
	c, err := open(h.goals).CreateCut(ctx, by, in.Body.spec())
	if err != nil {
		return nil, cutRefused(err)
	}
	return &cutOutput{Body: cutOf(c)}, nil
}

type cutInput struct {
	CutKey string `path:"cutKey"`
}

type updateCutInput struct {
	CutKey string `path:"cutKey"`
	Changing
	Body newCut
}

type cutChange struct {
	Impact impact `json:"impact"`
	Cut    cut    `json:"cut"`
}

type cutChangeOutput struct{ Body cutChange }

func (h handlers) updateCut(ctx context.Context, by identity.Admin, in *updateCutInput) (*cutChangeOutput, error) {
	key := release.CutKey(in.CutKey)
	i, err := shelf(h.goals).Change(ctx, by, targets.UpdateCut{Key: key, Spec: in.Body.spec()}, in.mode())
	if err != nil {
		return nil, cutRefused(err)
	}
	known, err := open(h.goals).ListCuts(ctx, by)
	if err != nil {
		return nil, err
	}
	c, ok := known.Get(key)
	if !ok {
		return nil, noSuchCut
	}
	return &cutChangeOutput{Body: cutChange{Impact: impactOf(i), Cut: cutOf(c)}}, nil
}

func (h handlers) deleteCut(ctx context.Context, by identity.Admin, in *cutInput) (*struct{}, error) {
	if err := open(h.goals).DeleteCut(ctx, by, release.CutKey(in.CutKey), files.HeldOf); err != nil {
		return nil, cutRefused(err)
	}
	return nil, nil
}
