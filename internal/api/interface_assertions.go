package api

import (
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/portal/provisioner"
	"github.com/slipstream/slipstream/internal/portal/requests"
)

// Phase 2 eliminations: these concrete types directly satisfy consumer interfaces,
// replacing the adapter types that were previously needed.
var (
	_ arrimport.RootFolderService = (*rootfolder.Service)(nil)
	_ arrimport.QualityService    = (*quality.Service)(nil)
	_ arrimport.MetadataRefresher = (*librarymanager.Service)(nil)
	_ slots.RootFolderProvider    = (*rootfolder.Service)(nil)
	_ requests.SeriesLookup       = (*tv.Service)(nil)
	_ requests.MediaProvisioner   = (*provisioner.Service)(nil)
)
