package subtitles

import "example.com/app/internal/storing"

func (s *Subtitles) calls() storing.SidecarCalls { return s.cfg.Storage("node") }
