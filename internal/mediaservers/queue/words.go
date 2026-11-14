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
	link.ChangeSidecarAdded:    "Sent the new Sidecar.",
	link.ChangeSidecarReplaced: "Sent the replaced Sidecar.",
	link.ChangeSidecarRemoved:  "Sent the Sidecar's removal.",
	link.ChangeLibrary:         "Refreshed the whole library.",
}

var lost = map[link.Loss]string{
	link.LostWatchState:  "watch state",
	link.LostPlayCount:   "play count",
	link.LostLastPlayed:  "last played",
	link.LostRating:      "rating",
	link.LostCollections: "collections",
}

// Words is the sentence the API shows for c: what it did and what it lost,
// or what it waits for.
func (c Change) Words() string {
	switch c.State {
	case Prepared:
		switch {
		case c.Snapshot != nil:
			return "Saved every user's watch data before the rename. SlipStream sends this change once the rename finishes."
		case len(c.Lost) > 0:
			return "SlipStream couldn't save the watch data before the rename. It tries again when it sends this change."
		}
		return "SlipStream sends this change once the rename finishes."
	case Sending:
		return "SlipStream is sending this change."
	case Due:
		return c.waiting()
	case Sent:
	}
	return c.sent()
}

func (c Change) waiting() string {
	switch {
	case c.Problem != nil:
		e := (&link.ServerError{Problem: *c.Problem}).Error()
		return strings.ToUpper(e[:1]) + e[1:] + ". SlipStream retries automatically."
	case c.Step.Started():
		return "This change is partly sent. SlipStream finishes it automatically."
	case c.Pairing():
		return "This change is waiting for the other half of its upgrade."
	}
	return "This change is waiting to be sent."
}

func (c Change) sent() string {
	switch c.Result {
	case link.ResultCovered:
		return "A refresh of the whole library covered this change."
	case link.ResultConverted:
		return "The server had no item at the old path. SlipStream sent this change as a new file."
	case link.ResultAccepted:
	}
	s := did[c.Kind]
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
