// Package record owns the transfers library set: the Transfer records,
// each pair's recent speed and route, and each Node's settings.
package ledger

import (
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger/internal/db"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the transfers library set. It shares its name with the
// transfers local set, as the role package's two sets do.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("transfers", schemaFiles) }

func New(r storage.Reader[file]) *Queries { return db.New(r) }

type (
	Queries              = db.Queries
	Transfer             = db.TransfersTransfer
	File                 = db.TransfersFile
	Pair                 = db.TransfersPair
	Node                 = db.TransfersNode
	InsertTransferParams = db.InsertTransferParams
	InsertFileParams     = db.InsertFileParams
	TransferOfJobParams  = db.TransferOfJobParams
	SetVerifiedParams    = db.SetVerifiedParams
	SetPlacedParams      = db.SetPlacedParams
	SetFailureParams     = db.SetFailureParams
	PutPairParams        = db.PutPairParams
	SetNodeParams        = db.SetNodeSettingsParams
)
