package importsapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// notBuilt answers what S33 (Import pipeline)'s later lanes build, and
// TestUnbuiltOperationsAnswer501 lists what still answers it.
var (
	notBuilt          = api.NewCode(http.StatusNotImplemented, "import_not_built", "This SlipStream build can't do this yet")
	importMoveSeeding = api.NewCode(http.StatusConflict, "import_move_seeding", "A tracked torrent seeds one of these files. Move files can't take a file while it seeds.")
	importOutside     = api.NewCode(http.StatusUnprocessableEntity, "import_outside", "The folder isn't in one of the Node's library roots or download folders")
)

type importFolderInput struct {
	Body struct {
		NodeID string `json:"nodeId" format:"uuid"`
		Folder string `json:"folder" minLength:"1" doc:"Absolute on the Node, inside one of its library roots or download folders"`
	}
}

type importsList struct {
	Imports []importEntry `json:"imports" nullable:"false"`
}

type importsOutput struct {
	Status int
	Body   importsList
}

type fileAction struct{}

func (fileAction) Variants() (discriminator string, variants []any) {
	return "action", []any{confirmFile{}, retryFile{}, declineFile{}, notThisReleaseFile{}, forceFile{}, matchFile{}}
}

type confirmFile struct {
	Action string `json:"action" enum:"confirm" doc:"Import the file with its current reading"`
}

type retryFile struct {
	Action string `json:"action" enum:"retry" doc:"Try the file again"`
}

type declineFile struct {
	Action string `json:"action" enum:"decline" doc:"Don't import the file"`
}

type notThisReleaseFile struct {
	Action string `json:"action" enum:"notThisRelease" doc:"Put the download's Release on the Blocklist and search again; a Grab's Import only"`
}

type forceFile struct {
	Action string  `json:"action" enum:"force" doc:"Put the file into a Target even though it isn't an upgrade there"`
	Cell   cellRef `json:"cell"`
}

type matchFile struct {
	Action      string   `json:"action" enum:"match" doc:"Change what the file is"`
	WorkID      string   `json:"workId" format:"uuid"`
	ItemIDs     []string `json:"itemIds" format:"uuid" minItems:"1" nullable:"false" doc:"The Items of the Work the file holds, in any order"`
	Edition     *string  `json:"edition" doc:"The key of the cut it is; null for the one its name carries"`
	ReleaseName string   `json:"releaseName" minLength:"1" doc:"The name SlipStream reads its quality and cut from"`
}

type fileActionInput struct {
	ID       string `path:"id" format:"uuid"`
	Position int    `path:"position" minimum:"0"`
	Body     api.OneOf[fileAction]
}

type moveFilesInput struct {
	Body struct {
		ImportIDs []string `json:"importIds" format:"uuid" minItems:"1" nullable:"false"`
		On        bool     `json:"on"`
	}
}

type extensionImpactInput struct {
	Extensions []string `query:"extensions" minItems:"1" maxItems:"100" doc:"The video file extensions to judge, such as mkv, without the dot"`
}

type extensionImpact struct {
	Files   int `json:"files" minimum:"0" doc:"The library files that stop counting as video"`
	Pending int `json:"pending" minimum:"0" doc:"The files of unfinished Imports that stop counting as video"`
}

type extensionImpactOutput struct{ Body extensionImpact }

func (h handlers) importFolder(ctx context.Context, by identity.Admin, in *importFolderInput) (*importsOutput, error) {
	node, err := storage.ParseUUID(in.Body.NodeID)
	if err != nil {
		return nil, err
	}
	opened, err := open(h.pending).ImportFolder(ctx, by, role.NodeID{UUID: node}, in.Body.Folder)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.entries(ctx, by, opened.IDs)
	if err != nil {
		return nil, err
	}
	out.Status = http.StatusOK
	if opened.Created {
		out.Status = http.StatusCreated
	}
	return out, nil
}

// entries is the Imports ids as the wire shows them, each with its
// sentences.
func (h handlers) entries(ctx context.Context, by identity.Admin, ids []imports.ImportID) (*importsOutput, error) {
	out := &importsOutput{Body: importsList{Imports: make([]importEntry, len(ids))}}
	for i, id := range ids {
		e, err := h.get(ctx, by, &importIDInput{ID: id.String()})
		if err != nil {
			return nil, err
		}
		out.Body.Imports[i] = e.Body
	}
	return out, nil
}

func (h handlers) actOnFile(ctx context.Context, by identity.Admin, in *fileActionInput) (*entryOutput, error) {
	id, err := importID(in.ID)
	if err != nil {
		return nil, err
	}
	a, err := actionOf(in.Body.Value())
	if err != nil {
		return nil, err
	}
	e, err := open(h.pending).Act(ctx, by, imports.FileRef{Import: id, Position: in.Position}, a)
	if err != nil {
		return nil, answer(err)
	}
	return h.get(ctx, by, &importIDInput{ID: e.ID.String()})
}

func actionOf(v any) (imports.Action, error) {
	switch a := v.(type) {
	case confirmFile:
		return imports.ConfirmFile{}, nil
	case retryFile:
		return imports.RetryFile{}, nil
	case declineFile:
		return imports.DeclineFile{}, nil
	case forceFile:
		item, err := storage.ParseUUID(a.Cell.ItemID)
		if err != nil {
			return nil, importItems
		}
		target, err := storage.ParseUUID(a.Cell.TargetID)
		if err != nil {
			return nil, importItems
		}
		return imports.ForceFile{Cell: files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.Edition(a.Cell.Edition), Target: targets.TargetID{UUID: target}}}, nil
	case matchFile:
		work, err := storage.ParseUUID(a.WorkID)
		if err != nil {
			return nil, importItems
		}
		m := imports.MatchFile{Work: works.WorkID{UUID: work}, ReleaseName: a.ReleaseName, Items: make([]works.ItemID, len(a.ItemIDs))}
		for i, s := range a.ItemIDs {
			item, err := storage.ParseUUID(s)
			if err != nil {
				return nil, importItems
			}
			m.Items[i] = works.ItemID{UUID: item}
		}
		if a.Edition != nil {
			m.Edition = new(release.CutKey(*a.Edition))
		}
		return m, nil
	}
	return nil, notBuilt
}

func (h handlers) confirm(ctx context.Context, by identity.Admin, in *importIDsInput) (*importsOutput, error) {
	ids, err := importIDs(in.Body.ImportIDs)
	if err != nil {
		return nil, err
	}
	confirmed, err := open(h.pending).ConfirmAll(ctx, by, ids)
	if err != nil {
		return nil, answer(err)
	}
	for i, e := range confirmed {
		ids[i] = e.ID
	}
	return h.entries(ctx, by, ids)
}

func (h handlers) moveFiles(ctx context.Context, by identity.Admin, in *moveFilesInput) (*struct{}, error) {
	ids, err := importIDs(in.Body.ImportIDs)
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.pending).SetMoveFiles(ctx, by, ids, in.Body.On))
}
