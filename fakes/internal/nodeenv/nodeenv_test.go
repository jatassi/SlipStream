package nodeenv_test

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
)

func TestEnvironInheritsOnlyWhatAProcessNeeds(t *testing.T) {
	for _, name := range []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "TMPDIR", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA"} {
		t.Setenv(name, "")
	}
	t.Setenv("PATH", "/bin")
	t.Setenv("HOME", "/home/ada")
	t.Setenv("SLIPSTREAM_TORRENT_DHT", "true")
	t.Setenv("SSL_CERT_FILE", "/etc/ssl/cert.pem")
	got := nodeenv.Environ("/runs/r1/ca.pem")
	want := []string{
		"PATH=/bin", "HOME=/home/ada", "USERPROFILE=", "SystemRoot=", "TMPDIR=", "TEMP=", "TMP=", "LOCALAPPDATA=", "APPDATA=",
		"SSL_CERT_FILE=/runs/r1/ca.pem",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Environ = %q\nwant %q", got, want)
	}
}
