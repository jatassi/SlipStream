package api

var (
	notDownloading = "That Node doesn't hold Downloading"
	stands         = "Every Role the Node stands in, in table order"
	standsFor      = "The saved Custom Format the draft stands in for"
	writeOff       = "Write off"
	writtenOff     = "Its Node was written off."
	route          = "/works/{workId}/cells"
	cell           = "3 cells become wanted."
	want           = "Another Want already chose that one."
	wants          = "Wants"
)

type view struct {
	Status string `json:"status" doc:"The first state its Items hold, in the glossary's order"`
}
