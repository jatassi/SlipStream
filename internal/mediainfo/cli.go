package mediainfo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// findExecutable finds an executable by name or explicit path.
func findExecutable(name, explicitPath string) string {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			return explicitPath
		}
	}

	// Try PATH lookup
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	// Platform-specific common locations
	var commonPaths []string
	switch runtime.GOOS {
	case "darwin":
		commonPaths = []string{
			"/usr/local/bin/" + name,
			"/opt/homebrew/bin/" + name,
		}
	case "linux":
		commonPaths = []string{
			"/usr/bin/" + name,
			"/usr/local/bin/" + name,
		}
	case "windows":
		commonPaths = []string{
			`C:\Program Files\MediaInfo\` + name + ".exe",
			`C:\Program Files (x86)\MediaInfo\` + name + ".exe",
		}
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// probeWithMediaInfo extracts info using mediainfo CLI.
func (s *Service) probeWithMediaInfo(ctx context.Context, path, binaryPath string) (*MediaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--Output=JSON", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mediainfo failed: %w: %s", err, stderr.String())
	}

	return parseMediaInfoJSON(stdout.Bytes())
}

// probeWithFFprobe extracts info using ffprobe CLI.
func (s *Service) probeWithFFprobe(ctx context.Context, path, binaryPath string) (*MediaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w: %s", err, stderr.String())
	}

	return parseFFprobeJSON(stdout.Bytes())
}

// mediaInfoOutput represents the JSON output from mediainfo.
type mediaInfoOutput struct {
	Media struct {
		Track []mediaInfoTrack `json:"track"`
	} `json:"media"`
}

type mediaInfoTrack struct {
	Type                string `json:"@type"`
	Format              string `json:"Format"`
	FormatProfile       string `json:"Format_Profile"`
	CodecID             string `json:"CodecID"`
	Width               string `json:"Width"`
	Height              string `json:"Height"`
	BitDepth            string `json:"BitDepth"`
	ColorPrimaries      string `json:"colour_primaries"`
	TransferCharacteristics string `json:"transfer_characteristics"`
	MatrixCoefficients  string `json:"matrix_coefficients"`
	HDRFormat           string `json:"HDR_Format"`
	HDRFormatCompatibility string `json:"HDR_Format_Compatibility"`
	Channels            string `json:"Channels"`
	ChannelLayout       string `json:"ChannelLayout"`
	Language            string `json:"Language"`
	Duration            string `json:"Duration"`
	FileSize            string `json:"FileSize"`
}

// parseMediaInfoJSON parses mediainfo JSON output.
func parseMediaInfoJSON(data []byte) (*MediaInfo, error) {
	var output mediaInfoOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse mediainfo output: %w", err)
	}

	info := &MediaInfo{}
	var audioLangs, subLangs []string
	var firstVideo, firstAudio bool

	for _, track := range output.Media.Track {
		switch track.Type {
		case "General":
			info.ContainerFormat = track.Format
			if track.FileSize != "" {
				if size, err := strconv.ParseInt(track.FileSize, 10, 64); err == nil {
					info.FileSize = size
				}
			}
			if track.Duration != "" {
				if dur, err := parseDuration(track.Duration); err == nil {
					info.Duration = dur
				}
			}

		case "Video":
			if firstVideo {
				continue
			}
			firstVideo = true

			info.VideoCodec = NormalizeVideoCodec(track.Format)
			if track.CodecID != "" && info.VideoCodec == track.Format {
				info.VideoCodec = NormalizeVideoCodec(track.CodecID)
			}

			if w, err := parseInt(track.Width); err == nil {
				info.Width = w
			}
			if h, err := parseInt(track.Height); err == nil {
				info.Height = h
			}
			if info.Width > 0 && info.Height > 0 {
				info.VideoResolution = fmt.Sprintf("%dx%d", info.Width, info.Height)
			}

			if bd, err := parseInt(track.BitDepth); err == nil {
				info.VideoBitDepth = bd
			}

			// Detect HDR
			hdrInfo := HDRInfo{
				BitDepth:       info.VideoBitDepth,
				ColorPrimaries: track.ColorPrimaries,
				TransferFunc:   track.TransferCharacteristics,
				MatrixCoeffs:   track.MatrixCoefficients,
				HDRFormat:      track.HDRFormat + " " + track.HDRFormatCompatibility,
			}
			info.DynamicRange, info.DynamicRangeType = DetectHDRType(hdrInfo)

		case "Audio":
			hasAtmos := strings.Contains(strings.ToLower(track.Format+track.FormatProfile), "atmos")

			if !firstAudio {
				firstAudio = true
				info.AudioCodec = NormalizeAudioCodec(track.Format, hasAtmos)

				channels, _ := parseInt(track.Channels)
				info.AudioChannels = FormatChannels(channels, track.ChannelLayout)
			}

			if track.Language != "" && track.Language != "und" {
				audioLangs = appendUnique(audioLangs, normalizeLanguage(track.Language))
			}

		case "Text":
			if track.Language != "" && track.Language != "und" {
				subLangs = appendUnique(subLangs, normalizeLanguage(track.Language))
			}
		}
	}

	info.AudioLanguages = audioLangs
	info.SubtitleLanguages = subLangs

	return info, nil
}

// ffprobeOutput represents the JSON output from ffprobe.
type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
}

type ffprobeStream struct {
	CodecType       string `json:"codec_type"`
	CodecName       string `json:"codec_name"`
	CodecTagString  string `json:"codec_tag_string"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	PixFmt          string `json:"pix_fmt"`
	ColorPrimaries  string `json:"color_primaries"`
	ColorTransfer   string `json:"color_transfer"`
	ColorSpace      string `json:"color_space"`
	Channels        int    `json:"channels"`
	ChannelLayout   string `json:"channel_layout"`
	Tags            ffprobeTags `json:"tags"`
	SideDataList    []ffprobeSideData `json:"side_data_list"`
}

type ffprobeTags struct {
	Language string `json:"language"`
}

type ffprobeSideData struct {
	SideDataType string `json:"side_data_type"`
}

// parseFFprobeJSON parses ffprobe JSON output.
func parseFFprobeJSON(data []byte) (*MediaInfo, error) {
	var output ffprobeOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &MediaInfo{}
	var audioLangs, subLangs []string
	var firstVideo, firstAudio bool

	info.ContainerFormat = output.Format.FormatName
	if output.Format.Size != "" {
		if size, err := strconv.ParseInt(output.Format.Size, 10, 64); err == nil {
			info.FileSize = size
		}
	}
	if output.Format.Duration != "" {
		if dur, err := parseFFprobeDuration(output.Format.Duration); err == nil {
			info.Duration = dur
		}
	}

	for _, stream := range output.Streams {
		switch stream.CodecType {
		case "video":
			if firstVideo {
				continue
			}
			firstVideo = true

			info.VideoCodec = NormalizeVideoCodec(stream.CodecName)
			info.Width = stream.Width
			info.Height = stream.Height
			if info.Width > 0 && info.Height > 0 {
				info.VideoResolution = fmt.Sprintf("%dx%d", info.Width, info.Height)
			}

			// Detect bit depth from pixel format
			info.VideoBitDepth = detectBitDepth(stream.PixFmt)

			// Detect HDR
			hdrInfo := HDRInfo{
				BitDepth:       info.VideoBitDepth,
				ColorPrimaries: stream.ColorPrimaries,
				TransferFunc:   stream.ColorTransfer,
				MatrixCoeffs:   stream.ColorSpace,
			}

			// Check for Dolby Vision in side data
			for _, sd := range stream.SideDataList {
				if strings.Contains(strings.ToLower(sd.SideDataType), "dolby vision") {
					hdrInfo.HasDolbyVision = true
				}
			}

			info.DynamicRange, info.DynamicRangeType = DetectHDRType(hdrInfo)

		case "audio":
			if !firstAudio {
				firstAudio = true
				info.AudioCodec = NormalizeAudioCodec(stream.CodecName, false)
				info.AudioChannels = FormatChannels(stream.Channels, stream.ChannelLayout)
			}

			if stream.Tags.Language != "" && stream.Tags.Language != "und" {
				audioLangs = appendUnique(audioLangs, normalizeLanguage(stream.Tags.Language))
			}

		case "subtitle":
			if stream.Tags.Language != "" && stream.Tags.Language != "und" {
				subLangs = appendUnique(subLangs, normalizeLanguage(stream.Tags.Language))
			}
		}
	}

	info.AudioLanguages = audioLangs
	info.SubtitleLanguages = subLangs

	return info, nil
}

// parseInt parses an int from a string, ignoring non-numeric suffixes.
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	// Remove non-numeric characters (like "1920 pixels")
	for i, c := range s {
		if c < '0' || c > '9' {
			s = s[:i]
			break
		}
	}
	return strconv.Atoi(s)
}

// parseDuration parses a duration string from mediainfo.
func parseDuration(s string) (time.Duration, error) {
	// MediaInfo outputs duration in seconds as a decimal
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(f * float64(time.Second)), nil
}

// parseFFprobeDuration parses a duration string from ffprobe.
func parseFFprobeDuration(s string) (time.Duration, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(f * float64(time.Second)), nil
}

// normalizeLanguage normalizes a language code to uppercase 2-3 letter code.
func normalizeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if len(lang) > 3 {
		lang = lang[:3]
	}
	return strings.ToUpper(lang)
}

// detectBitDepth detects bit depth from pixel format string.
func detectBitDepth(pixFmt string) int {
	lower := strings.ToLower(pixFmt)
	switch {
	case strings.Contains(lower, "10le"), strings.Contains(lower, "10be"),
		strings.Contains(lower, "p010"), strings.Contains(lower, "yuv420p10"):
		return 10
	case strings.Contains(lower, "12le"), strings.Contains(lower, "12be"):
		return 12
	case strings.Contains(lower, "8"):
		return 8
	default:
		return 8
	}
}

// appendUnique appends a value to a slice if not already present.
func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}
