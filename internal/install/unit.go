package install

import (
	"strconv"
	"strings"
	"text/template"

	"github.com/jatassi/slipstream/internal/exitcode"
)

const (
	// SystemExecutable is where the deb and rpm place the binary.
	SystemExecutable = "/usr/lib/slipstream/slipstream"
	// UnitName names both the system and the user unit.
	UnitName = "slipstream.service"
	// SystemUser owns the system service and /var/lib/slipstream.
	SystemUser = "slipstream"
)

var unitTemplate = template.Must(template.New("unit").Parse(`[Unit]
Description=SlipStream
{{- if .System}}
Wants=network-online.target
After=network-online.target
{{- end}}

[Service]
{{- if .System}}
User={{.User}}
StateDirectory=slipstream
StateDirectoryMode=0700
AmbientCapabilities=CAP_NET_BIND_SERVICE
{{- end}}
ExecStart={{.ExecStart}}
Restart=on-failure
RestartPreventExitStatus={{.NoRestart}}
TimeoutStopSec=45

[Install]
WantedBy={{.WantedBy}}
`))

type unit struct {
	System    bool
	User      string
	ExecStart string
	NoRestart string
	WantedBy  string
}

func render(u unit) string {
	codes := make([]string, len(exitcode.NoRestart))
	for i, c := range exitcode.NoRestart {
		codes[i] = strconv.Itoa(c)
	}
	u.NoRestart = strings.Join(codes, " ")
	var b strings.Builder
	if err := unitTemplate.Execute(&b, u); err != nil {
		panic("install: render unit: " + err.Error())
	}
	return b.String()
}

// SystemUnit is the deb and rpm unit: the packaged binary as the slipstream
// user on its default data dir.
func SystemUnit() string {
	return render(unit{System: true, User: SystemUser, ExecStart: execStart(SystemExecutable, "run"), WantedBy: "multi-user.target"})
}

// UserUnit is service install --user's unit: exe running the Node on dataDir.
func UserUnit(exe, dataDir string) string {
	return render(unit{ExecStart: execStart(exe, "run", "--data-dir", dataDir), WantedBy: "default.target"})
}

// execStart quotes each argument that needs it in systemd's syntax, where
// % starts a specifier and $ a variable.
func execStart(args ...string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		a = strings.NewReplacer("%", "%%", "$", "$$").Replace(a)
		if strings.ContainsAny(a, " \t\"'\\;") {
			a = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(a) + `"`
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}
