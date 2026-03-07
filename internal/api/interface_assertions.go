package api

import (
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
)

// Phase 2 eliminations: these concrete types directly satisfy consumer interfaces,
// replacing the adapter types that were previously needed.
var (
	_ arrimport.RootFolderService = (*rootfolder.Service)(nil)
	_ arrimport.QualityService    = (*quality.Service)(nil)
	_ slots.RootFolderProvider    = (*rootfolder.Service)(nil)
)
