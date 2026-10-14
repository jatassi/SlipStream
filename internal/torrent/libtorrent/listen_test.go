package libtorrent

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

var errRefused = &engine.Error{Category: engine.CategorySystem, Code: 10013, Text: "forbidden by its access permissions"}

func refusing(refused ...uint16) (open func(uint16) (string, error), tried *[]uint16) {
	tried = new([]uint16)
	return func(port uint16) (string, error) {
		*tried = append(*tried, port)
		if slices.Contains(refused, port) {
			return "", fmt.Errorf("%w on 127.0.0.1:%d: %w", engine.ErrNotListening, port, errRefused)
		}
		return fmt.Sprintf("session on %d", port), nil
	}, tried
}

func TestFirstListeningMovesPastRefusedPorts(t *testing.T) {
	open, tried := refusing(50001, 50002)

	s, err := firstListening([]uint16{50001, 50002, 50003, 50004}, open)
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}
	if s != "session on 50003" {
		t.Fatalf("got %q, want %q", s, "session on 50003")
	}
	if want := []uint16{50001, 50002, 50003}; !slices.Equal(*tried, want) {
		t.Fatalf("tried %v, want %v", *tried, want)
	}
}

func TestFirstListeningReportsTheLastRefusal(t *testing.T) {
	open, tried := refusing(50001, 50002)

	_, err := firstListening([]uint16{50001, 50002}, open)
	want := "shim error 3: not listening on 127.0.0.1:50002: system error 10013: forbidden by its access permissions"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if !errors.Is(err, engine.ErrNotListening) || !errors.Is(err, errRefused) {
		t.Fatalf("err = %v, want it to wrap both %v and %v", err, engine.ErrNotListening, errRefused)
	}
	if want := []uint16{50001, 50002}; !slices.Equal(*tried, want) {
		t.Fatalf("tried %v, want %v", *tried, want)
	}
}

func TestFirstListeningStopsOnAnErrorThatIsNotARefusedListen(t *testing.T) {
	var tried []uint16
	bad := &engine.Error{Category: engine.CategoryException, Text: "invalid address"}

	_, err := firstListening([]uint16{50001, 50002}, func(port uint16) (string, error) {
		tried = append(tried, port)
		return "", bad
	})
	if !errors.Is(err, bad) {
		t.Fatalf("err = %v, want %v", err, bad)
	}
	if want := []uint16{50001}; !slices.Equal(tried, want) {
		t.Fatalf("tried %v, want %v", tried, want)
	}
}
