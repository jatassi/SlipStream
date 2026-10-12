package node

import (
	"os"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestSchemaListsPassTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, librarySchemas())
	storagetest.AuditLocal(t, localSchemas())
}

func TestReadyNodeHasBothFilesBesideRun(t *testing.T) {
	env, _ := testEnv(t)
	startTest(t, env)
	entries, err := os.ReadDir(env.DataDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	for _, want := range []string{"library.db", "local.db", "run"} {
		if !slices.Contains(names, want) {
			t.Errorf("data dir holds %v, want %s among them", names, want)
		}
	}
}
