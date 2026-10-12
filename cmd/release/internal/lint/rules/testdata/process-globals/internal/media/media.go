package media

import (
	"log"
	"log/slog"
	mrand "math/rand"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
)

func Probe(logger *slog.Logger, r *rand.Rand) {
	_ = rand.IntN(10)
	_ = rand.NewPCG(1, 2)
	_ = rand.N(5)
	_ = rand.New(rand.NewChaCha8([32]byte{}))
	_ = r.IntN(10)
	_ = mrand.Int()
	_ = os.Getenv("HOME")
	_, _ = os.LookupEnv("HOME")
	_ = os.Environ()
	_ = os.Args
	slog.Info("probe")
	slog.SetDefault(logger)
	_ = slog.New(logger.Handler())
	logger.Info("probe", slog.String("k", "v"))
	log.Print("probe")
	_, _ = net.Listen("tcp", ":0")
	_, _ = net.ListenPacket("udp", ":0")
	_ = net.ListenConfig{}
	_, _ = net.Dial("tcp", ":0")
	_, _ = http.Get("/")
	_ = http.DefaultClient
	_ = http.ListenAndServe(":0", nil)
	_ = &http.Client{}
	http.Handle("/", nil)
	http.HandleFunc("/", nil)
	_ = http.DefaultServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/", nil)
	http.NewServeMux().Handle("/", nil)
}
