// Package nodeenv builds the explicit environment a Fake world Node runs
// with, so a stray variable in the caller's shell never reaches it.
package nodeenv

import "os"

// inherited are the variables a process needs to run on each OS. Windows
// needs SystemRoot to initialize sockets.
var inherited = []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "TMPDIR", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA"}

// Environ is the inherited variables the caller has set, then
// SSL_CERT_FILE naming the run's CA.
func Environ(caFile string) []string {
	var env []string
	for _, name := range inherited {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	return append(env, "SSL_CERT_FILE="+caFile)
}
