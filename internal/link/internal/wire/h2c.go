package wire

import (
	"context"
	"log/slog"
	"net"
	"net/http"
)

func h2c() *http.Protocols {
	p := new(http.Protocols)
	p.SetUnencryptedHTTP2(true)
	return p
}

// http2 sets both receive windows to the stream window, so a stream that
// yamux lets grow is not capped one layer up.
func http2(window uint32) *http.HTTP2Config {
	return &http.HTTP2Config{
		MaxReceiveBufferPerConnection: int(window),
		MaxReceiveBufferPerStream:     int(window),
	}
}

// Server serves HTTP/2 with prior knowledge, and nothing else, on every
// conn its listener yields.
func Server(window uint32, h http.Handler, connContext func(context.Context, net.Conn) context.Context, log *slog.Logger) *http.Server {
	//nolint:gosec // every peer passed the pin check before a stream reaches this server
	return &http.Server{
		Handler:     h,
		Protocols:   h2c(),
		HTTP2:       http2(window),
		ConnContext: connContext,
		ErrorLog:    slog.NewLogLogger(log.Handler(), slog.LevelDebug),
	}
}

// Transport sends each request over HTTP/2 with prior knowledge on a
// connection from dial.
func Transport(window uint32, dial func(context.Context) (net.Conn, error)) *http.Transport {
	return &http.Transport{
		Protocols: h2c(),
		HTTP2:     http2(window),
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dial(ctx)
		},
	}
}
