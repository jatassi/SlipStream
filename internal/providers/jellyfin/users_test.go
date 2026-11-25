package jellyfin_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

func TestTaken(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		err  error
		want bool
	}{
		{"the duplicate name", &jellyfin.Error{Status: 400, Detail: "A user with the name 'Dana' already exists."}, true},
		{"wrapped", fmt.Errorf("jellyfin POST /Users/New: %w", &jellyfin.Error{Status: 400, Detail: "A user with the name 'Dana' already exists."}), true},
		{"another 400", &jellyfin.Error{Status: 400, Detail: "Usernames can contain unicode symbols, numbers (0-9), dashes (-), underscores (_), apostrophes ('), and periods (.)"}, false},
		{"a 500 with the text", &jellyfin.Error{Status: 500, Detail: "A user with the name 'Dana' already exists."}, false},
		{"no response", errors.New("dial tcp: connection refused"), false},
	} {
		if got := jellyfin.Taken(c.err); got != c.want {
			t.Errorf("%s: Taken is %v, want %v", c.name, got, c.want)
		}
	}
}
