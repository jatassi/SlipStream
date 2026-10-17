package media_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	mediav1 "github.com/jatassi/slipstream/internal/pb/slipstream/media/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
)

func TestParseFFprobeReadsTheFormatAndFirstVideoStream(t *testing.T) {
	out, err := os.ReadFile("testdata/ffprobe-1080p.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := media.ParseFFprobe(out)
	want := media.Probe{Duration: 6 * time.Second, Container: "matroska,webm", Codec: "h264", Width: 1920, Height: 1080}
	if err != nil || got != want {
		t.Fatalf("ParseFFprobe = %+v, %v; want %+v", got, err, want)
	}
}

func TestParseFFprobeRefusesAFileWithoutVideo(t *testing.T) {
	out := []byte(`{"streams":[{"codec_type":"audio","codec_name":"aac"}],"format":{"format_name":"mp4","duration":"3.5"}}`)
	if _, err := media.ParseFFprobe(out); err == nil || err.Error() != "the file has no video stream" {
		t.Fatalf("ParseFFprobe(audio only) = %v, want the file has no video stream", err)
	}
}

func TestIsVideo(t *testing.T) {
	for name, want := range map[string]bool{
		"a/Orchard.mkv": true, "Orchard.MP4": true, "x.m4v": true, "x.avi": true, "x.ts": true,
		"x.nfo": false, "sample": false, "a.mkv/readme.txt": false,
	} {
		if got := media.IsVideo(name); got != want {
			t.Errorf("IsVideo(%q) = %v, want %v", name, got, want)
		}
	}
}

type payloads map[bencode.InfoHash]string

func (p payloads) Path(h bencode.InfoHash, index int) (string, error) {
	path, ok := p[h]
	if !ok || index != 0 {
		return "", errors.New("the torrent is not in this Node's client")
	}
	return path, nil
}

func TestServiceProbesTheFileWhereItLies(t *testing.T) {
	hash := bencode.InfoHash{1}
	prober := mediatest.Prober{Of: map[string]media.Probe{"orchard.mkv": {Duration: 6 * time.Second, Container: "matroska,webm", Codec: "h264", Width: 1920, Height: 1080}}}
	svc := media.NewService(prober, payloads{hash: "/downloads/0100/orchard.mkv", {2}: "/downloads/0200/notes.mkv"}, media.DefaultTimings())
	remote := media.Remote(func(role.NodeID) mediav1connect.MediaServiceHandler { return svc })

	got, err := remote.Probe(context.Background(), role.NodeID{}, hash, 0)
	if want := (media.Probe{Duration: 6 * time.Second, Container: "matroska,webm", Codec: "h264", Width: 1920, Height: 1080}); err != nil || got != want {
		t.Fatalf("Probe = %+v, %v; want %+v", got, err, want)
	}
	_, err = svc.Probe(context.Background(), &mediav1.ProbeRequest{InfoHash: make([]byte, 20), FileIndex: 0})
	if rpc.CodeOf(err) != rpc.CodeUnavailable || err.Error() != "unavailable: the torrent is not in this Node's client" {
		t.Fatalf("Probe of a torrent not here = %v, want unavailable", err)
	}
	_, err = remote.Probe(context.Background(), role.NodeID{}, bencode.InfoHash{2}, 0)
	if rpc.CodeOf(err) != rpc.CodeFailedPrecondition || err.Error() != "failed_precondition: mediatest: no probe for notes.mkv" {
		t.Fatalf("Probe of an unreadable file = %v, want failed_precondition", err)
	}
}
