//go:build unix

package notifying

var passedVars = []string{"PATH", "HOME", "TMPDIR"}

func sameVar(a, b string) bool { return a == b }
