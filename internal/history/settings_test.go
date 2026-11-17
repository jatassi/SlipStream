package history_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/history"
)

func TestKeepHistoryForIsAValidSetting(t *testing.T) {
	if err := history.KeepFor.Check(); err != nil {
		t.Fatal(err)
	}
}
