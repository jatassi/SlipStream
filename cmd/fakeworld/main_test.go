package main_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
)

// commandArg makes the test binary a command for fakeworld exec to run.
const commandArg = "fakeworld-test-command"

func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == commandArg {
		os.Exit(command(os.Args[2], os.Args[3:]))
	}
	os.Exit(testbin.Run(m))
}

// command is what the test binary does as a step's command:
//
//	echo exit=<n> <arg>...   print each arg and a line on stderr, exit n
//	touch <path>             create path
//	read <path>              print path's content, exit 1 if it is missing
//	get <ca file> <url>      GET url trusting the CA, print the status
func command(mode string, args []string) int {
	switch mode {
	case "echo":
		code, _ := strconv.Atoi(strings.TrimPrefix(args[0], "exit="))
		for _, a := range args[1:] {
			fmt.Println(a)
		}
		fmt.Fprintln(os.Stderr, "stderr of the step")
		return code
	case "touch":
		if err := os.WriteFile(args[0], nil, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "read":
		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Print(string(data))
		return 0
	case "get":
		pem, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM(pem)
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
		res, err := client.Get(args[1]) //nolint:noctx,gosec // the step command reaches its own run's fake until fakeworld exec ends it
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		_ = res.Body.Close()
		fmt.Println(res.StatusCode)
		return 0
	}
	fmt.Fprintln(os.Stderr, "unknown mode", mode)
	return 2
}
