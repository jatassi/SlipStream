package filesapi

import (
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestAConfirmOf20000WorksFitsTheDefaultBodyLimit(t *testing.T) {
	var in renameInput
	for range 20_000 {
		in.Body.Works = append(in.Body.Works, confirmedWork{Work: storage.NewUUID().String(), Digest: "9f2c41aa07be13d0"})
	}
	body, err := json.Marshal(in.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) >= 2<<20 {
		t.Fatalf("the confirm body of 20,000 Works is %d bytes, want under 2 MiB", len(body))
	}
	t.Logf("the confirm body of 20,000 Works is %d bytes", len(body))
}
