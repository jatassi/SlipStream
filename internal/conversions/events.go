package conversions

import (
	"context"
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/history/entry"
)

// failed fails the Conversion with the move's code and records Conversion
// failed. failed has no Fault transition, so a doubled report records
// nothing more.
func (m *move) failed(ctx context.Context, class Class) error {
	words := failedWords(m.row.Snapshot, m.code, class, m.failedOn, m.retry.Why)
	m.row.Failure = &Failure{Code: m.code, Class: class, Detail: words}
	m.row.Pin, m.row.Exclude = nil, nil
	return ConversionFailed.Record(ctx, m.tx, m.c.cfg.History, FailedFacts{
		Work:   entry.Work{ID: m.row.Work, Label: m.row.Snapshot.Work, Type: m.row.Type},
		Target: m.row.Snapshot.Target, Nodes: m.failedOn, Code: m.code, Class: class, Words: words,
	})
}

// failedWords is a failed Conversion's sentence: what failed and where,
// why, and for a source fault what to change.
func failedWords(s Snapshot, code Code, class Class, on []string, why RetryWhy) string {
	of := fmt.Sprintf("The %s Conversion of %s", s.Target, s.Work)
	if class == SourceClass {
		return of + " can't run. " + sourceWords[code](s.Target)
	}
	out := fmt.Sprintf("%s failed on %s. %s", of, joined(on), runWords[code])
	if why == NoOtherNode {
		out += " No other Node can convert it."
	}
	return out
}

var runWords = map[Code]string{
	CodeEncoder:      "The video encoder failed.",
	CodeFilter:       "Scaling or tone mapping the video failed.",
	CodeDevice:       "The GPU stopped responding.",
	CodeCrash:        "ffmpeg stopped unexpectedly.",
	CodeStalled:      "ffmpeg stopped making progress.",
	CodeShortOutput:  "The output was shorter than the source.",
	CodeStreams:      "The output was missing a video or audio track its settings make.",
	CodeCopyFailed:   "Copying the source file to the Node failed.",
	CodeReturnFailed: "Copying the output back to its library folder failed.",
}

var sourceWords = map[Code]func(target string) string{
	CodeUnreadable: func(string) string {
		return "ffmpeg can't read the source file. Search for a new file."
	},
	CodeContainerAudio: func(t string) string {
		return fmt.Sprintf("The source file has TrueHD or DTS audio, and %s makes MP4 files with copied audio. Change the audio in %s's Conversion settings.", t, t)
	},
	CodeRefused: func(t string) string {
		return fmt.Sprintf("The file it made doesn't meet %s's rules. Change %s's Conversion settings or its filters.", t, t)
	},
}

func joined(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
