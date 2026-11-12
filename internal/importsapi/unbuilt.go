package importsapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

// These operations are the contract S33 (Import pipeline)'s lanes build
// against. Each answers notBuilt until its lane replaces the handler, and
// TestUnbuiltOperationsAnswer501 lists the ones still left.
var (
	notBuilt          = api.NewCode(http.StatusNotImplemented, "import_not_built", "This SlipStream build can't do this yet")
	importMoveSeeding = api.NewCode(http.StatusConflict, "import_move_seeding", "A tracked torrent seeds one of these files. Move files can't take a file while it seeds.")
	importOutside     = api.NewCode(http.StatusUnprocessableEntity, "import_outside", "The folder isn't in one of the Node's library roots or download folders")
)

func unbuilt[I, O any](context.Context, identity.Admin, *I) (*O, error) { return nil, notBuilt }

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
