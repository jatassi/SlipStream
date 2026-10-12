package world

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const worldLogFile = "world.log"

func openWorldLog(runDir string) (*slog.Logger, *os.File, error) {
	f, err := os.OpenFile(filepath.Join(runDir, worldLogFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gosec // world.log is evidence anyone on the machine may read
	if err != nil {
		return nil, nil, fmt.Errorf("open world.log: %w", err)
	}
	return slog.New(slog.NewJSONHandler(f, nil)), f, nil
}

// Spawn is a Node process the world started, as world.log records it the
// moment it starts.
type Spawn struct {
	Node    string `json:"node"`
	PID     int    `json:"pid"`
	DataDir string `json:"data_dir"`
}

// WorldLog is what world.log says about a run's processes, enough to stop
// them by hand when the supervisor is gone.
type WorldLog struct {
	Supervisor Supervisor
	Spawns     []Spawn
}

const (
	launchRecord = "launch"
	spawnRecord  = "node spawned"
)

// ReadWorldLog reads the launch and spawn records of the run in runDir. It
// skips lines that are not JSON, such as a crashed supervisor's stderr.
func ReadWorldLog(runDir string) (WorldLog, error) {
	path := filepath.Join(runDir, worldLogFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return WorldLog{}, fmt.Errorf("read world.log: %w", err)
	}
	var wl WorldLog
	for line := range bytes.Lines(data) {
		var rec struct {
			Msg     string `json:"msg"`
			PID     int    `json:"pid"`
			Control string `json:"control"`
			Node    string `json:"node"`
			DataDir string `json:"data_dir"`
		}
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		switch rec.Msg {
		case launchRecord:
			wl.Supervisor = Supervisor{PID: rec.PID, Log: path, Control: rec.Control}
		case spawnRecord:
			wl.Spawns = append(wl.Spawns, Spawn{Node: rec.Node, PID: rec.PID, DataDir: rec.DataDir})
		}
	}
	if wl.Supervisor.PID == 0 {
		return WorldLog{}, errors.New(path + " has no launch record")
	}
	return wl, nil
}
