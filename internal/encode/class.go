package encode

import (
	"slices"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// Class is what a proof covers of a source: the clip it proves with. A
// decode class is a source the clip maker can't make, decoded on the
// processor, so only its filter and encode chain is proven.
type Class string

const (
	ClassH264x8      Class = "h264_8"
	ClassHEVCx8      Class = "hevc_8"
	ClassHEVCx10     Class = "hevc_10"
	ClassAV1x8       Class = "av1_8"
	ClassAV1x10      Class = "av1_10"
	ClassHDR10       Class = "hdr10"
	ClassHLG         Class = "hlg"
	ClassDV5         Class = "dv5"
	ClassDV81        Class = "dv81"
	ClassDecodeSDR   Class = "decode_sdr"
	ClassDecodeHDR10 Class = "decode_hdr10"
)

// sdrClasses are the codecs and bit depths the clip maker encodes.
var sdrClasses = map[Codec]map[int]Class{
	H264: {8: ClassH264x8},
	HEVC: {8: ClassHEVCx8, 10: ClassHEVCx10},
	AV1:  {8: ClassAV1x8, 10: ClassAV1x10},
}

// ClassOf is the class a probed source proves with. HDR10+ proves as
// HDR10, Dolby Vision 7 and 8.x as 8.1, and a codec or bit depth the clip
// maker can't encode, such as VC-1, MPEG-2 or 10-bit H.264, decodes on the
// processor.
func ClassOf(r media.Record) Class {
	sdr, decodable := sdrClasses[Codec(r.Codec)][max(r.BitDepth, 8)]
	has := func(h release.HDR) bool { return slices.Contains(r.HDR, h) }
	switch {
	case !decodable && hdr(r.HDR):
		return ClassDecodeHDR10
	case !decodable:
		return ClassDecodeSDR
	case r.DolbyVision != nil && r.DolbyVision.Profile == 5:
		return ClassDV5
	case r.DolbyVision != nil && (r.DolbyVision.Profile == 7 || r.DolbyVision.Profile == 8):
		return ClassDV81
	case has("HDR10") || has("HDR10+"):
		return ClassHDR10
	case has("HLG"):
		return ClassHLG
	}
	return sdr
}

// hdr reports whether a stream carries any HDR format.
func hdr(formats []release.HDR) bool {
	return slices.ContainsFunc(formats, func(h release.HDR) bool { return h != "SDR" })
}
