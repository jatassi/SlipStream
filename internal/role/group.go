package role

import (
	"fmt"
	"slices"
)

// Group is a Role Group: the Roles an Admin ticks together.
type Group string

const (
	Acquire Group = "acquire"
	Store   Group = "store"
	Process Group = "process"
	Connect Group = "connect"
	Reach   Group = "reach"
)

var groupOrder = []Group{Acquire, Store, Process, Connect, Reach}

// Groups is every Role Group, in the glossary's order.
func Groups() []Group { return slices.Clone(groupOrder) }

var groups = map[Name]Group{
	Searching:       Acquire,
	Downloading:     Acquire,
	Storage:         Store,
	Converting:      Process,
	Subtitles:       Process,
	Metadata:        Connect,
	MediaServerLink: Connect,
	Notifying:       Connect,
	Relaying:        Reach,
	Serving:         Reach,
}

// Group is the Role Group n belongs to.
func (n Name) Group() Group {
	g, ok := groups[n]
	if !ok {
		panic(fmt.Sprintf("role: %q is not a Role", string(n)))
	}
	return g
}
