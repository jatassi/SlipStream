// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdbits

import "github.com/jatassi/slipstream/internal/ports/port"

// SlipStream: Prowlarr's validator is the required flag on the
// credentials. The username is tagged key because HDBits refuses the pair
// with one status, so the stop names both.
type hdBitsSettings struct {
	Username          string `port:"username,key" label:"Username" help:"Your HDBits username."`
	port.PasskeyField `help:"Find your passkey on your HDBits User Details page."`
	Codecs            []hdBitsCodec  `port:"codecs" label:"Codecs" help:"Leave empty to search every codec."`
	Mediums           []hdBitsMedium `port:"mediums" label:"Mediums" help:"Leave empty to search every medium."`
	Origins           []hdBitsOrigin `port:"origins" label:"Origins" help:"Leave empty to search every origin."`
	FreeleechOnly     bool           `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	UseFilenames      bool           `port:"useFilenames" label:"Use file names" default:"true" help:"Use each torrent's file name as the Release title."`
}

type hdBitsCodec int

func (hdBitsCodec) Options() []port.Option {
	return []port.Option{
		{Value: "1", Label: "H.264"},
		{Value: "2", Label: "MPEG-2"},
		{Value: "3", Label: "VC-1"},
		{Value: "4", Label: "XviD"},
		{Value: "5", Label: "HEVC"},
	}
}

type hdBitsMedium int

const (
	hdBitsMediumBluray  hdBitsMedium = 1
	hdBitsMediumCapture hdBitsMedium = 4
	hdBitsMediumRemux   hdBitsMedium = 5
)

func (hdBitsMedium) Options() []port.Option {
	return []port.Option{
		{Value: "1", Label: "Blu-ray/HD DVD"},
		{Value: "3", Label: "Encode"},
		{Value: "4", Label: "Capture"},
		{Value: "5", Label: "Remux"},
		{Value: "6", Label: "WEB-DL"},
	}
}

type hdBitsOrigin int

func (hdBitsOrigin) Options() []port.Option {
	return []port.Option{
		{Value: "0", Label: "Undefined"},
		{Value: "1", Label: "Internal"},
	}
}
