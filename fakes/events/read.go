package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Read returns every record of the run in runDir in log order. A last
// line still being written is not a record yet.
func Read(runDir string) ([]Record, error) {
	data, err := os.ReadFile(filepath.Join(runDir, File))
	if err != nil {
		return nil, fmt.Errorf("read the event log: %w", err)
	}
	recs, _, err := parse(data, 0)
	return recs, err
}

// Select returns the records of the run in runDir that q matches.
func Select(runDir string, q Query) ([]Record, error) {
	recs, err := Read(runDir)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, r := range recs {
		if q.Matches(r) {
			out = append(out, r)
		}
	}
	return out, nil
}

// parse decodes the whole lines of data, which starts after line lines
// already read, and returns how many bytes they span.
func parse(data []byte, line int) ([]Record, int, error) {
	var recs []Record
	used := 0
	for {
		i := bytes.IndexByte(data[used:], '\n')
		if i < 0 {
			return recs, used, nil
		}
		line++
		var r Record
		if err := json.Unmarshal(data[used:used+i], &r); err != nil {
			return nil, 0, fmt.Errorf("%s:%d: %w", File, line, err)
		}
		recs = append(recs, r)
		used += i + 1
	}
}
