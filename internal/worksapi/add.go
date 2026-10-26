package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type addInput struct {
	Body newWork
}

type newWork struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string `json:"key" minLength:"1" maxLength:"200" doc:"The Work's key at its metadata source"`
	Monitor     *bool  `json:"monitor,omitempty" doc:"Monitor a new Work's Items; left out, the type's add default. Adding a Work already in the library changes nothing"`
	ProfileID   string `json:"profileId,omitempty" format:"uuid" doc:"The Profile of the Content Type a new Work wants from; left out, the type's default"`
	RootID      string `json:"rootId,omitempty" format:"uuid" doc:"The root of the Content Type a new Work lives in; left out, the type's default, or none while the type has no root"`
	Remember    bool   `json:"remember,omitempty" doc:"Make each choice given here its type's default for the next add. Adding a Work already in the library changes no default"`
}

type added struct {
	Created bool     `json:"created" doc:"False when the Work was already in the library"`
	Work    workView `json:"work"`
}

type addedOutput struct{ Body added }

func (h handlers) addWork(ctx context.Context, by identity.Admin, in *addInput) (*addedOutput, error) {
	key, err := works.ParseKey(in.Body.Key)
	if err != nil {
		return nil, api.Invalid
	}
	var profile *targets.ProfileID
	if in.Body.ProfileID != "" {
		u, err := storage.ParseUUID(in.Body.ProfileID)
		if err != nil {
			return nil, profileNotFound
		}
		profile = &targets.ProfileID{UUID: u}
	}
	var root *roots.ID
	if in.Body.RootID != "" {
		u, err := storage.ParseUUID(in.Body.RootID)
		if err != nil {
			return nil, rootNotFound
		}
		root = &roots.ID{UUID: u}
	}
	t, created, err := open(h.shelf).Add(ctx, by, works.TypeID(in.Body.ContentType), key, works.Adding{
		Monitor: in.Body.Monitor != nil && *in.Body.Monitor,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, root)
		},
	})
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.view(ctx, by, t.ID)
	if err != nil {
		return nil, err
	}
	return &addedOutput{Body: added{Created: created, Work: out}}, nil
}
