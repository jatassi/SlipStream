package grabsapi

import "github.com/jatassi/slipstream/internal/release"

// releaseFacts is what a Release's title says about itself, as every list
// of releases draws it beside its title and size.
type releaseFacts struct {
	Quality  string `json:"quality,omitempty" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"The quality the title names; absent when it names none"`
	Revision string `json:"revision,omitempty" doc:"Such as v2 or v2 REPACK; absent for the original release"`
	HDR      string `json:"hdr,omitempty" doc:"The first HDR format the title names, such as Dolby Vision; absent when it names none"`
	Audio    string `json:"audio,omitempty" doc:"The first audio codec and channel layout the title names, such as DD+ 5.1; absent when it names neither"`
	Group    string `json:"group,omitempty" doc:"The Release group, such as FLUX; absent when the title names none"`
}

// factsOf is s on the wire. It leaves out s.Size, because each object that
// carries facts carries the Release's size beside them.
func factsOf(s release.Summary) releaseFacts {
	var out releaseFacts
	if q, ok := s.Quality.Get(); ok {
		out.Quality = q.String()
	}
	if r, ok := s.Revision.Get(); ok {
		out.Revision = r.String()
	}
	if h, ok := s.HDR.Get(); ok {
		out.HDR = h.Words()
	}
	out.Audio, _ = s.Audio.Get()
	out.Group, _ = s.Group.Get()
	return out
}
