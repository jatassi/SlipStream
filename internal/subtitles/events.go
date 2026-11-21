package subtitles

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/works"
)

// Changed names the Works and files whose subtitles read differently now,
// one per Write.
type Changed struct {
	WorkIDs []string `json:"workIds" nullable:"false"`
	FileIDs []string `json:"fileIds" nullable:"false"`
}

func changedOf(ws []works.WorkID, fs []files.ID) Changed {
	out := Changed{WorkIDs: make([]string, len(ws)), FileIDs: make([]string, len(fs))}
	for i, w := range ws {
		out.WorkIDs[i] = w.String()
	}
	for i, f := range fs {
		out.FileIDs[i] = f.String()
	}
	return out
}

// ProvidersChanged says a Subtitle Provider's row or the settings lead may
// read differently now.
type ProvidersChanged struct{}

// Attempted says a pick's or an upload's step or outcome moved.
type Attempted struct {
	AttemptID string `json:"attemptId" format:"uuid"`
	FileID    string `json:"fileId" format:"uuid"`
}

var (
	ChangedEvent          = live.AdminEvent[Changed]("subtitles.changed")
	ProvidersChangedEvent = live.AdminEvent[ProvidersChanged]("subtitles.providers-changed")
	AttemptEvent          = live.AdminEvent[Attempted]("subtitles.attempt")
)

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}
