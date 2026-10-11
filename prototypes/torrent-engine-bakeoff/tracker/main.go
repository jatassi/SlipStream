package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type peer struct {
	ip   [4]byte
	port uint16
	left int64
	seen time.Time
}

type swarm struct{ peers map[string]*peer }

type annState struct {
	dupAt map[string]time.Time
	minAt time.Time
}

type logLine struct {
	T          float64  `json:"t"`
	Listener   string   `json:"listener"`
	Path       string   `json:"path"`
	Passkey    string   `json:"passkey"`
	Infohash   string   `json:"infohash"`
	PeerID     string   `json:"peer_id"`
	PeerIDHex  string   `json:"peer_id_hex"`
	UA         string   `json:"ua"`
	Event      string   `json:"event"`
	Uploaded   int64    `json:"uploaded"`
	Downloaded int64    `json:"downloaded"`
	Left       int64    `json:"left"`
	Port       int      `json:"port"`
	Numwant    int      `json:"numwant"`
	Violations []string `json:"violations"`
	Rejected   bool     `json:"rejected"`
}

var (
	interval, minInterval int
	enforce               bool
	delayMs               int
	failIH                = map[string]bool{}
	mu                    sync.Mutex
	swarms                = map[string]*swarm{}
	states                = map[string]*annState{}
	logCh                 = make(chan []byte, 1<<18)
)

func esc(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c < 0x7f && c != '\\' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "\\x%02x", c)
		}
	}
	return b.String()
}

func atoi(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }

func bstr(s string) string { return strconv.Itoa(len(s)) + ":" + s }

func fail(w http.ResponseWriter, msg string) {
	w.Write([]byte("d" + bstr("failure reason") + bstr(msg) + "e"))
}

func handler(listener string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[1] != "announce" {
			http.NotFound(w, r)
			return
		}
		now := time.Now()
		q, _ := parseRaw(r.URL.RawQuery)
		ih := hex.EncodeToString([]byte(q["info_hash"]))
		pid := q["peer_id"]
		ev := q["event"]
		passkey := parts[0]
		port := int(atoi(q["port"]))
		left := atoi(q["left"])
		numwant := 50
		if v, ok := q["numwant"]; ok {
			numwant = int(atoi(v))
		}
		ll := logLine{
			T: float64(now.UnixNano()) / 1e9, Listener: listener, Path: r.URL.Path, Passkey: passkey,
			Infohash: ih, PeerID: esc(pid), PeerIDHex: hex.EncodeToString([]byte(pid)), UA: r.UserAgent(),
			Event: ev, Uploaded: atoi(q["uploaded"]), Downloaded: atoi(q["downloaded"]), Left: left,
			Port: port, Numwant: numwant, Violations: []string{},
		}

		w.Header().Set("Content-Type", "text/plain")
		if len(q["info_hash"]) != 20 || len(pid) != 20 {
			ll.Rejected = true
			emit(&ll)
			fail(w, "bad request")
			return
		}
		if failIH[ih] {
			ll.Rejected = true
			emit(&ll)
			fail(w, "fake failure")
			return
		}

		key := passkey + "|" + ih + "|" + pid
		mu.Lock()
		st := states[key]
		if st == nil {
			st = &annState{dupAt: map[string]time.Time{}}
			states[key] = st
		}
		dupEv := ev
		if dupEv == "" {
			dupEv = "empty"
		}
		if t, ok := st.dupAt[dupEv]; ok && now.Sub(t) < 30*time.Second {
			ll.Violations = append(ll.Violations, "duplicate_event_30s")
		} else {
			st.dupAt[dupEv] = now
			window := time.Duration(0.9 * float64(minInterval) * float64(time.Second))
			if st.minAt.IsZero() || now.Sub(st.minAt) >= window {
				st.minAt = now
			} else if ev == "stopped" && now.Sub(st.minAt) > 5*time.Minute {
				st.minAt = time.Time{}
			} else if ev != "completed" && ev != "stopped" {
				ll.Violations = append(ll.Violations, "min_interval_violation")
			}
		}
		if enforce && len(ll.Violations) > 0 {
			ll.Rejected = true
			mu.Unlock()
			emit(&ll)
			fail(w, "announce violation: "+strings.Join(ll.Violations, ","))
			return
		}

		sw := swarms[ih]
		if sw == nil {
			sw = &swarm{peers: map[string]*peer{}}
			swarms[ih] = sw
		}
		if ev == "stopped" {
			delete(sw.peers, pid)
		} else {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if v := q["ip"]; v != "" {
				host = v
			}
			if ip := net.ParseIP(host).To4(); ip != nil {
				p := sw.peers[pid]
				if p == nil {
					p = &peer{}
					sw.peers[pid] = p
				}
				copy(p.ip[:], ip)
				p.port = uint16(port)
				p.left = left
				p.seen = now
			}
		}
		exp := now.Add(-2 * time.Duration(interval) * time.Second)
		var complete, incomplete int
		buf := make([]byte, 0, 6*50)
		for id, p := range sw.peers {
			if p.seen.Before(exp) {
				delete(sw.peers, id)
				continue
			}
			if p.left == 0 {
				complete++
			} else {
				incomplete++
			}
			if id != pid && ev != "stopped" && len(buf) < 6*numwant && len(buf) < 6*200 {
				buf = append(buf, p.ip[:]...)
				buf = binary.BigEndian.AppendUint16(buf, p.port)
			}
		}
		mu.Unlock()
		emit(&ll)
		out := "d8:completei" + strconv.Itoa(complete) + "e10:incompletei" + strconv.Itoa(incomplete) +
			"e8:intervali" + strconv.Itoa(interval) + "e12:min intervali" + strconv.Itoa(minInterval) +
			"e5:peers" + strconv.Itoa(len(buf)) + ":" + string(buf) + "e"
		w.Write([]byte(out))
	}
}

func parseRaw(raw string) (map[string]string, error) {
	m := map[string]string{}
	for _, kv := range strings.Split(raw, "&") {
		k, v, _ := strings.Cut(kv, "=")
		m[unesc(k)] = unesc(v)
	}
	return m, nil
}

func unesc(s string) string {
	if !strings.ContainsAny(s, "%") {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
				b = append(b, byte(n))
				i += 2
				continue
			}
		}
		b = append(b, s[i])
	}
	return string(b)
}

func emit(l *logLine) {
	b, _ := json.Marshal(l)
	logCh <- append(b, '\n')
}

func main() {
	logPath := flag.String("log", "tracker.jsonl", "")
	flag.IntVar(&interval, "interval", 1800, "")
	flag.IntVar(&minInterval, "min-interval", 900, "")
	listen := flag.String("listen", "127.0.0.1:7070", "")
	failList := flag.String("fail-infohash", "", "")
	flag.BoolVar(&enforce, "enforce", false, "")
	flag.IntVar(&delayMs, "delay-ms", 0, "")
	flag.Parse()
	for _, h := range strings.Split(*failList, ",") {
		if h != "" {
			failIH[strings.ToLower(h)] = true
		}
	}

	f, err := os.Create(*logPath)
	if err != nil {
		panic(err)
	}
	done := make(chan struct{})
	go func() {
		bw := bufio.NewWriterSize(f, 1<<20)
		tk := time.NewTicker(time.Second)
		for {
			select {
			case b, ok := <-logCh:
				if !ok {
					bw.Flush()
					f.Close()
					close(done)
					return
				}
				bw.Write(b)
			case <-tk.C:
				bw.Flush()
			}
		}
	}()

	var srvs []*http.Server
	for _, addr := range strings.Split(*listen, ",") {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		s := &http.Server{Handler: handler(addr)}
		srvs = append(srvs, s)
		go s.Serve(ln)
		fmt.Fprintln(os.Stderr, "listening", addr)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range srvs {
		s.Shutdown(ctx)
	}
	close(logCh)
	<-done
}
