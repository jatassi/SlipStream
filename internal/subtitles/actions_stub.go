package subtitles

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/tasks"
)

// ErrNotBuilt is every manual subtitle action, and Subtitle search's
// cadence, until Subtitle search is built.
var ErrNotBuilt = errors.New("manual subtitle actions aren't built yet")

var (
	ErrSearchExpired  = errors.New("the subtitle search expired")
	ErrAttemptExpired = errors.New("the subtitle attempt expired")
	ErrUploadExpired  = errors.New("the refused upload expired")
	ErrImage          = errors.New("image subtitles can't be timed")
)

// SetCadence refuses: no Subtitle search task is in the catalog yet.
func (Unscheduled) SetCadence(context.Context, tasks.TaskID, setting.Stored[int64]) error {
	return ErrNotBuilt
}

// Reason is a ranking step a candidate meets.
type Reason uint8

const (
	MadeByPerson Reason = iota + 1
	SameGroup
	SameSource
	ProviderOrder
)

// Tried is an earlier try of a candidate for the same want.
type Tried struct {
	Outcome TriedOutcome
	Match   int
	At      time.Time
}

// TriedOutcome is how a try ended, a subtitles_tried outcome.
type TriedOutcome string

const (
	TriedPassed       TriedOutcome = "passed"
	TriedFailedTiming TriedOutcome = "failed_timing"
	TriedMarkedBad    TriedOutcome = "marked_bad"
	TriedUnreadable   TriedOutcome = "unreadable"
	TriedImage        TriedOutcome = "image"
	TriedWrongEpisode TriedOutcome = "wrong_episode"
	TriedTooLarge     TriedOutcome = "too_large"
)

// Candidate is one ranked offer of a Manual subtitle search.
type Candidate struct {
	Key      string
	Provider subtitling.ProviderID
	Release  string
	Reasons  []Reason
	Tried    *Tried
	Disabled bool
}

// Candidates is a Manual subtitle search so far.
type Candidates struct {
	Done       bool
	Candidates []Candidate
	Outcomes   []calls.Outcome
}

// Step is where a pick or an upload is.
type Step uint8

const (
	Downloading Step = iota + 1
	Checking
	Finished
)

// Ended is how a finished pick or upload ended.
type Ended uint8

const (
	Added Ended = iota + 1
	FailedTiming
	Unreadable
	Image
	WrongEpisode
	MarkedBad
)

// Attempt is a pick's or an upload's step, and its outcome once finished.
// Upload is set on an upload that failed and can be kept anyway.
type Attempt struct {
	Step      Step
	Provider  subtitling.ProviderID
	Ended     Ended
	Match     int
	Threshold int
	Upload    storage.UUID
}

func (s *Subtitles) SearchNowFor(context.Context, identity.Admin, WantKey) error { return ErrNotBuilt }

func (s *Subtitles) StartCandidates(context.Context, identity.Admin, WantKey) (storage.UUID, error) {
	return storage.UUID{}, ErrNotBuilt
}

func (s *Subtitles) Candidates(context.Context, identity.Admin, WantKey, storage.UUID) (Candidates, error) {
	return Candidates{}, ErrNotBuilt
}

func (s *Subtitles) Pick(context.Context, identity.Admin, WantKey, storage.UUID, string) (storage.UUID, error) {
	return storage.UUID{}, ErrNotBuilt
}

// Upload refuses bytes over Timings.UploadCap with *TooLargeError.
func (s *Subtitles) Upload(_ context.Context, _ identity.Admin, _ WantKey, _ string, b []byte) (storage.UUID, error) {
	if err := s.uploadShape(b); err != nil {
		return storage.UUID{}, err
	}
	return storage.UUID{}, ErrNotBuilt
}

func (s *Subtitles) KeepUpload(context.Context, identity.Admin, files.ID, storage.UUID) error {
	return ErrNotBuilt
}

func (s *Subtitles) AttemptOf(context.Context, identity.Admin, files.ID, storage.UUID) (Attempt, error) {
	return Attempt{}, ErrNotBuilt
}

func (s *Subtitles) MarkBadNow(context.Context, identity.Admin, WantKey) error { return ErrNotBuilt }
