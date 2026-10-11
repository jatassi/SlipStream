package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"golang.org/x/time/rate"
)

type entry struct {
	Name     string `json:"name"`
	Infohash string `json:"infohash"`
	Torrent  string `json:"torrent"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
}

type allComplete struct{ gets atomic.Int64 }

func (a *allComplete) Get(metainfo.PieceKey) (storage.Completion, error) {
	a.gets.Add(1)
	return storage.Completion{Ok: true, Complete: true}, nil
}
func (a *allComplete) Set(metainfo.PieceKey, bool) error { return nil }
func (a *allComplete) Close() error                      { return nil }
func (a *allComplete) Persistent() bool                  { return false }

var statusF *os.File
var start = time.Now()

func status(event string, kv map[string]any) {
	m := map[string]any{"t": float64(time.Now().UnixNano()) / 1e9, "event": event}
	for k, v := range kv {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	if statusF != nil {
		statusF.Write(append(b, '\n'))
	}
}

func main() {
	mode := flag.String("mode", "seed", "")
	fix := flag.String("fixtures", "", "")
	port := flag.Int("listen-port", 42069, "")
	pidPrefix := flag.String("peer-id-prefix", "", "")
	statusPath := flag.String("status", "", "")
	index := flag.Int("index", 0, "")
	dataDir := flag.String("data-dir", "", "")
	ua := flag.String("user-agent", "", "")
	flag.Parse()
	if *statusPath != "" {
		statusF, _ = os.OpenFile(*statusPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}

	var man []entry
	b, err := os.ReadFile(filepath.Join(*fix, "manifest.json"))
	fatal(err)
	fatal(json.Unmarshal(b, &man))

	cfg := torrent.NewDefaultClientConfig()
	cfg.NoDHT = true
	cfg.DisablePEX = true
	cfg.NoDefaultPortForwarding = true
	cfg.Seed = true
	cfg.ListenPort = *port
	cfg.DisableIPv6 = true
	cfg.NoUpload = false
	cfg.DialRateLimiter = rate.NewLimiter(rate.Inf, 1)
	cfg.EstablishedConnsPerTorrent = 8
	cfg.HalfOpenConnsPerTorrent = 8
	cfg.TotalHalfOpenConns = 1000
	if *pidPrefix != "" {
		cfg.Bep20 = *pidPrefix
	}
	if *ua != "" {
		cfg.HTTPUserAgent = *ua
	}

	comp := &allComplete{}
	baseDir := filepath.Join(*fix, "data")
	if *mode == "lifecycle" {
		baseDir = *dataDir
	}
	opts := storage.NewFileClientOpts{ClientBaseDir: baseDir, UsePartFiles: g.Some(false)}
	if *mode == "seed" {
		opts.PieceCompletion = comp
	} else {
		opts.PieceCompletion = storage.NewMapPieceCompletion()
	}
	cfg.DefaultStorage = storage.NewFileOpts(opts)

	cl, err := torrent.NewClient(cfg)
	fatal(err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	if *mode == "seed" {
		var ts []*torrent.Torrent
		for _, e := range man {
			mi, err := metainfo.LoadFromFile(e.Torrent)
			fatal(err)
			spec := torrent.TorrentSpecFromMetaInfo(mi)
			spec.DisableInitialPieceCheck = true
			t, _, err := cl.AddTorrentSpec(spec)
			fatal(err)
			ts = append(ts, t)
		}
		for _, t := range ts {
			<-t.GotInfo()
			for !t.Complete().Bool() {
				time.Sleep(10 * time.Millisecond)
			}
		}
		status("all_added", map[string]any{"elapsed": time.Since(start).Seconds(), "count": len(ts), "completion_gets": comp.gets.Load()})
		<-sig
		t0 := time.Now()
		for _, t := range ts {
			t.Drop()
		}
		waitStopped(cl)
		if h := os.Getenv("HOLD_AFTER_DROP"); h != "" {
			d, _ := time.ParseDuration(h)
			time.Sleep(d)
		}
		cl.Close()
		status("shutdown_done", map[string]any{"elapsed": time.Since(t0).Seconds()})
		return
	}

	e := man[*index]
	mi, err := metainfo.LoadFromFile(e.Torrent)
	fatal(err)
	t, err := cl.AddTorrent(mi)
	fatal(err)
	t.DownloadAll()
	go func() {
		<-sig
		t.Drop()
		waitStopped(cl)
		cl.Close()
		os.Exit(0)
	}()
	status("step", map[string]any{"step": "added"})
	<-t.Complete().On()
	status("step", map[string]any{"step": "completed"})
	sleep(45)
	t.DisallowDataDownload()
	t.DisallowDataUpload()
	t.SetMaxEstablishedConns(0)
	status("step", map[string]any{"step": "paused"})
	sleep(45)
	t.SetMaxEstablishedConns(cfg.EstablishedConnsPerTorrent)
	t.AllowDataDownload()
	t.AllowDataUpload()
	status("step", map[string]any{"step": "resumed"})
	sleep(45)
	t.Drop()
	status("step", map[string]any{"step": "removed"})
	sleep(10)
	cl.Close()
	status("step", map[string]any{"step": "shutdown"})
}

func waitStopped(cl *torrent.Client) {
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		var sb strings.Builder
		cl.WriteStatus(&sb)
		if os.Getenv("DUMP_STATUS") != "" {
			os.WriteFile(os.Getenv("DUMP_STATUS"), []byte(sb.String()), 0o644)
		}
		if !strings.Contains(sb.String(), "stopped") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func sleep(s int) { time.Sleep(time.Duration(s) * time.Second) }

func fatal(err error) {
	if err != nil {
		status("error", map[string]any{"error": err.Error()})
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}
