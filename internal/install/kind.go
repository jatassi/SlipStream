// Package install names how a binary was installed: the Kind its marker
// file states, and every fact a Kind decides, such as the default data dir
// and the systemd units. The Node and the release tool both read it, so a
// name, a path or a unit is written once.
package install

import (
	"errors"
	"fmt"

	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// DataName is the per-user data folder's name. It differs from v1's
// SlipStream in more than case, because NTFS and APFS ignore case and v2
// runs beside v1.
const DataName = "SlipStream2"

// Kind is how a binary was installed. The zero Kind is Dev: no marker.
type Kind uint8

const (
	Dev Kind = iota
	MSI
	ZIP
	App
	Deb
	RPM
	AppImage
	Tarball
	Docker
)

type dataDirRule uint8

const (
	noDefault dataDirRule = iota
	localAppData
	besideExecutable
	applicationSupport
	varLib
	xdgData
	config
)

type row struct {
	token         string
	goos          string
	proto         linkv1.InstallKind
	dataDir       dataDirRule
	selfInstalled bool
	portsToTry    int
}

var kinds = [...]row{
	Dev:      {token: "dev", proto: linkv1.InstallKind_INSTALL_KIND_DEV, dataDir: noDefault, portsToTry: 10},
	MSI:      {token: "msi", goos: "windows", proto: linkv1.InstallKind_INSTALL_KIND_MSI, dataDir: localAppData, portsToTry: 10},
	ZIP:      {token: "zip", goos: "windows", proto: linkv1.InstallKind_INSTALL_KIND_WINDOWS_ZIP, dataDir: besideExecutable, portsToTry: 10},
	App:      {token: "app", goos: "darwin", proto: linkv1.InstallKind_INSTALL_KIND_MACOS_APP, dataDir: applicationSupport, portsToTry: 10},
	Deb:      {token: "deb", goos: "linux", proto: linkv1.InstallKind_INSTALL_KIND_DEB, dataDir: varLib, portsToTry: 10},
	RPM:      {token: "rpm", goos: "linux", proto: linkv1.InstallKind_INSTALL_KIND_RPM, dataDir: varLib, portsToTry: 10},
	AppImage: {token: "appimage", goos: "linux", proto: linkv1.InstallKind_INSTALL_KIND_APPIMAGE, dataDir: xdgData, selfInstalled: true, portsToTry: 10},
	Tarball:  {token: "tarball", goos: "linux", proto: linkv1.InstallKind_INSTALL_KIND_TARBALL, dataDir: xdgData, selfInstalled: true, portsToTry: 10},
	Docker:   {token: "docker", goos: "linux", proto: linkv1.InstallKind_INSTALL_KIND_DOCKER, dataDir: config, portsToTry: 1},
}

// Kinds is every Kind, Dev first.
func Kinds() []Kind {
	out := make([]Kind, len(kinds))
	for i := range kinds {
		out[i] = Kind(i)
	}
	return out
}

// String is the Kind's marker token.
func (k Kind) String() string { return kinds[k].token }

func (k Kind) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

func (k *Kind) UnmarshalText(b []byte) error {
	parsed, ok := parseToken(string(b))
	if !ok {
		return fmt.Errorf("%q is not an install kind", b)
	}
	*k = parsed
	return nil
}

// GOOS is the only OS the Kind runs on, or "" for Dev, which runs on any.
func (k Kind) GOOS() string { return kinds[k].goos }

func (k Kind) Proto() linkv1.InstallKind { return kinds[k].proto }

// SelfInstalled reports a Kind no package manager installed: a tarball or
// an AppImage. Only these may install a user service, and only these check
// the host's glibc and kernel floor at start.
func (k Kind) SelfInstalled() bool { return kinds[k].selfInstalled }

// PortsToTry is http.ports_to_try's default. Docker tries only its own
// port, which the container's port mapping names.
func (k Kind) PortsToTry() int { return kinds[k].portsToTry }

func parseToken(token string) (Kind, bool) {
	for _, k := range Kinds() {
		if k.String() == token {
			return k, true
		}
	}
	return Dev, false
}

// ErrNoDefault is Dev's data dir rule: a build without a marker never
// opens a real install's data, so the data dir must be named.
var ErrNoDefault = errors.New("this build has no default data dir")

// DefaultDataDir is where k keeps its data on l when no data dir is named.
func (k Kind) DefaultDataDir(l Layout) (string, error) {
	switch kinds[k].dataDir {
	case noDefault:
		return "", ErrNoDefault
	case localAppData:
		return l.under("LOCALAPPDATA", DataName)
	case besideExecutable:
		return l.join(l.dir(l.Executable), "data"), nil
	case applicationSupport:
		return l.under("HOME", "Library", "Application Support", DataName)
	case varLib:
		return "/var/lib/slipstream", nil
	case xdgData:
		if dir := l.Getenv("XDG_DATA_HOME"); l.isAbs(dir) {
			return l.join(dir, linuxDataName), nil
		}
		return l.under("HOME", ".local", "share", linuxDataName)
	case config:
		return "/config", nil
	default:
		panic(fmt.Sprintf("install: kind %d has data dir rule %d", k, kinds[k].dataDir))
	}
}
