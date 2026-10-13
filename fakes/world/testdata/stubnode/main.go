package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/internal/node"
)

var (
	resolved string
	torrent  string
	runExit  string
)

func main() {
	if len(os.Args) != 4 || os.Args[2] != "--data-dir" {
		fmt.Fprintln(os.Stderr, "usage: stubnode run|stop|config --data-dir <dir>")
		os.Exit(node.ExitUsage)
	}
	dir := os.Args[3]
	switch os.Args[1] {
	case "config":
		config()
	case "run":
		run(dir)
	case "stop":
		stop(dir)
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
	control, err := net.Listen("unix", d.ControlSocket())
	fail(err)

	fmt.Fprintln(os.Stderr, `{"msg":"setup code","code":"0000-0000-0000"}`)
	ready := node.Ready{
		PID:     os.Getpid(),
		DataDir: d.Path(),
		Version: "stub",
		Seed:    cfg.Seed,
		Listeners: node.Listeners{
			Control: d.ControlSocket(),
			Torrent: netip.MustParseAddrPort(cfg.Torrent.Listen),
			HTTP:    netip.MustParseAddrPort(cfg.HTTP.Listen),
			Link:    netip.MustParseAddrPort(cfg.Link.Listen),
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
	fmt.Fprintln(os.Stderr, `{"msg":"stopped"}`)
	_ = conn.Close()
	_ = control.Close()
}

func stop(dir string) {
	d, err := node.ParseDataDir(dir)
	fail(err)
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: d.ControlSocket(), Net: "unix"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "slipstream stop: no Node answered on "+dir)
		os.Exit(node.ExitFailed)
	}
	fail(conn.CloseWrite())
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
