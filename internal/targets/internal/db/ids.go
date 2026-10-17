package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	TargetID  struct{ storage.UUID }
	ProfileID struct{ storage.UUID }
)

// Attribute is a fact a hard filter rules on.
type Attribute string

const (
	HDR        Attribute = "hdr"
	VideoCodec Attribute = "video_codec"
	AudioCodec Attribute = "audio_codec"
	Channels   Attribute = "channels"
	Language   Attribute = "language"
)

// Mode is how a hard filter rules. Any, the zero value, has no row.
type Mode string

const (
	Any  Mode = ""
	Only Mode = "only"
	Not  Mode = "not"
)
