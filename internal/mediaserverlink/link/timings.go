package link

import "time"

// Timings is the media_servers block of the Node's Timings: protocol
// internals, never shown in the UI.
type Timings struct {
	// Call bounds one rpc call and one server request.
	Call time.Duration `yaml:"call"`
	// KeepAlive is how often the holder pings a server's push socket.
	KeepAlive    time.Duration `yaml:"keep_alive"`
	ReconnectMin time.Duration `yaml:"reconnect_min"`
	ReconnectMax time.Duration `yaml:"reconnect_max"`
	// RetryMin and RetryMax bound the backoff of a failed send.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// PinPoll is how often the Controller polls a Plex PIN. plex.tv wants
	// at least a second, and it doubles on 429.
	PinPoll time.Duration `yaml:"pin_poll"`
	// ScanEndPoll is the first wait between looks at a full scan, doubling
	// to ScanEndPollMax. A tiny Plex section scans in 0.25 s (recorded run
	// 2026-11-14, case G).
	ScanEndPoll    time.Duration `yaml:"scan_end_poll"`
	ScanEndPollMax time.Duration `yaml:"scan_end_poll_max"`
	// ScanEndBound is how long the fresh folder rule waits for a scan to
	// end.
	ScanEndBound time.Duration `yaml:"scan_end_bound"`
	// MovedConfirm bounds the wait, past the monitor delay, for a carry's
	// new item to appear.
	MovedConfirm time.Duration `yaml:"moved_confirm"`
	// MovedStep bounds each other waiting step of a carry, past the monitor
	// delay.
	MovedStep time.Duration `yaml:"moved_step"`
	// StepPoll is the wait between looks inside a waiting step.
	StepPoll time.Duration `yaml:"step_poll"`
	// RenameSnapshot bounds the snapshots taken once per disk queue pass
	// before a rename's bytes move.
	RenameSnapshot time.Duration `yaml:"rename_snapshot"`
	// UpgradePairing is how long either half of an upgrade waits for the
	// other, whatever Wait before sending is.
	UpgradePairing time.Duration `yaml:"upgrade_pairing"`
	// SectionFolders is how many folders of one Plex section Plan refreshes
	// one by one before it refreshes the section.
	SectionFolders int `yaml:"section_folders"`
	// SessionCap is the most sessions kept from one look at a server.
	SessionCap int `yaml:"session_cap"`
	// IndexAge is how old a path index entry may be before it is checked.
	IndexAge time.Duration `yaml:"index_age"`
	// SentAge is how long housekeeping keeps a sent change.
	SentAge time.Duration `yaml:"sent_age"`
	// IndexPage is how many items one Index step reads.
	IndexPage int `yaml:"index_page"`
}

// ScanWait is the wait before the next look at a full scan the fresh
// folder rule decided on at requested, zero for not yet: ScanEndPoll,
// growing with the time waited up to ScanEndPollMax.
func (t Timings) ScanWait(requested, now time.Time) time.Duration {
	if requested.IsZero() {
		return t.ScanEndPoll
	}
	return min(max(t.ScanEndPoll, now.Sub(requested)), t.ScanEndPollMax)
}

func DefaultTimings() Timings {
	return Timings{
		Call:           10 * time.Second,
		KeepAlive:      30 * time.Second,
		ReconnectMin:   time.Second,
		ReconnectMax:   time.Minute,
		RetryMin:       10 * time.Second,
		RetryMax:       15 * time.Minute,
		PinPoll:        time.Second,
		ScanEndPoll:    250 * time.Millisecond,
		ScanEndPollMax: 5 * time.Second,
		ScanEndBound:   2 * time.Hour,
		MovedConfirm:   2 * time.Minute,
		MovedStep:      5 * time.Minute,
		StepPoll:       2 * time.Second,
		RenameSnapshot: 20 * time.Second,
		UpgradePairing: 2 * time.Minute,
		SectionFolders: 10,
		SessionCap:     100,
		IndexAge:       24 * time.Hour,
		SentAge:        7 * 24 * time.Hour,
		IndexPage:      500,
	}
}
