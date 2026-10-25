package api

import "errors"

type Problem struct {
	Title string `json:"title" doc:"Please read this robust tag"`
}

const fallback = "Something went wrong"

var (
	errSorry = errors.New("sorry, PLEASE retry")
	dash     = "Ready — now"
	range_   = `1–3`
	spaced   = "You're all\n set"
	fine     = "pleased, seamlessness and allgood"
	adverb   = "Seamlessly synced"
	quote    = 'a'
)
