package subtitles

func (s *Subtitles) remove() { s.cfg.Storage("node").List() }
