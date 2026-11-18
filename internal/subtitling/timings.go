package subtitling

import (
	"time"

	"github.com/jatassi/slipstream/internal/calls"
)

// Timings bound a holder's work. Timing sync's constants aren't here: the
// Controller sends them with each call.
type Timings struct {
	// Calls is the call layer of the subtitles Lines.
	Calls calls.Timings `yaml:"calls"`
	// FetchCap bounds a fetched or synced subtitle's bytes.
	FetchCap int `yaml:"fetch_cap"`
	// ArchiveCap bounds an archive's unpacked bytes, and ArchiveMembers its
	// subtitle members.
	ArchiveCap     int `yaml:"archive_cap"`
	ArchiveMembers int `yaml:"archive_members"`
	// PackMembers and PackBytes bound the other members of a pack one Fetch
	// answer carries, so it fits one link message.
	PackMembers int `yaml:"pack_members"`
	PackBytes   int `yaml:"pack_bytes"`
}

func DefaultTimings() Timings {
	return Timings{
		Calls:          calls.Timings{Attempts: 3, Try: 30 * time.Second, RetryMin: time.Second, RetryMax: 8 * time.Second, Wait: 10 * time.Second},
		FetchCap:       5 << 20,
		ArchiveCap:     32 << 20,
		ArchiveMembers: 400,
		PackMembers:    24,
		PackBytes:      12 << 20,
	}
}
