package searching

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"
)

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
	// signing holds a token while a sign-in runs, solving while a
	// Challenge solver clears a page, and deriving while a job reads the
	// derived values.
	signing, solving, deriving chan struct{}
	mu                         sync.Mutex
	session                    *Session
	// derived are the values a native driver read once for this revision.
	derived map[string]string
}

// slot is the Indexer's slot for job, holding the newer of the cached
// session and the one job brought.
func (ss *sessions) slot(job Job) *slot {
	ss.mu.Lock()
	sl := ss.by[job.Indexer]
	if sl == nil || sl.revision < job.Revision {
		sl = &slot{revision: job.Revision, signing: make(chan struct{}, 1), solving: make(chan struct{}, 1), deriving: make(chan struct{}, 1)}
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

// put holds s from now on.
func (sl *slot) put(s *Session) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.session = s
}

// keep has the session hold cookies, the same session base held with what
// a response set, unless another session took its place since. A rotated
// session counts as earned now.
func (sl *slot) keep(base *Session, cookies []Cookie, rotated bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.session == nil || !sl.session.Earned.Equal(base.Earned) {
		return
	}
	kept := *sl.session
	kept.Cookies = cookies
	if rotated {
		kept.Earned = time.Now()
	}
	sl.session = &kept
}

// cleared has the session send c's cookies, in place of any of the same
// name, domain and path, and c's User-Agent, as a session earned when c
// was. A slot with no session starts one.
func (sl *slot) cleared(c clearance) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	var s Session
	if sl.session != nil {
		s = *sl.session
	}
	s.Cookies, s.UserAgent, s.Earned = withCookies(s.Cookies, c.cookies), c.userAgent, c.at
	sl.session = &s
}

// SignIn runs fn, one at a time for the Indexer on this Node, and holds
// the session it returns. A caller that saw seen and finds a session
// earned after it once its turn comes uses that one, unless must. fn's
// requests go through in, whose jar holds the cookies the sign-in earns
// and any pasted ones.
func (s *Site) SignIn(ctx context.Context, seen *Session, must bool, fn func(ctx context.Context, in *SigningIn) (*Session, error)) error {
	sl := s.slot
	select {
	case sl.signing <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("wait for another sign-in: %w", ctx.Err())
	}
	defer func() { <-sl.signing }()
	if held := sl.held(); !must && held != nil && (seen == nil || held.Earned.After(seen.Earned)) {
		return nil
	}
	session, err := fn(ctx, &SigningIn{site: s, jar: s.signInJar()})
	if err != nil {
		return err
	}
	if session != nil {
		sl.put(session)
	}
	return nil
}

// SigningIn is one sign-in's requests and the cookies they earned.
type SigningIn struct {
	site *Site
	jar  *jar
}

// Do sends req as o says, with the sign-in's cookies.
func (in *SigningIn) Do(ctx context.Context, req *http.Request, o Send) (*Response, error) {
	return in.site.send(ctx, req, o, in.jar)
}

// Cookies are the cookies the sign-in holds now.
func (in *SigningIn) Cookies() []Cookie {
	cookies, _ := in.jar.held()
	return cookies
}

// UseCookie puts the pairs of a Cookie header in the sign-in's jar, as
// cookies of host only.
func (in *SigningIn) UseCookie(header, host string) error { return in.jar.add(header, host) }

// signInJar is a sign-in's jar: the pasted cookies, which every sign-in
// request sends beside its own.
func (s *Site) signInJar() *jar {
	if held := s.slot.held(); held != nil && held.Pasted {
		return newJar(held.Cookies)
	}
	return newJar(nil)
}

// Derived is what read returned for the Indexer's revision on this Node.
// The first job of a revision runs read, one job at a time, and an error
// leaves it for the next job to run again. The values stay in this Node's
// memory and never reach the Controller.
func (s *Site) Derived(ctx context.Context, read func() (map[string]string, error)) (map[string]string, error) {
	sl := s.slot
	select {
	case sl.deriving <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for another job's derived values: %w", ctx.Err())
	}
	defer func() { <-sl.deriving }()
	if sl.derived == nil {
		values, err := read()
		if err != nil {
			return nil, err
		}
		sl.derived = map[string]string{}
		maps.Copy(sl.derived, values)
	}
	return maps.Clone(sl.derived), nil
}
