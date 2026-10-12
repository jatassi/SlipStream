package notices

import (
	"strings"
	"unicode"
)

const (
	noLicense  = "NONE"
	unreadable = "NOASSERTION"
)

type fingerprint struct {
	license string
	phrases []string
}

// fingerprints are ordered so a denied license wins over a permissive one it
// contains, which also makes a file bundling several licenses report the
// strictest. A GPL text cannot say "or later", so it reads as "-only".
var fingerprints = []fingerprint{
	{"AGPL-3.0", []string{"gnu affero general public license version 3 19 november 2007"}},
	{"GPL-2.0-only", []string{"gnu general public license version 2 june 1991"}},
	{"BSD-4-Clause", []string{"all advertising materials mentioning features or use of this software must display the following acknowledgement"}},
	{"Apache-2.0", []string{"apache license version 2 0 january 2004"}},
	{"MPL-2.0", []string{"mozilla public license version 2 0"}},
	{"BSL-1.0", []string{"boost software license version 1 0"}},
	{"CC0-1.0", []string{"cc0 1 0 universal"}},
	{"BSD-3-Clause", []string{
		"redistributions of source code must retain the above copyright notice",
		"redistributions in binary form must reproduce the above copyright notice",
		"neither the name of",
	}},
	{"BSD-2-Clause", []string{
		"redistributions of source code must retain the above copyright notice",
		"redistributions in binary form must reproduce the above copyright notice",
	}},
	{"MIT", []string{
		"permission is hereby granted free of charge to any person obtaining a copy",
		"the above copyright notice and this permission notice shall be included in all copies or substantial portions of the software",
	}},
	{"ISC", []string{"permission to use copy modify and or distribute this software for any purpose with or without fee is hereby granted provided that the above copyright notice and this permission notice appear in all copies"}},
	{"0BSD", []string{"permission to use copy modify and or distribute this software for any purpose with or without fee is hereby granted the software is provided as is"}},
	{"Zlib", []string{
		"this software is provided as is without any express or implied warranty",
		"altered source versions must be plainly marked as such",
	}},
}

// Classify names the SPDX license a license file's text matches, or
// unreadable when it matches none.
func Classify(text string) string {
	words := " " + strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ") + " "
	for _, f := range fingerprints {
		if containsAll(words, f.phrases) {
			return f.license
		}
	}
	return unreadable
}

func containsAll(words string, phrases []string) bool {
	for _, p := range phrases {
		if !strings.Contains(words, " "+p+" ") {
			return false
		}
	}
	return true
}
