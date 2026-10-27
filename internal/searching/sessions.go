package searching

import "sync"

// sessions are the newest session of each Indexer on this Node: the one a
// job brought or one a job earned, whichever was earned later. An edit may
// change the credentials, so a newer revision starts with what its job
// brings.
type sessions struct {
	mu sync.Mutex
	by map[string]*slot
}

type slot struct {
	revision int64
	mu       sync.Mutex
	session  *Session
}

// slot is the Indexer's slot for job, holding the newer of the cached
// session and the one job brought.
func (ss *sessions) slot(job Job) *slot {
	ss.mu.Lock()
	sl := ss.by[job.Indexer]
	if sl == nil || sl.revision < job.Revision {
		sl = &slot{revision: job.Revision}
		ss.by[job.Indexer] = sl
	}
	ss.mu.Unlock()
	sl.offer(job.Session)
	return sl
}

// offer keeps s when it was earned after the session held.
func (sl *slot) offer(s *Session) {
	if s == nil {
		return
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.session == nil || s.Earned.After(sl.session.Earned) {
		sl.session = s
	}
}

// newer is the session held when it was earned after than, nil otherwise:
// what an Answer reports as changed.
func (sl *slot) newer(than *Session) *Session {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.session == nil || (than != nil && !sl.session.Earned.After(than.Earned)) {
		return nil
	}
	return sl.session
}

// held is the session the slot holds, nil for none.
func (sl *slot) held() *Session {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.session
}
