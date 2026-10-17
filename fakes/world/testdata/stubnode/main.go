package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/internal/node"
)

var (
	resolved string
	torrent  string
	runExit  string
	// successor restarts as Windows does, with a new process that holds
	// this one's stderr, instead of in place.
	successor string
)

func main() {
	if len(os.Args) != 4 || os.Args[2] != "--data-dir" {
		fmt.Fprintln(os.Stderr, "usage: stubnode run|stop|restart|config --data-dir <dir>")
		os.Exit(node.ExitUsage)
	}
	dir := os.Args[3]
	switch os.Args[1] {
	case "config":
		config()
	case "run":
		run(dir)
	case "stop":
		send(dir, "")
	case "restart":
		send(dir, "restart")
	}
}

func config() {
	var r node.Resolved
	if err := json.Unmarshal([]byte(resolved), &r); err != nil {
		fmt.Fprintln(os.Stderr, "slipstream config:", err)
		os.Exit(node.ExitUsage)
	}
	out, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(out))
}

func run(dir string) {
	fmt.Fprintln(os.Stderr, "stub node starting, a line that is not JSON")
	if runExit != "" {
		fmt.Fprintln(os.Stderr, "stub node told to exit with "+runExit)
		code, _ := strconv.Atoi(runExit)
		os.Exit(code)
	}
	var cfg struct {
		Seed    string `yaml:"seed"`
		Torrent struct {
			Listen string `yaml:"listen"`
		} `yaml:"torrent"`
		HTTP struct {
			Listen string `yaml:"listen"`
		} `yaml:"http"`
		Link struct {
			Listen string `yaml:"listen"`
		} `yaml:"link"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	fail(err)
	fail(yaml.Unmarshal(data, &cfg))
	if torrent != "" {
		cfg.Torrent.Listen = torrent
	}
	d, err := node.ParseDataDir(dir)
	fail(err)
	fail(os.MkdirAll(d.RunDir(), 0o700))
	// A killed stub leaves its socket behind, which a Node removes.
	if err := os.Remove(d.ControlSocket()); err != nil && !errors.Is(err, os.ErrNotExist) {
		fail(err)
	}
	control, err := net.Listen("unix", d.ControlSocket())
	fail(err)

	fmt.Fprintln(os.Stderr, `{"msg":"setup code","code":"0000-0000-0000"}`)
	ready := node.Ready{
		PID:     os.Getpid(),
		DataDir: d.Path(),
		Version: "stub",
		Seed:    cfg.Seed,
		NodeKey: "0000-0000-0000-0000",
		Listeners: node.Listeners{
			Control: d.ControlSocket(),
			Torrent: bind(cfg.Torrent.Listen),
			HTTP:    bind(cfg.HTTP.Listen),
			Link:    bind(cfg.Link.Listen),
		},
	}
	line, err := json.Marshal(struct {
		Msg   string     `json:"msg"`
		Ready node.Ready `json:"ready"`
	}{"ready", ready})
	fail(err)
	fmt.Fprintln(os.Stderr, string(line))

	conn, err := control.Accept()
	fail(err)
	op, err := io.ReadAll(conn)
	fail(err)
	if string(op) == "restart" {
		fmt.Fprintln(os.Stderr, `{"msg":"stopped to restart"}`)
		_ = conn.Close()
		_ = control.Close()
		for _, l := range bound {
			_ = l.Close()
		}
		restart()
	}
	fmt.Fprintln(os.Stderr, `{"level":"ERROR","msg":"stub node told to stop","detail":"logged verbatim"}`)
	fmt.Fprintln(os.Stderr, `{"msg":"stopped"}`)
	_ = conn.Close()
	_ = control.Close()
}

// restart runs this executable again on the same arguments: in place, or
// as a successor that inherits stderr while this process exits 0.
func restart() {
	exe, err := os.Executable()
	fail(err)
	if successor == "" {
		fail(syscall.Exec(exe, os.Args, os.Environ()))
	}
	cmd := exec.Command(exe, os.Args[1:]...) //nolint:gosec,noctx // the stub's own executable and arguments, outliving this process
	cmd.Stderr = os.Stderr
	fail(cmd.Start())
	os.Exit(0)
}

// bind binds a listen address on port 0 and reports the port it took, as
// a Node does. It reports any other address as given.
func bind(listen string) netip.AddrPort {
	ap := netip.MustParseAddrPort(listen)
	if ap.Port() != 0 {
		return ap
	}
	l, err := net.Listen("tcp", listen)
	fail(err)
	bound = append(bound, l)
	return netip.MustParseAddrPort(l.Addr().String())
}

// bound keeps each listener open while the stub runs.
var bound []net.Listener

// send asks the Node on dir to run op, empty for a stop, and returns once
// it has closed the connection.
func send(dir, op string) {
	d, err := node.ParseDataDir(dir)
	fail(err)
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: d.ControlSocket(), Net: "unix"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "slipstream "+os.Args[1]+": no Node answered on "+dir)
		os.Exit(node.ExitFailed)
	}
	_, err = io.WriteString(conn, op)
	fail(err)
	// macOS answers ENOTCONN once the Node has closed; that is the answer.
	if err := conn.CloseWrite(); !errors.Is(err, syscall.ENOTCONN) {
		fail(err)
	}
	if _, err := io.Copy(io.Discard, conn); err != nil && !errors.Is(err, net.ErrClosed) {
		fail(err)
	}
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "stub node:", err)
		os.Exit(node.ExitFailed)
	}
}
