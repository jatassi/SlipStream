package subtitles

import st "example.com/app/internal/storing"

func (s *Subtitles) leave() {
	var c st.SidecarCalls = s.cfg.Storage("node")
	c.List()
}
