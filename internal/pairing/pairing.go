// Package pairing owns a Worker's half of a pairing: pairing.json and the
// join config section.
package pairing

import "github.com/jatassi/slipstream/internal/joincode"

// RecordFile is the pairing record in the data dir. Its existence makes a
// Node a Worker.
const RecordFile = "pairing.json"

// Settings is the join config section.
type Settings struct {
	// Code joins an Unclaimed Node at start. It is always redacted.
	Code joincode.Code `yaml:"code"`
}
