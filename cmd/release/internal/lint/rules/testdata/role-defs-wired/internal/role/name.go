package role

type Name string

const (
	Searching   Name = "searching"
	Downloading Name = "downloading"
	Converting  Name = "converting"
)

type Wide string

const (
	Transfers Wide = "transfers"
	Moving    Wide = "moving"
	Relaying  Wide = "relaying"
)
