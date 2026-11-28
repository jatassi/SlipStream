package notifying

import "strings"

var passedVars = []string{"PATH", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "TEMP", "TMP", "SystemRoot"}

func sameVar(a, b string) bool { return strings.EqualFold(a, b) }
