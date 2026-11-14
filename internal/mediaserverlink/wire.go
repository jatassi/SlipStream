package mediaserverlink

import (
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// Times cross the wire as unix milliseconds, 0 for the zero time.
func msOf(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func timeOf(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func idOnWire(u storage.UUID) []byte {
	if u.IsZero() {
		return nil
	}
	return u.Bytes()
}

func uuidOf(b []byte) (storage.UUID, error) {
	if len(b) == 0 {
		return storage.UUID{}, nil
	}
	return storage.UUIDFromBytes(b)
}

func pathOf(s string) (link.ServerPath, error) {
	if s == "" {
		return link.ServerPath{}, nil
	}
	return link.ParsePath(s)
}

func pathsOnWire(ps []link.ServerPath) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}

func pathsOf(ss []string) ([]link.ServerPath, error) {
	out := make([]link.ServerPath, len(ss))
	for i, s := range ss {
		p, err := link.ParsePath(s)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

// AdvanceRequestOf is j as the Controller sends it.
func AdvanceRequestOf(j link.Job) *msv1.AdvanceRequest {
	return &msv1.AdvanceRequest{ServerId: idOnWire(j.Server.ID.UUID), SettingsVersion: j.Server.SettingsVersion, Job: JobOnWire(j)}
}

func JobOnWire(j link.Job) *msv1.Job {
	m := &msv1.Job{Step: string(j.Step), StepUnixMs: msOf(j.At)}
	switch w := j.Work.(type) {
	case link.Refresh:
		m.Work = &msv1.Job_Refresh{Refresh: &msv1.Refresh{Changes: changesOnWire(w.Changes)}}
	case link.Carry:
		m.Work = &msv1.Job_Carry{Carry: &msv1.Carry{Changes: changesOnWire(w.Changes)}}
	case link.Fresh:
		m.Work = &msv1.Job_Fresh{Fresh: &msv1.Fresh{Library: w.Library, Folder: w.Folder.String(),
			RequestedUnixMs: msOf(w.Requested), StartedUnixMs: msOf(w.Started)}}
	case link.Rescan:
		ids := make([][]byte, len(w.Changes))
		for i, id := range w.Changes {
			ids[i] = idOnWire(id.UUID)
		}
		m.Work = &msv1.Job_Rescan{Rescan: &msv1.Rescan{Libraries: w.Libraries, Changes: ids}}
	case link.Lookup:
		m.Work = &msv1.Job_Lookup{Lookup: &msv1.Lookup{Paths: pathsOnWire(w.Paths)}}
	case link.Index:
		m.Work = &msv1.Job_Index{Index: &msv1.Index{Library: w.Library, Cursor: w.Cursor}}
	case link.Libraries:
		m.Work = &msv1.Job_Libraries{Libraries: &msv1.Libraries{}}
	}
	return m
}

// JobOf parses m, a Job for server.
func JobOf(server link.ServerRef, m *msv1.Job) (link.Job, error) {
	step, err := link.ParseStep(m.GetStep())
	if err != nil {
		return link.Job{}, err
	}
	w, err := workOf(m)
	if err != nil {
		return link.Job{}, err
	}
	return link.Job{Server: server, Work: w, Step: step, At: timeOf(m.GetStepUnixMs())}, nil
}

func workOf(m *msv1.Job) (link.Work, error) {
	switch w := m.GetWork().(type) {
	case *msv1.Job_Refresh:
		cs, err := changesOf(w.Refresh.GetChanges())
		return link.Refresh{Changes: cs}, err
	case *msv1.Job_Carry:
		cs, err := changesOf(w.Carry.GetChanges())
		return link.Carry{Changes: cs}, err
	case *msv1.Job_Fresh:
		f, err := link.ParsePath(w.Fresh.GetFolder())
		return link.Fresh{Library: w.Fresh.GetLibrary(), Folder: f,
			Requested: timeOf(w.Fresh.GetRequestedUnixMs()), Started: timeOf(w.Fresh.GetStartedUnixMs())}, err
	case *msv1.Job_Rescan:
		ids := make([]link.ChangeID, len(w.Rescan.GetChanges()))
		for i, b := range w.Rescan.GetChanges() {
			u, err := uuidOf(b)
			if err != nil {
				return nil, err
			}
			ids[i] = link.ChangeID{UUID: u}
		}
		return link.Rescan{Libraries: w.Rescan.GetLibraries(), Changes: ids}, nil
	case *msv1.Job_Lookup:
		ps, err := pathsOf(w.Lookup.GetPaths())
		return link.Lookup{Paths: ps}, err
	case *msv1.Job_Index:
		return link.Index{Library: w.Index.GetLibrary(), Cursor: w.Index.GetCursor()}, nil
	case *msv1.Job_Libraries:
		return link.Libraries{}, nil
	}
	return nil, errors.New("a Media Server job names no work")
}

func changesOnWire(cs []link.JobChange) []*msv1.JobChange {
	out := make([]*msv1.JobChange, len(cs))
	for i, c := range cs {
		out[i] = &msv1.JobChange{Id: idOnWire(c.ID.UUID), Kind: string(c.Kind), Library: c.Library,
			Path: c.Path.String(), FromPath: c.From.String(), Snapshot: SnapshotOnWire(c.Snapshot)}
	}
	return out
}

func changesOf(ms []*msv1.JobChange) ([]link.JobChange, error) {
	out := make([]link.JobChange, len(ms))
	for i, m := range ms {
		c, err := changeOf(m)
		if err != nil {
			return nil, err
		}
		out[i] = c
	}
	return out, nil
}

func changeOf(m *msv1.JobChange) (link.JobChange, error) {
	id, err := uuidOf(m.GetId())
	if err != nil {
		return link.JobChange{}, err
	}
	kind, err := link.ParseChangeKind(m.GetKind())
	if err != nil {
		return link.JobChange{}, err
	}
	path, err := pathOf(m.GetPath())
	if err != nil {
		return link.JobChange{}, err
	}
	from, err := pathOf(m.GetFromPath())
	if err != nil {
		return link.JobChange{}, err
	}
	return link.JobChange{ID: link.ChangeID{UUID: id}, Kind: kind, Library: m.GetLibrary(), Path: path, From: from,
		Snapshot: SnapshotOf(m.GetSnapshot())}, nil
}

// SnapshotOnWire is nil for a nil s: no item was at the old path.
func SnapshotOnWire(s *link.Snapshot) *msv1.Snapshot {
	if s == nil {
		return nil
	}
	users := make([]*msv1.UserData, len(s.Users))
	for i, u := range s.Users {
		users[i] = &msv1.UserData{User: u.User, Played: u.Played, PlayCount: int32(min(u.PlayCount, 1<<31-1)), //nolint:gosec // clamped to int32
			PositionMs: u.Position.Milliseconds(), Favorite: u.Favorite, LastPlayedUnixMs: msOf(u.LastPlayed), Rating: u.Rating}
	}
	return &msv1.Snapshot{Item: itemOnWire(s.Item), Users: users, Collections: s.Collections}
}

func SnapshotOf(m *msv1.Snapshot) *link.Snapshot {
	if m == nil {
		return nil
	}
	users := make([]link.UserData, len(m.GetUsers()))
	for i, u := range m.GetUsers() {
		users[i] = link.UserData{User: u.GetUser(), Played: u.GetPlayed(), PlayCount: int(u.GetPlayCount()),
			Position: time.Duration(u.GetPositionMs()) * time.Millisecond, Favorite: u.GetFavorite(),
			LastPlayed: timeOf(u.GetLastPlayedUnixMs()), Rating: u.GetRating()}
	}
	return &link.Snapshot{Item: itemOf(m.GetItem()), Users: users, Collections: m.GetCollections()}
}

func itemOnWire(k link.ItemKey) *msv1.ItemKey { return &msv1.ItemKey{Item: k.Item, Top: k.Top} }

func itemOf(m *msv1.ItemKey) link.ItemKey { return link.ItemKey{Item: m.GetItem(), Top: m.GetTop()} }

func AdvancedOnWire(a link.Advanced) *msv1.AdvanceResponse {
	m := &msv1.AdvanceResponse{Job: JobOnWire(a.Job), Done: a.Done, WaitUnixMs: msOf(a.Wait), Libraries: LibrariesOnWire(a.Libraries)}
	for _, o := range a.Outcomes {
		lost := make([]string, len(o.Lost))
		for i, l := range o.Lost {
			lost[i] = string(l)
		}
		m.Outcomes = append(m.Outcomes, &msv1.Outcome{ChangeId: idOnWire(o.Change.UUID), Result: string(o.Result), Lost: lost})
	}
	for _, e := range a.Learned {
		m.Learned = append(m.Learned, &msv1.IndexEntry{Path: e.Path.String(), Library: e.Library, Item: itemOnWire(e.Item), Gone: e.Gone})
	}
	if a.Mapping != nil {
		m.Mapping = &msv1.MappingFact{StartedUnixMs: msOf(a.Mapping.Started), Scanned: a.Mapping.Scanned}
	}
	return m
}

// AdvancedOf parses m, a step's answer for a Job on server. A server's
// failure is a *link.ServerError.
func AdvancedOf(server link.ServerRef, m *msv1.AdvanceResponse) (link.Advanced, error) {
	if m.GetProblem() != nil {
		return link.Advanced{}, ServerErrorOf(m.GetProblem())
	}
	j, err := JobOf(server, m.GetJob())
	if err != nil {
		return link.Advanced{}, err
	}
	libs, err := LibrariesOf(m.GetLibraries())
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: j, Done: m.GetDone(), Wait: timeOf(m.GetWaitUnixMs()), Libraries: libs}
	for _, o := range m.GetOutcomes() {
		out, err := outcomeOf(o)
		if err != nil {
			return link.Advanced{}, err
		}
		a.Outcomes = append(a.Outcomes, out)
	}
	for _, e := range m.GetLearned() {
		p, err := link.ParsePath(e.GetPath())
		if err != nil {
			return link.Advanced{}, err
		}
		a.Learned = append(a.Learned, link.IndexEntry{Path: p, Library: e.GetLibrary(), Item: itemOf(e.GetItem()), Gone: e.GetGone()})
	}
	if f := m.GetMapping(); f != nil {
		a.Mapping = &link.MappingFact{Started: timeOf(f.GetStartedUnixMs()), Scanned: f.GetScanned()}
	}
	return a, nil
}

func outcomeOf(m *msv1.Outcome) (link.Outcome, error) {
	id, err := uuidOf(m.GetChangeId())
	if err != nil {
		return link.Outcome{}, err
	}
	result, err := link.ParseResult(m.GetResult())
	if err != nil {
		return link.Outcome{}, err
	}
	o := link.Outcome{Change: link.ChangeID{UUID: id}, Result: result}
	for _, s := range m.GetLost() {
		l, err := link.ParseLoss(s)
		if err != nil {
			return link.Outcome{}, err
		}
		o.Lost = append(o.Lost, l)
	}
	return o, nil
}

// ProblemOnWire splits a call's error for its response: a server's failure
// becomes the response's problem, and any other error stays the call's.
func ProblemOnWire(err error) (*msv1.Problem, error) {
	p, ok := link.ServerProblem(err)
	if !ok {
		return nil, err
	}
	return &msv1.Problem{Reason: string(p.Reason), Status: int32(p.Status), Found: p.Found, Detail: p.Detail}, nil //nolint:gosec // an HTTP status
}

// ServerErrorOf is m as a *link.ServerError, nil for no problem.
func ServerErrorOf(m *msv1.Problem) error {
	if m == nil {
		return nil
	}
	reason, err := link.ParseReason(m.GetReason())
	if err != nil {
		return fmt.Errorf("read a Media Server problem: %w", err)
	}
	return &link.ServerError{Problem: link.Problem{Reason: reason, Status: int(m.GetStatus()), Found: m.GetFound(), Detail: m.GetDetail()}}
}

func IdentityOnWire(id link.Identity) *msv1.Identity {
	return &msv1.Identity{Id: id.ID, Name: id.Name, Version: id.Version.String()}
}

func IdentityOf(m *msv1.Identity) (link.Identity, error) {
	v, err := link.ParseVersion(m.GetVersion())
	if err != nil {
		return link.Identity{}, err
	}
	return link.Identity{ID: m.GetId(), Name: m.GetName(), Version: v}, nil
}

func CredentialOnWire(c link.Credential) *msv1.Credential {
	return &msv1.Credential{Token: c.Token, ApiKey: c.APIKey}
}

func CredentialOf(m *msv1.Credential) link.Credential {
	return link.Credential{Token: m.GetToken(), APIKey: m.GetApiKey()}
}

func LibrariesOnWire(ls []link.Library) []*msv1.Library {
	var out []*msv1.Library
	for _, l := range ls {
		out = append(out, &msv1.Library{Key: l.Key, Name: l.Name, Folders: pathsOnWire(l.Folders)})
	}
	return out
}

func LibrariesOf(ms []*msv1.Library) ([]link.Library, error) {
	var out []link.Library
	for _, m := range ms {
		fs, err := pathsOf(m.GetFolders())
		if err != nil {
			return nil, err
		}
		out = append(out, link.Library{Key: m.GetKey(), Name: m.GetName(), Folders: fs})
	}
	return out, nil
}

// CheckedOnWire is a Check's answer, its problem split out of err.
func CheckedOnWire(c link.Checked, err error) (*msv1.CheckResponse, error) {
	if err != nil {
		p, err := ProblemOnWire(err)
		return &msv1.CheckResponse{Problem: p}, err
	}
	return &msv1.CheckResponse{Identity: IdentityOnWire(c.Identity), MonitorDelayMs: c.MonitorDelay.Milliseconds(),
		Libraries: LibrariesOnWire(c.Libraries)}, nil
}

func CheckedOf(m *msv1.CheckResponse) (link.Checked, error) {
	if m.GetProblem() != nil {
		return link.Checked{}, ServerErrorOf(m.GetProblem())
	}
	id, err := IdentityOf(m.GetIdentity())
	if err != nil {
		return link.Checked{}, err
	}
	libs, err := LibrariesOf(m.GetLibraries())
	if err != nil {
		return link.Checked{}, err
	}
	return link.Checked{Identity: id, MonitorDelay: time.Duration(m.GetMonitorDelayMs()) * time.Millisecond, Libraries: libs}, nil
}
