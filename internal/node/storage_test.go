package node

import (
	"testing"

	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestSchemaListsPassTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, LibrarySchemas(noTypes))
	storagetest.AuditLocal(t, localSchemas())
}
