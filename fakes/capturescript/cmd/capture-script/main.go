// Command capture-script plays an Admin's Custom Script. It reports its
// arguments, its variables and the event file to the capture-script fake
// at --report, then plays what its flags ask: print, write a lot, start a
// child that outlives it, sleep, and exit with a code.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	report := flag.String("report", "", "the capture-script fake's URL")
	exit := flag.Int("exit", 0, "the exit code")
	say := flag.String("say", "", "text to print")
	spam := flag.Int("spam", 0, "bytes to write")
	sleep := flag.Duration("sleep", 0, "how long to run")
	child := flag.Bool("child", false, "start a child that sleeps as long, then sleep")
	flag.Parse()

	if *report != "" {
		if err := send(*report, *exit); err != nil {
			fmt.Fprintln(os.Stderr, "capture-script:", err)
			os.Exit(125)
		}
	}
	fmt.Print(*say)
	fmt.Print(strings.Repeat("x", *spam))
	if *child {
		c := exec.CommandContext(context.Background(), os.Args[0], "--sleep", sleep.String())
		if err := c.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "capture-script:", err)
			os.Exit(125)
		}
	}
	time.Sleep(*sleep)
	os.Exit(*exit)
}

// run is what the fake keeps as a capturescript.Run. The binary doesn't
// import the fake, which would build the whole Node into it.
type run struct {
	Argv  []string          `json:"argv"`
	Env   map[string]string `json:"env"`
	Event json.RawMessage   `json:"event"`
	Exit  int               `json:"exit"`
}

func send(url string, exit int) error {
	r := run{Argv: os.Args, Env: map[string]string{}, Exit: exit}
	for _, kv := range os.Environ() {
		name, value, _ := strings.Cut(kv, "=")
		r.Env[name] = value
	}
	if path := os.Getenv("NOTIFICATION_EVENT_FILE"); path != "" {
		event, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read the event file: %w", err)
		}
		r.Event = event
	}
	body, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode the run: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/runs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build the report: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("report the run: %w", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("report the run: status %d", res.StatusCode)
	}
	return nil
}
