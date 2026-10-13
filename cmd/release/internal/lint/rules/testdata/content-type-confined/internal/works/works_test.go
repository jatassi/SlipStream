package works

import "testing"

func TestFixture(t *testing.T) {
	if levels["fixture"] != 0 {
		t.Fatal("tv")
	}
}
