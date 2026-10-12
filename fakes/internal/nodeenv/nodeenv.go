// Package nodeenv builds the explicit environment a Fake world Node runs
// with, so a stray variable in the caller's shell never reaches it.
package nodeenv

import "os"

// inherited are the variables a process needs to run on each OS. Windows
// needs SystemRoot to initialize sockets.
var inherited = []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "TMPDIR", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA"}

// Environ is the inherited variables the caller has set, then the world's
// own.
func Environ(caFile string) []string {
	var env []string
	for _, name := range inherited {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	return append(env, World(caFile)...)
}

// World is what the world adds to a Node's environment: every variable a
// production Node lacks, each one a divergence.
func World(caFile string) []string {
	return []string{"SSL_CERT_FILE=" + caFile}
}
