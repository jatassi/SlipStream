package transfers

import (
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
)

// LibrarySchema is the Controller's record of every Transfer.
func LibrarySchema() storage.Schema[storage.Library] { return ledger.Schema() }
