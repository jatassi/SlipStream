package mediabrowser

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func (s *server) routes() fake.Routes {
	r := fake.Routes{
		"info":           {Pattern: "GET /System/Info/Public", Handler: s.guard(public, s.info)},
		"config":         {Pattern: "GET /System/Configuration", Handler: s.guard(elevated, s.config)},
		"set-config":     {Pattern: "POST /System/Configuration", Handler: s.guard(elevated, s.setConfig)},
		"authenticate":   {Pattern: "POST /Users/AuthenticateByName", Handler: s.guard(public, s.authenticate)},
		"logout":         {Pattern: "POST /Sessions/Logout", Handler: s.guard(signedIn, s.logout)},
		"users":          {Pattern: "GET /Users", Handler: s.guard(elevated, s.listUsers)},
		"create-user":    {Pattern: "POST /Users/New", Handler: s.guard(elevated, s.createUser)},
		"user":           {Pattern: "GET /Users/{id}", Handler: s.guard(elevated, s.withUser(s.getUser))},
		"update-user":    {Pattern: "POST /Users/{id}", Handler: s.guard(elevated, s.withUser(s.updateUser))},
		"user-policy":    {Pattern: "POST /Users/{id}/Policy", Handler: s.guard(elevated, s.withUser(s.setPolicy))},
		"delete-user":    {Pattern: "DELETE /Users/{id}", Handler: s.guard(elevated, s.withUser(s.deleteUser))},
		"keys":           {Pattern: "GET /Auth/Keys", Handler: s.guard(elevated, s.listKeys)},
		"create-key":     {Pattern: "POST /Auth/Keys", Handler: s.guard(elevated, s.createKey)},
		"revoke-key":     {Pattern: "DELETE /Auth/Keys/{key}", Handler: s.guard(elevated, s.revokeKey)},
		"updated":        {Pattern: "POST /Library/Media/Updated", Handler: s.guard(signedIn, s.updated)},
		"refresh":        {Pattern: "POST /Library/Refresh", Handler: s.guard(elevated, s.refreshAll)},
		"tasks":          {Pattern: "GET /ScheduledTasks", Handler: s.guard(elevated, s.tasks)},
		"task":           {Pattern: "GET /ScheduledTasks/{id}", Handler: s.guard(elevated, s.task)},
		"user-items":     {Pattern: "GET /Users/{user}/Items", Handler: s.guard(signedIn, s.userItems)},
		"user-item":      {Pattern: "GET /Users/{user}/Items/{id}", Handler: s.guard(signedIn, s.userItem)},
		"user-item-data": {Pattern: "POST /Users/{user}/Items/{id}/UserData", Handler: s.guard(signedIn, s.userItemData)},
		"collections":    {Pattern: "POST /Collections", Handler: s.guard(signedIn, s.createCollection)},
		"collect":        {Pattern: "POST /Collections/{id}/Items", Handler: s.guard(signedIn, s.addToCollection)},
		"sessions":       {Pattern: "GET /Sessions", Handler: s.guard(signedIn, s.sessions)},
		"playing":        {Pattern: "POST /Sessions/Playing", Handler: s.guard(signedIn, s.playing)},
	}
	if s.p.Flavor == Jellyfin {
		r["libraries"] = fake.Route{Pattern: "GET /Library/VirtualFolders", Handler: s.guard(elevated, s.libraries)}
		r["add-path"] = fake.Route{Pattern: "POST /Library/VirtualFolders/Paths", Handler: s.guard(elevated, s.addPath)}
		r["items"] = fake.Route{Pattern: "GET /Items", Handler: s.guard(signedIn, s.items)}
		r["item"] = fake.Route{Pattern: "GET /Items/{id}", Handler: s.guard(signedIn, s.item)}
		r["user-data"] = fake.Route{Pattern: "GET /UserItems/{id}/UserData", Handler: s.guard(signedIn, s.getUserData)}
		r["set-user-data"] = fake.Route{Pattern: "POST /UserItems/{id}/UserData", Handler: s.guard(signedIn, s.setUserData)}
		r["played"] = fake.Route{Pattern: "POST /UserPlayedItems/{id}", Handler: s.guard(signedIn, s.markPlayed)}
		r["socket"] = fake.Route{Pattern: "GET /socket", Handler: s.socket}
		r["network"] = fake.Route{Pattern: "GET /System/Configuration/network", Handler: s.guard(elevated, s.network)}
		r["user-password"] = fake.Route{Pattern: "POST /Users/Password", Handler: s.guard(elevated, s.setPassword)}
	} else {
		r["libraries"] = fake.Route{Pattern: "GET /Library/VirtualFolders/Query", Handler: s.guard(signedIn, s.libraries)}
		r["item-refresh"] = fake.Route{Pattern: "POST /Items/{id}/Refresh", Handler: s.guard(elevated, s.refreshItem)}
		r["played"] = fake.Route{Pattern: "POST /Users/{user}/PlayedItems/{id}", Handler: s.guard(signedIn, s.markPlayed)}
		r["favorite"] = fake.Route{Pattern: "POST /Users/{user}/FavoriteItems/{id}", Handler: s.guard(signedIn, s.favorite(true))}
		r["unfavorite"] = fake.Route{Pattern: "DELETE /Users/{user}/FavoriteItems/{id}", Handler: s.guard(signedIn, s.favorite(false))}
		r["socket"] = fake.Route{Pattern: "GET /embywebsocket", Handler: s.socket}
		r["system-info"] = fake.Route{Pattern: "GET /System/Info", Handler: s.guard(elevated, s.systemInfo)}
		r["user-password"] = fake.Route{Pattern: "POST /Users/{id}/Password", Handler: s.guard(elevated, s.withUser(s.setEmbyPassword))}
	}
	for name, route := range r {
		route.Handler = s.failing(name, route.Handler)
		r[name] = route
	}
	return r
}

func writeJSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(body)
}

func (s *server) info(w http.ResponseWriter, _ *http.Request, _ caller) {
	writeJSON(w, map[string]any{
		"LocalAddress": "http://127.0.0.1:8096", "ServerName": s.srv.Name, "Version": s.srv.Version,
		"ProductName": s.p.ProductName, "OperatingSystem": "", "Id": s.srv.ID, "StartupWizardCompleted": true,
	})
}

func (s *server) network(w http.ResponseWriter, _ *http.Request, _ caller) {
	writeJSON(w, map[string]any{
		"EnableRemoteAccess": s.srv.Remote, "EnableIPv6": s.srv.IPv6, "EnableIPv4": true,
		"PublishedServerUriBySubnet": append([]string{}, s.srv.Published...), "InternalHttpPort": 8096,
	})
}

func (s *server) systemInfo(w http.ResponseWriter, _ *http.Request, _ caller) {
	wan, remote := "", []string{}
	if len(s.srv.Published) > 0 {
		wan, remote = s.srv.Published[0], s.srv.Published[1:]
	}
	writeJSON(w, map[string]any{
		"LocalAddress": "http://127.0.0.1:8096", "WanAddress": wan, "RemoteAddresses": remote, "ServerName": s.srv.Name,
		"Version": s.srv.Version, "Id": s.srv.ID, "HttpServerPortNumber": 8096,
	})
}

// delayField is the configuration's monitor delay in seconds: Jellyfin's
// LibraryMonitorDelay and Emby's LibraryMonitorDelaySeconds.
func (s *server) delayField() string {
	if s.p.Flavor == Emby {
		return "LibraryMonitorDelaySeconds"
	}
	return "LibraryMonitorDelay"
}

func (s *server) config(w http.ResponseWriter, _ *http.Request, _ caller) {
	writeJSON(w, map[string]any{
		s.delayField(): int(s.core.MonitorDelay() / time.Second), "EnableLegacyAuthorization": false,
		"LibraryScanFanoutConcurrency": 0, "EnableFolderView": false,
	})
}

func (s *server) setConfig(w http.ResponseWriter, r *http.Request, _ caller) {
	var in map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if raw, ok := in[s.delayField()]; ok {
		var seconds int
		if err := json.Unmarshal(raw, &seconds); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.core.SetMonitorDelay(time.Duration(seconds) * time.Second)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) libraries(w http.ResponseWriter, _ *http.Request, _ caller) {
	out := []map[string]any{}
	for _, l := range s.core.Libraries() {
		locations := make([]string, len(l.Folders))
		infos := make([]map[string]string, len(l.Folders))
		for i, f := range l.Folders {
			locations[i], infos[i] = f.String(), map[string]string{"Path": f.String()}
		}
		v := map[string]any{
			"Name": l.Name, "Locations": locations, "CollectionType": l.Type, "ItemId": l.Key,
			"LibraryOptions": map[string]any{"Enabled": true, "EnableRealtimeMonitor": false, "PathInfos": infos},
		}
		switch {
		case s.p.Flavor == Emby:
			v["Id"], v["Guid"] = l.Key, libraryGUID(l.Key)
		case l.Scan.Running:
			v["RefreshStatus"], v["RefreshProgress"] = "Active", 0
		default:
			v["RefreshStatus"] = "Idle"
		}
		out = append(out, v)
	}
	if s.p.Flavor == Emby {
		writeJSON(w, map[string]any{"Items": out, "TotalRecordCount": len(out)})
		return
	}
	writeJSON(w, out)
}

// addPath is Jellyfin's POST /Library/VirtualFolders/Paths, which takes
// the library by Name and needs the folder to exist. It drops pending
// reports as a full scan does.
func (s *server) addPath(w http.ResponseWriter, r *http.Request, _ caller) {
	var in struct {
		Name     string
		Path     string
		PathInfo struct{ Path string }
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	raw := either(in.PathInfo.Path, in.Path)
	p, err := link.ParsePath(raw)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := s.core.AddFolder(libraryKey(in.Name), p); errors.Is(err, mediaserver.ErrNoLibrary) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func either(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// updated is POST /Library/Media/Updated, 204 at once, each path scanned
// after the monitor delay.
func (s *server) updated(w http.ResponseWriter, r *http.Request, _ caller) {
	var in struct {
		Updates []struct{ Path, UpdateType string }
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rec := Report{Updates: []ReportedPath{}}
	var paths []link.ServerPath
	for _, u := range in.Updates {
		p, err := link.ParsePath(u.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rec.Updates = append(rec.Updates, ReportedPath{Path: p.String(), UpdateType: u.UpdateType})
		paths = append(paths, p)
	}
	s.reports.Record(r.Context(), rec)
	s.core.Report(paths)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) refreshAll(w http.ResponseWriter, r *http.Request, _ caller) {
	s.scans.Record(r.Context(), Scan{})
	s.core.RefreshAll()
	w.WriteHeader(http.StatusNoContent)
}

// refreshItem is Emby's item refresh. On a library's item with
// Recursive it scans that library.
func (s *server) refreshItem(w http.ResponseWriter, r *http.Request, _ caller) {
	key := r.PathValue("id")
	if _, ok := s.core.Item(key); ok {
		s.core.RefreshItem(key)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !strings.EqualFold(r.URL.Query().Get("Recursive"), "true") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.scans.Record(r.Context(), Scan{Library: key})
	if _, err := s.core.RefreshLibrary(key); errors.Is(err, mediaserver.ErrNoLibrary) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) renderTask() map[string]any {
	st := s.core.TaskState()
	t := map[string]any{
		"Name": s.p.TaskName, "State": "Idle", "Id": s.p.TaskID, "Key": "RefreshLibrary", "Triggers": []any{},
		"Description": "Scans your media library for new files and refreshes metadata.", "Category": "Library", "IsHidden": false,
	}
	if st.Running {
		t["State"], t["CurrentProgressPercentage"] = "Running", 0
	}
	if !st.Ended.IsZero() {
		t["LastExecutionResult"] = map[string]any{
			"StartTimeUtc": st.Started.UTC(), "EndTimeUtc": st.Ended.UTC(), "Status": "Completed",
			"Name": s.p.TaskName, "Key": "RefreshLibrary", "Id": s.p.TaskID,
		}
	}
	return t
}

func (s *server) tasks(w http.ResponseWriter, _ *http.Request, _ caller) {
	writeJSON(w, []map[string]any{s.renderTask()})
}

func (s *server) task(w http.ResponseWriter, r *http.Request, _ caller) {
	if r.PathValue("id") != s.p.TaskID {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, s.renderTask())
}

// asUser is the user a call reads data as: the userId it names, else the
// signed-in user, else the server's owner for a key.
func (s *server) asUser(c caller, uid string) (user, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	uid = either(uid, c.user)
	if uid == "" {
		return s.owner(), true
	}
	u, ok := s.userAt(uid)
	if !ok {
		return user{}, false
	}
	return *u, true
}

func (s *server) items(w http.ResponseWriter, r *http.Request, c caller) {
	s.listItems(w, r, c, r.URL.Query().Get("userId"))
}

func (s *server) userItems(w http.ResponseWriter, r *http.Request, c caller) {
	s.listItems(w, r, c, r.PathValue("user"))
}

// listItems is GET /Items: by ParentId (a library or a collection),
// IncludeItemTypes, and on Emby a Path, a file's exact path or the folder
// an item is in. Emby 4.10 answered one item for a film's folder (online
// run 2026-11-14); which item it was is unrecorded, and the fake answers
// the film. Jellyfin has no Path filter and lists alternate versions in
// the primary item, which alone a listing names.
func (s *server) listItems(w http.ResponseWriter, r *http.Request, c caller, uid string) {
	u, ok := s.asUser(c, uid)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	q := r.URL.Query()
	types := splitTypes(q.Get("IncludeItemTypes"))
	parent := q.Get("ParentId")
	var out []map[string]any
	if slices.Contains(types, "BoxSet") && parent == "" {
		for _, name := range s.core.CollectionNames() {
			out = append(out, s.renderCollection(name))
		}
	}
	for _, it := range s.allItems() {
		if !s.sees(u, it.Library) || !s.picked(it, parent, types, q) {
			continue
		}
		out = append(out, s.renderItem(it, u.id))
	}
	total := len(out)
	start, _ := strconv.Atoi(q.Get("StartIndex"))
	start = min(start, total)
	end := total
	if limit, err := strconv.Atoi(q.Get("Limit")); err == nil {
		end = min(start+limit, total)
	}
	writeJSON(w, map[string]any{"Items": nonNil(out[start:end]), "TotalRecordCount": total, "StartIndex": start})
}

func nonNil(v []map[string]any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	return v
}

func splitTypes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func (s *server) allItems() []mediaserver.Item {
	var out []mediaserver.Item
	for _, l := range s.core.Libraries() {
		page, _ := s.core.Items(l.Key, 0, 1<<30)
		out = append(out, page...)
	}
	return out
}

func (s *server) picked(it mediaserver.Item, parent string, types []string, q map[string][]string) bool {
	if parent != "" && parent != it.Library && !slices.Contains(s.core.Members(collectionOf(s, parent)), it.Key) {
		return false
	}
	if len(types) > 0 && !slices.Contains(types, itemType(it)) {
		return false
	}
	if s.p.Flavor == Emby {
		if paths, ok := q["Path"]; ok {
			return it.Folder.String() == paths[0] || slices.ContainsFunc(it.Parts, func(p mediaserver.Part) bool { return p.File.String() == paths[0] })
		}
	}
	return true
}

// collectionOf is the name of the collection with id, empty for none.
func collectionOf(s *server, cid string) string {
	for _, name := range s.core.CollectionNames() {
		if collectionID(name) == cid {
			return name
		}
	}
	return ""
}

func itemType(it mediaserver.Item) string {
	if it.Show != "" {
		return "Episode"
	}
	return "Movie"
}

func (s *server) renderCollection(name string) map[string]any {
	return map[string]any{"Name": name, "ServerId": s.srv.ID, "Id": collectionID(name), "IsFolder": true, "Type": "BoxSet"}
}

func ticks(d time.Duration) int64 { return int64(d / 100) }

func (s *server) renderUserData(uid string, it mediaserver.Item) map[string]any {
	d := s.core.UserData(uid, it.Key)
	out := map[string]any{
		"PlaybackPositionTicks": ticks(d.Position), "PlayCount": d.PlayCount, "IsFavorite": d.Favorite,
		"Played": d.Played, "Key": it.Key, "ItemId": it.Key,
	}
	if !d.LastPlayed.IsZero() {
		out["LastPlayedDate"] = d.LastPlayed.UTC()
	}
	return out
}

func providerIDs(it mediaserver.Item) map[string]string {
	out := map[string]string{}
	if name, value, ok := strings.Cut(it.Provider, "-"); ok {
		out[strings.ToUpper(name[:1])+name[1:]] = value
	}
	return out
}

func (s *server) renderItem(it mediaserver.Item, uid string) map[string]any {
	return s.renderAs(it, s.sources(it), uid)
}

// renderAs is it listing srcs. On Jellyfin the first source is the item
// it renders as: the primary, or an alternate version's own item.
func (s *server) renderAs(it mediaserver.Item, srcs []source, uid string) map[string]any {
	var sources []map[string]any
	key, path := it.Key, ""
	for _, src := range srcs {
		file := src.part.File.String()
		m := map[string]any{
			"Protocol": "File", "Id": src.id, "Path": file, "Type": "Default",
			"Container": strings.TrimPrefix(filepath.Ext(file), "."), "Size": src.part.Size,
			"Name":         strings.TrimSuffix(filepath.Base(filepath.FromSlash(strings.ReplaceAll(file, `\`, "/"))), filepath.Ext(file)),
			"MediaStreams": nonNil(renderStreams(src.part)),
		}
		if s.p.Flavor == Emby {
			m["ItemId"] = src.item.Key
		}
		if path == "" && src.item.Key == it.Key {
			path = file
			if s.p.Flavor == Jellyfin {
				key = src.id
			}
		}
		sources = append(sources, m)
	}
	m := map[string]any{
		"Name": it.Title, "ServerId": s.srv.ID, "Id": key, "Path": path, "Type": itemType(it), "IsFolder": false,
		"MediaType": "Video", "LocationType": "FileSystem", "VideoType": "VideoFile", "ProviderIds": providerIDs(it),
		"MediaSources": nonNil(sources), "UserData": s.renderUserData(uid, it), "ImageTags": map[string]any{},
	}
	if it.Year > 0 {
		m["ProductionYear"] = it.Year
	}
	if it.Show != "" {
		m["SeriesId"], m["SeriesName"], m["ParentIndexNumber"], m["IndexNumber"] = it.ShowKey, it.Show, it.Season, it.Episode
	}
	if s.p.Flavor == Emby {
		m["PresentationUniqueKey"] = it.GUID + "/" + it.Edition
	}
	return m
}

func (s *server) item(w http.ResponseWriter, r *http.Request, c caller) {
	s.oneItem(w, r, c, r.URL.Query().Get("userId"))
}

func (s *server) userItem(w http.ResponseWriter, r *http.Request, c caller) {
	s.oneItem(w, r, c, r.PathValue("user"))
}

func (s *server) oneItem(w http.ResponseWriter, r *http.Request, c caller, uid string) {
	u, ok := s.asUser(c, uid)
	var srcs []source
	it, found := s.core.Item(r.PathValue("id"))
	if found {
		srcs = s.sources(it)
	} else {
		it, srcs, found = s.alternate(r.PathValue("id"))
	}
	if !ok || !found || !s.sees(u, it.Library) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, s.renderAs(it, srcs, u.id))
}

func (s *server) getUserData(w http.ResponseWriter, r *http.Request, c caller) {
	u, ok := s.asUser(c, r.URL.Query().Get("userId"))
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, s.renderUserData(u.id, it))
}

func (s *server) setUserData(w http.ResponseWriter, r *http.Request, c caller) {
	s.writeUserData(w, r, c, r.URL.Query().Get("userId"))
}

func (s *server) userItemData(w http.ResponseWriter, r *http.Request, c caller) {
	s.writeUserData(w, r, c, r.PathValue("user"))
}

// writeUserData sets the fields the body names on the user's data. Emby
// 4.10 keeps the favorite it had (online run 2026-11-14).
func (s *server) writeUserData(w http.ResponseWriter, r *http.Request, c caller, uid string) {
	u, ok := s.asUser(c, uid)
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var in struct {
		PlaybackPositionTicks *int64
		PlayCount             *int
		IsFavorite            *bool
		Played                *bool
		LastPlayedDate        *time.Time
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d := s.core.UserData(u.id, it.Key)
	if in.PlaybackPositionTicks != nil {
		d.Position = time.Duration(*in.PlaybackPositionTicks) * 100
	}
	if in.PlayCount != nil {
		d.PlayCount = *in.PlayCount
	}
	if in.IsFavorite != nil && s.p.Flavor != Emby {
		d.Favorite = *in.IsFavorite
	}
	if in.Played != nil {
		d.Played = *in.Played
	}
	if in.LastPlayedDate != nil {
		d.LastPlayed = *in.LastPlayedDate
	}
	s.core.SetUserData(u.id, it.Key, d)
	writeJSON(w, s.renderUserData(u.id, it))
}

// markPlayed marks the item played. Jellyfin counts a play and stamps
// it; Emby sets neither (#101).
func (s *server) markPlayed(w http.ResponseWriter, r *http.Request, c caller) {
	uid := r.URL.Query().Get("userId")
	if s.p.Flavor == Emby {
		uid = r.PathValue("user")
	}
	u, ok := s.asUser(c, uid)
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	d := s.core.UserData(u.id, it.Key)
	d.Played = true
	if s.p.Flavor != Emby {
		d.PlayCount, d.LastPlayed = d.PlayCount+1, time.Now()
	}
	s.core.SetUserData(u.id, it.Key, d)
	writeJSON(w, s.renderUserData(u.id, it))
}

// favorite is Emby's FavoriteItems: POST marks the item a favorite of the
// user and DELETE unmarks it.
func (s *server) favorite(on bool) func(http.ResponseWriter, *http.Request, caller) {
	return func(w http.ResponseWriter, r *http.Request, c caller) {
		u, ok := s.asUser(c, r.PathValue("user"))
		it, found := s.core.Item(r.PathValue("id"))
		if !ok || !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		d := s.core.UserData(u.id, it.Key)
		d.Favorite = on
		s.core.SetUserData(u.id, it.Key, d)
		writeJSON(w, s.renderUserData(u.id, it))
	}
}

func idsOf(r *http.Request) []string {
	var out []string
	for _, v := range strings.Split(r.URL.Query().Get("Ids"), ",") {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// createCollection makes a collection holding Ids and answers its Id.
func (s *server) createCollection(w http.ResponseWriter, r *http.Request, _ caller) {
	name := r.URL.Query().Get("Name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.core.AddToCollection(name, idsOf(r)...)
	writeJSON(w, map[string]string{"Id": collectionID(name)})
}

// addToCollection puts Ids in the collection. An item already in it
// stays in it once, the answer the conformance case
// CollectionAddIsIdempotent pins.
func (s *server) addToCollection(w http.ResponseWriter, r *http.Request, _ caller) {
	name := collectionOf(s, r.PathValue("id"))
	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.core.AddToCollection(name, idsOf(r)...)
	w.WriteHeader(http.StatusNoContent)
}
