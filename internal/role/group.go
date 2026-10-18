package role

import "slices"

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

var groupLabels = map[Group]string{
	Acquire: "Acquire",
	Store:   "Store",
	Process: "Process",
	Connect: "Connect",
	Reach:   "Reach",
}

// Label is the Group's name as the UI writes it.
func (g Group) Label() string { return groupLabels[g] }
