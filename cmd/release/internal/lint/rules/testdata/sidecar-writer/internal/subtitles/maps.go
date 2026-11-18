package subtitles

func (s *Subtitles) build() { s.cfg.Storage("node").List() }
