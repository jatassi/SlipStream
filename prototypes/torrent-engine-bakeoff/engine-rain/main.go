package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/cenkalti/rain/torrent"
	"go.etcd.io/bbolt"
)

type entry struct {
	Name     string `json:"name"`
	Infohash string `json:"infohash"`
	Torrent  string `json:"torrent"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
}

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

func fatal(err error) {
	if err != nil {
		status("error", map[string]any{"error": err.Error()})
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func writeResumeDB(path string, man []entry, firstPort int) {
	db, err := bbolt.Open(path, 0o600, nil)
	fatal(err)
	defer db.Close()
	fatal(db.Update(func(tx *bbolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("torrents"))
		if err != nil {
			return err
		}
		for i, e := range man {
			mi, err := metainfo.LoadFromFile(e.Torrent)
			if err != nil {
				return err
			}
			info, err := mi.UnmarshalInfo()
			if err != nil {
				return err
			}
			np := info.NumPieces()
			bf := make([]byte, (np+7)/8)
			for p := 0; p < np; p++ {
				bf[p/8] |= 1 << (7 - p%8)
			}
			trackers, _ := json.Marshal(mi.UpvertedAnnounceList())
			b, err := root.CreateBucketIfNotExists([]byte(e.Infohash))
			if err != nil {
				return err
			}
			b.Put([]byte("info_hash"), mi.HashInfoBytes().Bytes())
			b.Put([]byte("port"), []byte(strconv.Itoa(firstPort+i)))
			b.Put([]byte("name"), []byte(info.Name))
			b.Put([]byte("trackers"), trackers)
			b.Put([]byte("url_list"), []byte("null"))
			b.Put([]byte("fixed_peers"), []byte("null"))
			b.Put([]byte("info"), mi.InfoBytes)
			b.Put([]byte("bitfield"), bf)
			b.Put([]byte("added_at"), []byte(time.Now().Format(time.RFC3339)))
			b.Put([]byte("bytes_downloaded"), []byte("0"))
			b.Put([]byte("bytes_uploaded"), []byte("0"))
			b.Put([]byte("bytes_wasted"), []byte("0"))
			b.Put([]byte("seeded_for"), []byte("0s"))
			b.Put([]byte("started"), []byte("true"))
			b.Put([]byte("stop_after_download"), []byte("false"))
			b.Put([]byte("stop_after_metadata"), []byte("false"))
			b.Put([]byte("complete_cmd_run"), []byte("false"))
			b.Put([]byte("version"), []byte("3"))
		}
		return nil
	}))
}

func main() {
	mode := flag.String("mode", "seed", "")
	fix := flag.String("fixtures", "", "")
	port := flag.Int("listen-port", 42001, "")
	pidPrefix := flag.String("peer-id-prefix", "", "")
	statusPath := flag.String("status", "", "")
	index := flag.Int("index", 0, "")
	dataDir := flag.String("data-dir", "", "")
	flag.Parse()
	if *statusPath != "" {
		statusF, _ = os.OpenFile(*statusPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}
	var man []entry
	b, err := os.ReadFile(filepath.Join(*fix, "manifest.json"))
	fatal(err)
	fatal(json.Unmarshal(b, &man))

	tmp, err := os.MkdirTemp("", "rain-")
	fatal(err)
	defer os.RemoveAll(tmp)

	cfg := torrent.DefaultConfig
	cfg.Database = filepath.Join(tmp, "session.db")
	cfg.DataDirIncludesTorrentID = false
	cfg.Host = "127.0.0.1"
	cfg.DHTEnabled = false
	cfg.PEXEnabled = false
	cfg.RPCEnabled = false
	var rl syscall.Rlimit
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl)
	cfg.MaxOpenFiles = rl.Max
	cfg.ResumeOnStartup = true
	cfg.BlocklistEnabledForTrackers = false
	cfg.BlocklistEnabledForOutgoingConnections = false
	cfg.BlocklistEnabledForIncomingConnections = false
	cfg.HealthCheckInterval = time.Minute
	cfg.HealthCheckTimeout = 10 * time.Minute
	if *pidPrefix != "" {
		cfg.PrivatePeerIDPrefix = *pidPrefix
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	if *mode == "seed" {
		cfg.DataDir = filepath.Join(*fix, "data")
		cfg.PortBegin = uint16(*port)
		cfg.PortEnd = uint16(*port + len(man) + 1)
		t0 := time.Now()
		writeResumeDB(cfg.Database, man, *port)
		dbWrite := time.Since(t0).Seconds()
		t0 = time.Now()
		s, err := torrent.NewSession(cfg)
		fatal(err)
		load := time.Since(t0).Seconds()
		for {
			n := 0
			for _, t := range s.ListTorrents() {
				if t.Stats().Status == torrent.Seeding {
					n++
				}
			}
			if n == len(man) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		status("all_added", map[string]any{"elapsed": time.Since(start).Seconds(), "count": len(man), "db_write_s": dbWrite, "session_load_s": load})
		<-sig
		t0 = time.Now()
		shutdown(s)
		status("shutdown_done", map[string]any{"elapsed": time.Since(t0).Seconds()})
		return
	}

	e := man[*index]
	cfg.DataDir = *dataDir
	cfg.PortBegin = uint16(*port)
	cfg.PortEnd = uint16(*port + 2)
	s, err := torrent.NewSession(cfg)
	fatal(err)
	f, err := os.Open(e.Torrent)
	fatal(err)
	t, err := s.AddTorrent(f, nil)
	fatal(err)
	f.Close()
	go func() {
		<-sig
		shutdown(s)
		os.Exit(0)
	}()
	status("step", map[string]any{"step": "added"})
	for t.Stats().Status != torrent.Seeding {
		time.Sleep(100 * time.Millisecond)
	}
	status("step", map[string]any{"step": "completed"})
	time.Sleep(45 * time.Second)
	t.Stop()
	status("step", map[string]any{"step": "paused"})
	time.Sleep(45 * time.Second)
	t.Start()
	status("step", map[string]any{"step": "resumed"})
	time.Sleep(45 * time.Second)
	data := filepath.Join(*dataDir, e.Name)
	os.Link(data, data+".keep")
	s.RemoveTorrent(t.ID())
	status("step", map[string]any{"step": "removed"})
	time.Sleep(10 * time.Second)
	s.Close()
	status("step", map[string]any{"step": "shutdown"})
}

func shutdown(s *torrent.Session) {
	s.StopAll()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		busy := false
		for _, t := range s.ListTorrents() {
			if t.Stats().Status == torrent.Stopping {
				busy = true
			}
		}
		if !busy {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	s.Close()
}
