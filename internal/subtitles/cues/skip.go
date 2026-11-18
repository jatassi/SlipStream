package cues

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	bracketed = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)`)
	credit    = regexp.MustCompile(`(?i)www\.|https?:|\.(com|org|net)\b|\b(subtitles?|subs|captions?|synced|sync|corrected|ripped|translated|translation|encoded|resynced?)\s+(by|and)\b|opensubtitles|addic7ed|subscene|subdl|subsource`)
)

// markSkips sets skip on each cue whose body, with markup removed, gives
// Timing sync no speech to place: no letters or digits outside brackets,
// or a credit line as the first or last cue.
func markSkips(cs []cue, bodies []string, markup *regexp.Regexp) {
	for i := range cs {
		body := markup.ReplaceAllString(bodies[i], " ")
		cs[i].skip = !spoken(bracketed.ReplaceAllString(body, " ")) ||
			(i == 0 || i == len(cs)-1) && credit.MatchString(body)
	}
}

func spoken(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) })
}
