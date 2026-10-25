package install

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

func TestDataNameDiffersFromV1InMoreThanCase(t *testing.T) {
	if strings.EqualFold(DataName, "SlipStream") {
		t.Fatalf("DataName %q matches v1's SlipStream ignoring case", DataName)
	}
}

func TestEveryKindHasARow(t *testing.T) {
	tokens := map[string]Kind{}
	protos := map[linkv1.InstallKind]Kind{}
	for _, k := range Kinds() {
		if prev, dup := tokens[k.String()]; dup || k.String() == "" {
			t.Errorf("kind %d has token %q, also kind %d's", k, k.String(), prev)
		}
		tokens[k.String()] = k
		if prev, dup := protos[k.Proto()]; dup || k.Proto() == linkv1.InstallKind_INSTALL_KIND_UNSPECIFIED {
			t.Errorf("kind %s has proto %s, also kind %s's", k, k.Proto(), prev)
		}
		protos[k.Proto()] = k
		if got, ok := parseToken(k.String()); !ok || got != k {
			t.Errorf("token %q parses to %s, %v", k, got, ok)
		}
	}
	for v, name := range linkv1.InstallKind_name {
		if _, ok := protos[linkv1.InstallKind(v)]; !ok && v != 0 {
			t.Errorf("proto %s has no Kind", name)
		}
	}
}

func TestDefaultDataDir(t *testing.T) {
	win := Layout{GOOS: "windows", Executable: `C:\Users\ana\AppData\Local\Programs\SlipStream\slipstream.exe`, Environ: []string{`LOCALAPPDATA=C:\Users\ana\AppData\Local`}}
	zip := Layout{GOOS: "windows", Executable: `D:\Portable\SlipStream\slipstream.exe`}
	mac := Layout{GOOS: "darwin", Executable: "/Applications/SlipStream.app/Contents/MacOS/slipstream", Environ: []string{"HOME=/Users/ana"}}
	linux := Layout{GOOS: "linux", Executable: "/home/ana/slipstream-1.0.0/slipstream", Environ: []string{"HOME=/home/ana"}}
	xdg := Layout{GOOS: "linux", Executable: "/tmp/.mount_SlipSt/usr/bin/slipstream", Environ: []string{"HOME=/home/ana", "XDG_DATA_HOME=/data/ana"}}
	relativeXDG := Layout{GOOS: "linux", Environ: []string{"HOME=/home/ana", "XDG_DATA_HOME=data"}}
	for _, tc := range []struct {
		kind Kind
		l    Layout
		want string
	}{
		{MSI, win, `C:\Users\ana\AppData\Local\SlipStream2`},
		{ZIP, zip, `D:\Portable\SlipStream\data`},
		{App, mac, "/Users/ana/Library/Application Support/SlipStream2"},
		{Deb, linux, "/var/lib/slipstream"},
		{RPM, linux, "/var/lib/slipstream"},
		{Tarball, linux, "/home/ana/.local/share/slipstream2"},
		{AppImage, xdg, "/data/ana/slipstream2"},
		{Tarball, relativeXDG, "/home/ana/.local/share/slipstream2"},
		{Docker, linux, "/config"},
	} {
		got, err := tc.kind.DefaultDataDir(tc.l)
		if err != nil || got != tc.want {
			t.Errorf("%s on %s: %q, %v; want %q", tc.kind, tc.l.GOOS, got, err, tc.want)
		}
	}
	if _, err := Dev.DefaultDataDir(linux); !errors.Is(err, ErrNoDefault) {
		t.Errorf("dev: %v, want ErrNoDefault", err)
	}
	noHome := Layout{GOOS: "darwin"}
	if _, err := App.DefaultDataDir(noHome); err == nil || err.Error() != `HOME is "", not an absolute path` {
		t.Errorf("app without HOME: %v", err)
	}
}

func TestMarkerPath(t *testing.T) {
	for _, tc := range []struct {
		l    Layout
		want string
	}{
		{Layout{GOOS: "windows", Executable: `C:\Programs\SlipStream\slipstream.exe`}, `C:\Programs\SlipStream\install-kind`},
		{Layout{GOOS: "darwin", Executable: "/Applications/SlipStream.app/Contents/MacOS/slipstream"}, "/Applications/SlipStream.app/Contents/Resources/install-kind"},
		{Layout{GOOS: "darwin", Executable: "/Users/ana/src/slipstream"}, "/Users/ana/src/install-kind"},
		{Layout{GOOS: "linux", Executable: "/usr/lib/slipstream/slipstream"}, "/usr/lib/slipstream/install-kind"},
	} {
		if got := MarkerPath(tc.l); got != tc.want {
			t.Errorf("MarkerPath(%s) = %q, want %q", tc.l.Executable, got, tc.want)
		}
	}
}

func linuxTempDir(t *testing.T) string { return filepath.ToSlash(t.TempDir()) }

func TestRead(t *testing.T) {
	dir := linuxTempDir(t)
	l := Layout{GOOS: "linux", Executable: dir + "/slipstream"}
	marker := dir + "/" + MarkerName
	if k, err := Read(l); k != Dev || err != nil {
		t.Fatalf("no marker: %s, %v; want dev", k, err)
	}
	for _, tc := range []struct {
		content string
		want    Kind
		err     string
	}{
		{content: "tarball", want: Tarball},
		{content: "deb\n", want: Deb},
		{content: "deb\n\n", err: `install marker ` + marker + ` holds "deb\n\n", not an install kind`},
		{content: " docker", err: `install marker ` + marker + ` holds " docker", not an install kind`},
		{content: "snap\n", err: `install marker ` + marker + ` holds "snap\n", not an install kind`},
		{content: "msi\n", err: `install marker ` + marker + ` holds "msi\n", a windows install kind, on linux`},
	} {
		if err := os.WriteFile(marker, []byte(tc.content), 0o600); err != nil {
			t.Fatal(err)
		}
		k, err := Read(l)
		if tc.err != "" {
			if err == nil || err.Error() != tc.err {
				t.Errorf("marker %q: %s, %v; want %s", tc.content, k, err, tc.err)
			}
			continue
		}
		if err != nil || k != tc.want {
			t.Errorf("marker %q: %s, %v; want %s", tc.content, k, err, tc.want)
		}
	}
}

func TestWriteIsWhatReadReads(t *testing.T) {
	dir := linuxTempDir(t)
	l := Layout{GOOS: "linux", Executable: dir + "/slipstream"}
	if err := Write(MarkerPath(l), AppImage); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dir + "/install-kind")
	if err != nil || string(b) != "appimage\n" {
		t.Fatalf("marker = %q, %v; want appimage and a newline", b, err)
	}
	if k, err := Read(l); k != AppImage || err != nil {
		t.Fatalf("Read = %s, %v; want appimage", k, err)
	}
}

func TestSystemUnit(t *testing.T) {
	const want = `[Unit]
Description=SlipStream
Wants=network-online.target
After=network-online.target

[Service]
User=slipstream
StateDirectory=slipstream
StateDirectoryMode=0700
AmbientCapabilities=CAP_NET_BIND_SERVICE
ExecStart=/usr/lib/slipstream/slipstream run
Restart=on-failure
RestartPreventExitStatus=3 4 5 6
TimeoutStopSec=45

[Install]
WantedBy=multi-user.target
`
	if got := SystemUnit(); got != want {
		t.Fatalf("system unit:\n%s\nwant:\n%s", got, want)
	}
}

func TestUserUnit(t *testing.T) {
	const want = `[Unit]
Description=SlipStream

[Service]
ExecStart="/home/ana/my apps/slipstream" run --data-dir /home/ana/100%%/$$HOME
Restart=on-failure
RestartPreventExitStatus=3 4 5 6
TimeoutStopSec=45

[Install]
WantedBy=default.target
`
	if got := UserUnit("/home/ana/my apps/slipstream", "/home/ana/100%/$HOME"); got != want {
		t.Fatalf("user unit:\n%s\nwant:\n%s", got, want)
	}
}

const tarballUnit = `[Unit]
Description=SlipStream

[Service]
ExecStart=/home/ana/slipstream-1.0.0/slipstream run --data-dir /home/ana/.local/share/slipstream2
Restart=on-failure
RestartPreventExitStatus=3 4 5 6
TimeoutStopSec=45

[Install]
WantedBy=default.target
`

var tarball = Layout{GOOS: "linux", Executable: "/home/ana/slipstream-1.0.0/slipstream", Environ: []string{"HOME=/home/ana"}}

func TestUserServiceInstallPlan(t *testing.T) {
	const unitPath = "/home/ana/.config/systemd/user/slipstream.service"
	linger := Command{Args: []string{"loginctl", "enable-linger", "1000"}}
	off := Linger{UID: 1000}
	written := func(path, unit string) []Step {
		return []Step{
			WriteFile{Path: path, Content: unit},
			Command{Args: []string{"systemctl", "--user", "daemon-reload"}},
			Command{Args: []string{"systemctl", "--user", "enable", "slipstream.service"}},
			Command{Args: []string{"systemctl", "--user", "restart", "slipstream.service"}},
			linger,
			AwaitRunning{},
		}
	}
	appImage := Layout{GOOS: "linux", Executable: "/tmp/.mount_SlipStabc/usr/bin/slipstream", Environ: []string{"HOME=/home/ana", "XDG_CONFIG_HOME=/cfg", "APPIMAGE=/home/ana/SlipStream-1.0.0-x86_64.AppImage"}}
	appImageUnit := strings.Replace(tarballUnit, "/home/ana/slipstream-1.0.0/slipstream", "/home/ana/SlipStream-1.0.0-x86_64.AppImage", 1)
	for _, tc := range []struct {
		name    string
		kind    Kind
		l       Layout
		current []byte
		want    []Step
	}{
		{"fresh", Tarball, tarball, nil, written(unitPath, tarballUnit)},
		{"identical rerun", Tarball, tarball, []byte(tarballUnit), []Step{
			Command{Args: []string{"systemctl", "--user", "daemon-reload"}},
			Command{Args: []string{"systemctl", "--user", "enable", "--now", "slipstream.service"}},
			linger,
			AwaitRunning{},
		}},
		{"changed unit", Tarball, tarball, []byte(strings.Replace(tarballUnit, "slipstream2", "old", 1)), written(unitPath, tarballUnit)},
		{"appimage", AppImage, appImage, nil, written("/cfg/systemd/user/slipstream.service", appImageUnit)},
	} {
		got, err := UserServiceInstall(tc.kind, tc.l, "/home/ana/.local/share/slipstream2", tc.current, off)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %#v, %v\nwant %#v", tc.name, got, err, tc.want)
		}
	}
}

func TestUserServiceInstallLeavesLingerOnAlone(t *testing.T) {
	got, err := UserServiceInstall(Tarball, tarball, "/home/ana/.local/share/slipstream2", []byte(tarballUnit), Linger{UID: 1000, On: true})
	want := []Step{
		Command{Args: []string{"systemctl", "--user", "daemon-reload"}},
		Command{Args: []string{"systemctl", "--user", "enable", "--now", "slipstream.service"}},
		AwaitRunning{},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%#v, %v\nwant %#v", got, err, want)
	}
}

func TestUserServiceIsForSelfInstalledKindsOnly(t *testing.T) {
	_, err := UserServiceInstall(Deb, tarball, "/var/lib/slipstream", nil, Linger{})
	if want := "a user service is for a tarball or AppImage install, and this is a deb install"; err == nil || err.Error() != want {
		t.Fatalf("deb: %v, want %s", err, want)
	}
	noAppImage := Layout{GOOS: "linux", Executable: "/tmp/.mount_x/usr/bin/slipstream", Environ: []string{"HOME=/home/ana"}}
	_, err = UserServiceInstall(AppImage, noAppImage, "/d", nil, Linger{})
	if want := `APPIMAGE is "", not an absolute path. Start service install --user from the AppImage`; err == nil || err.Error() != want {
		t.Fatalf("appimage without APPIMAGE: %v, want %s", err, want)
	}
}

func TestUserServiceUninstallPlan(t *testing.T) {
	got, err := UserServiceUninstall(Tarball, tarball, true)
	want := []Step{
		Command{Args: []string{"systemctl", "--user", "disable", "--now", "slipstream.service"}},
		RemoveFile{Path: "/home/ana/.config/systemd/user/slipstream.service"},
		Command{Args: []string{"systemctl", "--user", "daemon-reload"}},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("uninstall: %#v, %v\nwant %#v", got, err, want)
	}
	if got, err := UserServiceUninstall(Tarball, tarball, false); err != nil || got != nil {
		t.Fatalf("uninstall with no unit: %#v, %v; want no steps", got, err)
	}
}
