package storing

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Fold is the one key for comparing paths without case: NFC, then
// cases.Fold, so "Heat.mkv" and "heat.mkv" share a key, and so do a
// composed and a decomposed "é".
func Fold(rel string) string { return cases.Fold().String(norm.NFC.String(rel)) }
