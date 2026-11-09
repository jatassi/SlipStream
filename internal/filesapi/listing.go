package filesapi

import (
	"context"
	"errors"
	"math"
	"path"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ListedFile is a library file as the Work page's Files section lists it.
type ListedFile struct {
	ID         string                             `json:"id" format:"uuid"`
	Path       string                             `json:"path" doc:"Relative to its root, / separated"`
	Folder     string                             `json:"folder" doc:"The folder it sits in, relative to its root, / separated"`
	Size       int64                              `json:"size" minimum:"0"`
	Quality    string                             `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam"`
	Codec      api.Nullable[api.Enum[videoCodec]] `json:"codec" doc:"Its video codec in the words a Target filters on; null for a codec they have no word for"`
	Cut        string                             `json:"cut" doc:"The key of its own cut"`
	Label      string                             `json:"label" doc:"The Target name its file name carries, before any suffix"`
	Kept       bool                               `json:"kept" doc:"A Kept file, which fills no cell"`
	Fills      []fill                             `json:"fills" nullable:"false" doc:"The cells it fills; empty for a Kept or gone file"`
	ModifiedAt *time.Time                         `json:"modifiedAt" doc:"Its modification time as a library scan or its import last read it; null until one did"`
	GoneAt     *time.Time                         `json:"goneAt" doc:"When a library scan found it missing from its root; null while it is there"`
	ProbeError *string                            `json:"probeError" doc:"The first line ffprobe printed when the last library scan couldn't read it, and media is then the read before; null when that read worked"`
	Media      api.Nullable[mediaInfo]            `json:"media" doc:"What ffprobe read of it; null for a file read before SlipStream kept its tracks"`
}

type fill struct {
	ItemID   string `json:"itemId" format:"uuid"`
	Edition  string `json:"edition" doc:"any under any cut, else the key of the cut the cell wants"`
	TargetID string `json:"targetId" format:"uuid"`
}

type mediaInfo struct {
	Container       string          `json:"container" doc:"ffprobe's format name, such as matroska,webm"`
	DurationSeconds int64           `json:"durationSeconds" minimum:"0" doc:"0 when the container states none"`
	Bitrate         int64           `json:"bitrate" minimum:"0" doc:"Bits per second; 0 when the container states none"`
	Video           videoInfo       `json:"video" doc:"Its first video stream that isn't cover art"`
	Audio           []audioTrack    `json:"audio" nullable:"false" doc:"In stream order"`
	Subtitles       []subtitleTrack `json:"subtitles" nullable:"false" doc:"In stream order"`
}

type videoInfo struct {
	Codec       string                    `json:"codec" doc:"ffprobe's codec name, such as hevc"`
	Profile     string                    `json:"profile" doc:"Such as Main 10; empty when ffprobe names none"`
	Width       int                       `json:"width" minimum:"0"`
	Height      int                       `json:"height" minimum:"0"`
	BitDepth    int                       `json:"bitDepth" minimum:"0" doc:"0 when the stream states none"`
	FrameRate   float64                   `json:"frameRate" minimum:"0" doc:"Frames per second to three decimals; 0 when the stream states none"`
	HDR         []api.Enum[hdrFormat]     `json:"hdr" nullable:"false" doc:"Every HDR format it carries, or SDR alone"`
	DolbyVision api.Nullable[dolbyVision] `json:"dolbyVision" doc:"Null when it carries no Dolby Vision"`
}

type dolbyVision struct {
	Profile       int `json:"profile" minimum:"0"`
	Level         int `json:"level" minimum:"0"`
	Compatibility int `json:"compatibility" minimum:"0" doc:"The base layer's signal: 1 HDR10, 2 SDR, 4 HLG, 6 Blu-ray HDR10, 0 none"`
}

type audioTrack struct {
	Codec    string  `json:"codec" doc:"ffprobe's codec name, such as eac3"`
	Profile  string  `json:"profile" doc:"Such as Dolby Digital Plus + Dolby Atmos; empty when ffprobe names none"`
	Atmos    bool    `json:"atmos"`
	Channels int     `json:"channels" minimum:"0"`
	Layout   string  `json:"layout" doc:"ffprobe's channel layout, such as 5.1(side); empty when it names none"`
	Language *string `json:"language" doc:"An ISO 639 code; null when the track names none"`
	Default  bool    `json:"default"`
	Title    string  `json:"title" doc:"Empty when the track has none"`
}

type subtitleTrack struct {
	Codec    string  `json:"codec" doc:"ffprobe's codec name, such as subrip"`
	Image    bool    `json:"image" doc:"A bitmap format, such as PGS"`
	Language *string `json:"language" doc:"An ISO 639 code; null when the track names none"`
	Forced   bool    `json:"forced"`
	Default  bool    `json:"default"`
}

type videoCodec struct{}

func (videoCodec) Values() []string { return wordsOf(release.VideoCodecs()) }

type hdrFormat struct{}

func (hdrFormat) Values() []string { return wordsOf(release.HDRs()) }

func wordsOf[S ~string](vs []S) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

func listedOf(f files.Listed) ListedFile {
	fills := make([]fill, len(f.Fills))
	for i, k := range f.Fills {
		fills[i] = fill{ItemID: k.Item.String(), Edition: string(k.Edition), TargetID: k.Target.String()}
	}
	out := ListedFile{
		ID: f.ID.String(), Path: f.Path, Folder: path.Dir(f.Path), Size: f.Size, Quality: f.Quality.String(),
		Cut: string(f.Cut), Label: f.Label, Kept: f.Kept(), Fills: fills, GoneAt: timeOrNull(f.GoneAt),
	}
	if c, known := release.VideoCodecOf(f.Media.Codec).Get(); known {
		out.Codec = api.Some(api.NewEnum[videoCodec](string(c)))
	}
	if f.Seen != nil {
		out.ModifiedAt = &f.Seen.Modified
	}
	if f.ProbeError != "" {
		out.ProbeError = &f.ProbeError
	}
	if f.Media.Version > 0 {
		out.Media = api.Some(mediaOf(f.Media))
	}
	return out
}

func mediaOf(r media.Record) mediaInfo {
	out := mediaInfo{
		Container: r.Container, DurationSeconds: int64(r.Duration / time.Second), Bitrate: r.Bitrate,
		Video: videoInfo{
			Codec: r.Codec, Profile: r.Profile, Width: r.Width, Height: r.Height, BitDepth: r.BitDepth,
			HDR: make([]api.Enum[hdrFormat], len(r.HDR)),
		},
		Audio:     make([]audioTrack, len(r.Audio)),
		Subtitles: make([]subtitleTrack, len(r.Subtitles)),
	}
	if r.FrameRate.Den > 0 {
		out.Video.FrameRate = math.Round(float64(r.FrameRate.Num)/float64(r.FrameRate.Den)*1000) / 1000
	}
	for i, h := range r.HDR {
		out.Video.HDR[i] = api.NewEnum[hdrFormat](string(h))
	}
	if dv := r.DolbyVision; dv != nil {
		out.Video.DolbyVision = api.Some(dolbyVision{Profile: dv.Profile, Level: dv.Level, Compatibility: dv.Compatibility})
	}
	for i, a := range r.Audio {
		out.Audio[i] = audioTrack{
			Codec: a.Codec, Profile: a.Profile, Atmos: a.Atmos, Channels: a.Channels, Layout: a.Layout,
			Language: languageOf(a.Language), Default: a.Default, Title: a.Title,
		}
	}
	for i, s := range r.Subtitles {
		out.Subtitles[i] = subtitleTrack{Codec: s.Codec, Image: s.Image, Language: languageOf(s.Language), Forced: s.Forced, Default: s.Default}
	}
	return out
}

func languageOf(l release.Language) *string {
	if l == "" {
		return nil
	}
	return new(string(l))
}

func timeOrNull(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return new(t.Time())
}

type fileList struct {
	Folder string       `json:"folder" doc:"The Work's folder under its root, the one its naming template gives while it has no file"`
	Files  []ListedFile `json:"files" nullable:"false" doc:"Filling files first, in Profile order of the first Target each fills, then Kept and gone files, oldest first"`
}

type fileListOutput struct{ Body fileList }

func (h handlers) listWorkFiles(ctx context.Context, by identity.Admin, in *cellsInput) (*fileListOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	l, err := open(h.lib).WorkFiles(ctx, by, works.WorkID{UUID: u})
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := fileList{Folder: l.Folder, Files: make([]ListedFile, len(l.Files))}
	for i, f := range l.Files {
		out.Files[i] = listedOf(f)
	}
	return &fileListOutput{Body: out}, nil
}
