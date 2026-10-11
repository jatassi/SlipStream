package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type entry struct {
	Name     string `json:"name"`
	Infohash string `json:"infohash"`
	Torrent  string `json:"torrent"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
}

var (
	statusMu  sync.Mutex
	statusOut *os.File
)

func status(event string, kv ...any) {
	m := map[string]any{"t": float64(time.Now().UnixNano()) / 1e9, "event": event}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	b, _ := json.Marshal(m)
	statusMu.Lock()
	defer statusMu.Unlock()
	if statusOut != nil {
		statusOut.Write(append(b, '\n'))
	}
	fmt.Fprintln(os.Stderr, string(b))
}

func fatal(err error) {
	status("error", "error", err.Error())
	os.Exit(1)
}

func main() {
	mode := flag.String("mode", "seed", "")
	fixtures := flag.String("fixtures", "", "")
	port := flag.Int("listen-port", 6881, "")
	prefix := flag.String("peer-id-prefix", "", "")
	statusFile := flag.String("status", "", "")
	index := flag.Int("index", 0, "")
	dataDir := flag.String("data-dir", "", "")
	verbose := flag.Bool("v", false, "")
	flag.Parse()

	if *statusFile != "" {
		f, err := os.OpenFile(*statusFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			panic(err)
		}
		statusOut = f
	}

	raw, err := os.ReadFile(filepath.Join(*fixtures, "manifest.json"))
	if err != nil {
		fatal(err)
	}
	var man []entry
	if err := json.Unmarshal(raw, &man); err != nil {
		fatal(err)
	}

	kv := [][2]string{{"listen_interfaces", "127.0.0.1:" + strconv.Itoa(*port)}, {"ssrf_mitigation", "false"}}
	if *prefix != "" {
		kv = append(kv, [2]string{"peer_fingerprint", *prefix})
	}
	s, err := newSession(kv)
	if err != nil {
		fatal(err)
	}

	var alerts chan alert
	if *mode == "lifecycle" {
		alerts = make(chan alert, 64)
	}
	stop := make(chan struct{})
	pollDone := make(chan struct{})
	go func() {
		defer close(pollDone)
		buf := make([]ltAlertBuf, 64)
		for {
			select {
			case <-stop:
				return
			default:
			}
			for _, a := range s.poll(buf, 200) {
				if a.Kind == alertError || a.Kind == alertTrkError {
					status("error", "alert", a.What, "torrent", a.Torrent, "message", a.Message)
				} else if *verbose {
					fmt.Fprintf(os.Stderr, "alert %s t=%d %s\n", a.What, a.Torrent, a.Message)
				}
				if alerts != nil && a.Kind >= alertAdded && a.Kind <= alertRemoved {
					select {
					case alerts <- a:
					case <-stop:
						return
					}
				}
			}
		}
	}()
	shutdown := func() {
		close(stop)
		<-pollDone
		s.close()
	}

	switch *mode {
	case "seed":
		seed(s, man, shutdown)
	case "lifecycle":
		lifecycle(s, man[*index], *dataDir, alerts, shutdown)
	default:
		fatal(fmt.Errorf("unknown mode %q", *mode))
	}
}

func seed(s *session, man []entry, shutdown func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	start := time.Now()
	ts := make([]torrent, 0, len(man))
	for _, e := range man {
		t, err := s.add(e.Torrent, filepath.Dir(e.Data), true)
		if err != nil {
			status("error", "torrent", e.Name, "error", err.Error())
			continue
		}
		ts = append(ts, t)
	}
	for _, t := range ts {
		for {
			st, err := s.status(t)
			if err != nil || st.is_seeding != 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	status("all_added", "elapsed", time.Since(start).Seconds(), "count", len(ts))
	<-sig
	stopAt := time.Now()
	shutdown()
	status("shutdown_done", "elapsed", time.Since(stopAt).Seconds())
}

func lifecycle(s *session, e entry, dataDir string, alerts <-chan alert, shutdown func()) {
	start := time.Now()
	step := func(name string) { status("step", "step", name, "elapsed", time.Since(start).Seconds()) }
	waitFor := func(t torrent, kind int) {
		for a := range alerts {
			if a.Torrent == t && a.Kind == kind {
				return
			}
		}
	}
	t, err := s.add(e.Torrent, dataDir, false)
	if err != nil {
		fatal(err)
	}
	step("added")
	waitFor(t, alertFinished)
	st, _ := s.status(t)
	status("step", "step", "completed", "elapsed", time.Since(start).Seconds(), "total_done", int64(st.total_done))
	time.Sleep(45 * time.Second)
	if err := s.pause(t); err != nil {
		fatal(err)
	}
	waitFor(t, alertPaused)
	step("paused")
	time.Sleep(45 * time.Second)
	if err := s.resume(t); err != nil {
		fatal(err)
	}
	waitFor(t, alertResumed)
	step("resumed")
	time.Sleep(45 * time.Second)
	if err := s.remove(t); err != nil {
		fatal(err)
	}
	waitFor(t, alertRemoved)
	step("removed")
	time.Sleep(10 * time.Second)
	shutdown()
	step("shutdown")
}
