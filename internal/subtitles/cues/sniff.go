package cues

import (
	"bytes"
	"regexp"
)

// image is the image format b's bytes are, or zero. A VobSub .sub is an
// MPEG program stream, its .idx a text index, and a PGS .sup a run of
// segments that each open with "PG".
func image(b []byte) Format {
	switch {
	case bytes.HasPrefix(b, []byte{0, 0, 1, 0xBA}):
		return VobSub
	case bytes.HasPrefix(bytes.TrimLeft(b, "\xef\xbb\xbf \t\r\n"), []byte("# VobSub index file")):
		return VobSub
	case len(b) >= 13 && b[0] == 'P' && b[1] == 'G' && pgsSegments[b[10]]:
		return PGS
	}
	return 0
}

var pgsSegments = map[byte]bool{0x14: true, 0x15: true, 0x16: true, 0x17: true, 0x80: true}

var (
	sniffSRT       = regexp.MustCompile(`^\s*\d{1,3}:\d{1,2}:\d{1,2}[,.]\d{1,3}\s*-->\s*\d{1,3}:\d{1,2}:\d{1,2}[,.]\d{1,3}`)
	sniffMicroDVD  = regexp.MustCompile(`^\{\d+\}\{\d*\}`)
	sniffMPL2      = regexp.MustCompile(`^\[\d+\]\[\d*\]`)
	sniffSubViewer = regexp.MustCompile(`^(\[INFORMATION\]|\d{1,2}:\d{2}:\d{2}\.\d{1,3},\d{1,2}:\d{2}:\d{2}\.\d{1,3}\s*$)`)
	sniffTMPlayer  = regexp.MustCompile(`^\d{1,2}:\d{2}:\d{2}(,\d+)?[:=]`)
	sniffASS       = regexp.MustCompile(`(?im)^\s*\[events\]\s*$`)
	sniffDialogue  = regexp.MustCompile(`(?im)^\s*dialogue\s*:`)
	sniffSSA       = regexp.MustCompile(`(?im)^\s*(scripttype\s*:\s*v4\.00\s*$|\[v4 styles\])`)
	sniffSAMI      = regexp.MustCompile(`(?i)<sami[\s>]`)
	sniffTTML      = regexp.MustCompile(`<(\w+:)?tt[\s>]`)
)

// sniffLines is how many non-blank lines sniff reads for a line format.
const sniffLines = 50

// sniff is the text format text is in, or zero, read from the bytes alone.
func sniff(text []byte) Format {
	head := bytes.TrimLeft(text, " \t\r\n")
	switch {
	case bytes.HasPrefix(head, []byte("WEBVTT")):
		return WebVTT
	case sniffASS.Match(text) && sniffDialogue.Match(text):
		if sniffSSA.Match(text) {
			return SSA
		}
		return ASS
	case sniffSAMI.Match(head[:min(len(head), 4096)]):
		return SAMI
	case bytes.HasPrefix(head, []byte("<")) && sniffTTML.Match(head[:min(len(head), 4096)]):
		return TTML
	}
	seen := 0
	for line := range bytes.Lines(text) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		switch {
		case sniffSRT.Match(line):
			return SRT
		case sniffMicroDVD.Match(line):
			return MicroDVD
		case sniffMPL2.Match(line):
			return MPL2
		case sniffSubViewer.Match(line):
			return SubViewer
		case sniffTMPlayer.Match(line):
			return TMPlayer
		}
		seen++
		if seen == sniffLines {
			break
		}
	}
	return 0
}
