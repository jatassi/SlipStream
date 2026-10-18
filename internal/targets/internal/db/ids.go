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

type Sourcing string

const Download Sourcing = "download"

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// List is the kind of an Edition list: any cut, or named cuts in their
// own rows. A Work's NULL list is its Profile's.
type List string

const (
	AnyList  List = "any"
	CutsList List = "cuts"
)
