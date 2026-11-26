package mbkind

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// carry keeps watch data and collections across a new path, which both
// products make a new item for: snapshot every user's data and the
// item's collections, report the old paths Deleted and wait until they
// are gone, report the new paths Created and wait until they exist, then
// write the data back and compare what the server now holds.
var carry = steps[link.Carry]{
	{link.StepSnapshotted, snapshot},
	{link.StepOldRemoved, reportOld},
	{link.StepNewAdded, reportNew},
	{link.StepNewFound, findNew},
	{link.StepRestored, restore},
}

// wantAt is each change's path that at names, in the library that path is
// in: a Move's old path is in its FromLibrary.
func wantAt(cs []link.JobChange, at side) want {
	w := want{}
	for _, ch := range cs {
		w.add(at.library(ch), at.path(ch))
	}
	return w
}

// side is one end of a carry: its path and the library the path is in.
type side struct {
	path    func(link.JobChange) link.ServerPath
	library func(link.JobChange) string
}

var (
	from = side{func(ch link.JobChange) link.ServerPath { return ch.From }, link.JobChange.OldLibrary}
	to   = side{func(ch link.JobChange) link.ServerPath { return ch.Path }, func(ch link.JobChange) string { return ch.Library }}
)

// kept are the changes whose old path held an item.
func kept(cs []link.JobChange) []link.JobChange {
	return slices.DeleteFunc(slices.Clone(cs), func(ch link.JobChange) bool { return ch.Snapshot == nil })
}

func userData(user string, d *jellyfin.UserData) link.UserData {
	out := link.UserData{User: user}
	if d == nil {
		return out
	}
	out.Played, out.PlayCount, out.Favorite = d.Played, d.PlayCount, d.IsFavorite
	out.Position = time.Duration(d.PlaybackPositionTicks) * 100
	if d.LastPlayedDate != nil {
		out.LastPlayed = d.LastPlayedDate.UTC()
	}
	return out
}

func wire(d link.UserData) jellyfin.UserData {
	out := jellyfin.UserData{Played: d.Played, PlayCount: d.PlayCount, IsFavorite: d.Favorite, PlaybackPositionTicks: int64(d.Position / 100)}
	if !d.LastPlayed.IsZero() {
		t := d.LastPlayed
		out.LastPlayedDate = &t
	}
	return out
}

// view is every user's data on the items at the paths of w, and each
// item's collections.
type view struct {
	users       []jellyfin.User
	byUser      []found
	collections map[string][]string
	ids         map[string]string
}

func (c *conn) view(ctx context.Context, w want) (view, error) {
	us, err := c.users(ctx)
	if err != nil {
		return view{}, err
	}
	v := view{users: us, byUser: make([]found, len(us))}
	for i, u := range us {
		if v.byUser[i], err = c.find(ctx, u.ID, w); err != nil {
			return view{}, err
		}
	}
	v.collections, v.ids, err = c.collections(ctx, us[0].ID)
	return v, err
}

// snapshot reads each item at the old paths. A change whose old path
// holds no item keeps a nil snapshot and ends converted, sent as added.
func snapshot(ctx context.Context, c *conn, j link.Job, w link.Carry) (link.Advanced, error) {
	v, err := c.view(ctx, wantAt(w.Changes, from))
	if err != nil {
		return link.Advanced{}, err
	}
	w.Changes = slices.Clone(w.Changes)
	for i, ch := range w.Changes {
		it, ok := v.byUser[0].at(ch.From)
		if !ok {
			continue
		}
		s := &link.Snapshot{Item: key(it), Collections: v.collections[it.ID]}
		for u, user := range v.users {
			if mine, ok := v.byUser[u].at(ch.From); ok {
				s.Users = append(s.Users, userData(user.ID, mine.UserData))
			}
		}
		w.Changes[i].Snapshot = s
	}
	return link.Advanced{Job: reached(j, link.StepSnapshotted, w)}, nil
}

func reportPaths(ctx context.Context, c *conn, typ jellyfin.UpdateType, paths []link.ServerPath) error {
	us := make([]jellyfin.Update, len(paths))
	for i, p := range paths {
		us[i] = jellyfin.Update{Path: p.String(), UpdateType: typ}
	}
	return c.c.ReportUpdated(ctx, us)
}

func pathsOf(cs []link.JobChange, at func(link.JobChange) link.ServerPath) []link.ServerPath {
	out := make([]link.ServerPath, len(cs))
	for i, ch := range cs {
		out[i] = at(ch)
	}
	return out
}

func reportOld(ctx context.Context, c *conn, j link.Job, w link.Carry) (link.Advanced, error) {
	if old := kept(w.Changes); len(old) > 0 {
		if err := reportPaths(ctx, c, jellyfin.Deleted, pathsOf(old, from.path)); err != nil {
			return link.Advanced{}, err
		}
	}
	return waitFor(reached(j, link.StepOldRemoved, w), c.d.MonitorDelay), nil
}

// past is whether bound has passed since j reached its step, counted from
// the end of the server's monitor delay.
func (c *conn) past(j link.Job, bound time.Duration) bool {
	return time.Now().After(j.At.Add(c.d.MonitorDelay + bound))
}

// reportNew reports the new paths Created once the old items are gone,
// or Timings.MovedStep passed. A change with no old item ends here.
func reportNew(ctx context.Context, c *conn, j link.Job, w link.Carry) (link.Advanced, error) {
	old := kept(w.Changes)
	still := found{}
	if len(old) > 0 {
		us, err := c.users(ctx)
		if err != nil {
			return link.Advanced{}, err
		}
		if still, err = c.find(ctx, us[0].ID, wantAt(old, from)); err != nil {
			return link.Advanced{}, err
		}
	}
	if len(still) > 0 && !c.past(j, c.d.Timings.MovedStep) {
		return waitFor(j, c.d.Timings.StepPoll), nil
	}
	if err := reportPaths(ctx, c, jellyfin.Created, pathsOf(w.Changes, to.path)); err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{}
	for _, ch := range w.Changes {
		if ch.Snapshot == nil {
			a.Outcomes = append(a.Outcomes, link.Outcome{Change: ch.ID, Result: link.ResultConverted})
		}
	}
	for _, ch := range old {
		if _, there := still.at(ch.From); !there {
			a.Learned = append(a.Learned, link.IndexEntry{Path: ch.From, Library: ch.OldLibrary(), Gone: true})
		}
	}
	w.Changes = old
	a.Job, a.Done = reached(j, link.StepNewAdded, w), len(old) == 0
	if !a.Done {
		a.Wait = time.Now().Add(c.d.MonitorDelay)
	}
	return a, nil
}

func (v view) byUser0() found {
	if len(v.byUser) == 0 {
		return found{}
	}
	return v.byUser[0]
}

var lostItem = []link.Loss{link.LostWatchState, link.LostCollections}

// findNew waits until every new item exists. Past Timings.MovedConfirm a
// change whose new item never appeared ends, its watch data and
// collections lost.
func findNew(ctx context.Context, c *conn, j link.Job, w link.Carry) (link.Advanced, error) {
	v, err := c.view(ctx, wantAt(w.Changes, to))
	if err != nil {
		return link.Advanced{}, err
	}
	now := v.byUser0()
	missing := slices.ContainsFunc(w.Changes, func(ch link.JobChange) bool { _, ok := now.at(ch.Path); return !ok })
	if missing && !c.past(j, c.d.Timings.MovedConfirm) {
		return waitFor(j, c.d.Timings.StepPoll), nil
	}
	var a link.Advanced
	var left []link.JobChange
	for _, ch := range w.Changes {
		it, ok := now.at(ch.Path)
		if !ok {
			a.Outcomes = append(a.Outcomes, link.Outcome{Change: ch.ID, Result: link.ResultAccepted, Lost: lostItem})
			continue
		}
		a.Learned = append(a.Learned, link.IndexEntry{Path: ch.Path, Library: ch.Library, Item: key(it)})
		left = append(left, ch)
	}
	w.Changes = left
	a.Job, a.Done = reached(j, link.StepNewFound, w), len(left) == 0
	return a, nil
}

// restore writes every user's data and the collections back onto each
// new item, then reads them again: what the server didn't keep is lost.
func restore(ctx context.Context, c *conn, j link.Job, w link.Carry) (link.Advanced, error) {
	v, err := c.view(ctx, wantAt(w.Changes, to))
	if err != nil {
		return link.Advanced{}, err
	}
	for _, ch := range w.Changes {
		it, ok := v.byUser0().at(ch.Path)
		if !ok {
			continue
		}
		if err := c.write(ctx, v, it.ID, ch.Path, ch.Snapshot); err != nil {
			return link.Advanced{}, err
		}
	}
	after, err := c.view(ctx, wantAt(w.Changes, to))
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: reached(j, link.StepRestored, w), Done: true}
	for _, ch := range w.Changes {
		a.Outcomes = append(a.Outcomes, link.Outcome{Change: ch.ID, Result: link.ResultAccepted, Lost: after.lost(ch)})
	}
	return a, nil
}

// write puts the snapshot onto item at path, leaving out each user whose
// data there already matches, so a user who never had watch data never
// gets a change to it, which deleting a made user relies on.
func (c *conn) write(ctx context.Context, v view, item string, path link.ServerPath, s *link.Snapshot) error {
	for _, d := range s.Users {
		u := slices.IndexFunc(v.users, func(u jellyfin.User) bool { return u.ID == d.User })
		if u < 0 {
			continue
		}
		if mine, _ := v.byUser[u].at(path); holds(userData(d.User, mine.UserData), d) {
			continue
		}
		if err := c.setUserData(ctx, item, d); err != nil {
			return err
		}
	}
	for _, name := range s.Collections {
		if slices.Contains(v.collections[item], name) {
			continue
		}
		if id, ok := v.ids[name]; ok {
			if err := c.c.AddToCollection(ctx, id, []string{item}); err != nil {
				return err
			}
			continue
		}
		id, err := c.c.CreateCollection(ctx, name, []string{item})
		if err != nil {
			return err
		}
		v.ids[name] = id
	}
	return nil
}

func (c *conn) setUserData(ctx context.Context, item string, d link.UserData) error {
	if err := c.c.SetUserData(ctx, d.User, item, wire(d)); err != nil || !c.f.favoriteApart {
		return err
	}
	return c.c.SetFavorite(ctx, d.User, item, d.Favorite)
}

// holds is whether now is was, to the second of the last play.
func holds(now, was link.UserData) bool {
	return now.Played == was.Played && now.PlayCount == was.PlayCount && now.Favorite == was.Favorite && now.Position == was.Position &&
		now.LastPlayed.Truncate(time.Second).Equal(was.LastPlayed.Truncate(time.Second))
}

// lost is what the item at ch's new path doesn't hold of its snapshot.
func (v view) lost(ch link.JobChange) []link.Loss {
	it, ok := v.byUser0().at(ch.Path)
	if !ok {
		return lostItem
	}
	miss := map[link.Loss]bool{}
	for _, was := range ch.Snapshot.Users {
		u := slices.IndexFunc(v.users, func(u jellyfin.User) bool { return u.ID == was.User })
		if u < 0 {
			miss[link.LostWatchState] = true
			continue
		}
		mine, _ := v.byUser[u].at(ch.Path)
		now := userData(was.User, mine.UserData)
		miss[link.LostWatchState] = miss[link.LostWatchState] || now.Played != was.Played || now.Position != was.Position || now.Favorite != was.Favorite
		miss[link.LostPlayCount] = miss[link.LostPlayCount] || now.PlayCount != was.PlayCount
		miss[link.LostLastPlayed] = miss[link.LostLastPlayed] || !now.LastPlayed.Truncate(time.Second).Equal(was.LastPlayed.Truncate(time.Second))
	}
	for _, name := range ch.Snapshot.Collections {
		miss[link.LostCollections] = miss[link.LostCollections] || !slices.Contains(v.collections[it.ID], name)
	}
	var out []link.Loss
	for _, l := range link.Losses() {
		if miss[l] {
			out = append(out, l)
		}
	}
	return out
}
