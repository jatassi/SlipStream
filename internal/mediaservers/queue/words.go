package queue

import (
	"strings"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

var did = map[link.ChangeKind]string{
	link.ChangeAdded:           "Sent the new file.",
	link.ChangeReplaced:        "Sent the replaced file.",
	link.ChangeRemoved:         "Sent the removal.",
	link.ChangeMoved:           "Sent the rename.",
	link.ChangeSidecarAdded:    "Sent the new subtitle file.",
	link.ChangeSidecarReplaced: "Sent the replaced subtitle file.",
	link.ChangeSidecarRemoved:  "Sent the subtitle file's removal.",
	link.ChangeLibrary:         "Refreshed the whole library.",
}

const watchData = "play count, resume position, favorite and rating"

var lost = map[link.Loss]string{
	link.LostWatchState:  "watch state",
	link.LostPlayCount:   "play count",
	link.LostLastPlayed:  "last played",
	link.LostRating:      "rating",
	link.LostCollections: "collections",
}

// Words is the sentence the API shows for c: what it did and what it lost,
// or what it waits for. to is how copy names c's root, which a Move's
// change names.
func (c Change) Words(to string) string {
	what := "rename"
	if c.Move() {
		what = "Move"
	}
	switch c.State {
	case Prepared:
		switch {
		case c.Snapshot != nil:
			return "Saved who watched it and each user's " + watchData + " before the " + what + ". SlipStream sends this change once the " + what + " finishes."
		case len(c.Lost) > 0:
			return "SlipStream couldn't save who watched it or each user's " + watchData + " before the " + what + ". It tries again when it sends this change."
		}
		return "SlipStream sends this change once the " + what + " finishes."
	case Sending:
		return "SlipStream is sending this change."
	case Due:
		return c.waiting()
	case Sent:
	}
	return c.sent(to)
}

func (c Change) waiting() string {
	switch {
	case c.Problem != nil:
		e := (&link.ServerError{Problem: *c.Problem}).Error()
		return strings.ToUpper(e[:1]) + e[1:] + ". SlipStream retries automatically."
	case c.Step.Started():
		return "This change is partly sent. SlipStream finishes it automatically."
	case c.Pairing():
		return "This change is part of an upgrade. SlipStream sends it together with the upgrade's other file change."
	}
	return "This change is waiting to be sent."
}

func (c Change) sent(to string) string {
	switch c.Result {
	case link.ResultCovered:
		return "A refresh of the whole library covered this change."
	case link.ResultConverted:
		return "The server had no item at the old path. SlipStream sent this change as a new file."
	case link.ResultAccepted:
	}
	s := did[c.Kind]
	if c.Move() {
		s = "Sent the Move to " + to + "."
	}
	switch {
	case !c.Carry():
		return s
	case len(c.Lost) == 0:
		return s + " Kept every user's watch data."
	}
	names := make([]string, len(c.Lost))
	for i, l := range c.Lost {
		names[i] = lost[l]
	}
	return s + " Lost: " + strings.Join(names, ", ") + "."
}
