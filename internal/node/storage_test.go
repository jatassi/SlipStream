package node

import (
	"testing"

	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestSchemaListsPassTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, librarySchemas())
	storagetest.AuditLocal(t, localSchemas())
}
