package api

import "errors"

type Problem struct {
	Title  string `json:"title" doc:"Please read this robust tag"`
	Detail string `json:"detail,omitempty" example:"Oops" doc:"What went wrong"`
}

const fallback = "Something went wrong"

var (
	errSorry = errors.New("sorry, PLEASE retry")
	dash     = "Ready — now"
	range_   = `1–3`
	empty    = " — "
	entity   = "Ready&mdash;now"
	spaced   = "You're all\n set"
	fine     = "pleased, seamlessness and allgood"
	adverb   = "Seamlessly synced"
	added    = "Let's enhance it"
	quote    = 'a'
)
