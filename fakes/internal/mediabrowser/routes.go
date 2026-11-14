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
		"users":          {Pattern: "GET /Users", Handler: s.guard(elevated, s.users)},
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
	} else {
		r["libraries"] = fake.Route{Pattern: "GET /Library/VirtualFolders/Query", Handler: s.guard(signedIn, s.libraries)}
		r["item-refresh"] = fake.Route{Pattern: "POST /Items/{id}/Refresh", Handler: s.guard(elevated, s.refreshItem)}
		r["played"] = fake.Route{Pattern: "POST /Users/{user}/PlayedItems/{id}", Handler: s.guard(signedIn, s.markPlayed)}
		r["socket"] = fake.Route{Pattern: "GET /embywebsocket", Handler: s.socket}
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
			"Name": l.Name, "Locations": locations, "CollectionType": l.Type, "ItemId": l.Key, "RefreshStatus": "Idle",
			"LibraryOptions": map[string]any{"Enabled": true, "EnableRealtimeMonitor": false, "PathInfos": infos},
		}
		if l.Scan.Running {
			v["RefreshStatus"], v["RefreshProgress"] = "Active", 0
		}
		if s.p.Flavor == Emby {
			v["Id"], v["Guid"] = l.Key, l.Key
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
	var updates []mediaserver.Update
	for _, u := range in.Updates {
		p, err := link.ParsePath(u.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rec.Updates = append(rec.Updates, ReportedPath{Path: p.String(), UpdateType: u.UpdateType})
		updates = append(updates, mediaserver.Update{Path: p, Deleted: u.UpdateType == "Deleted"})
	}
	s.reports.Record(r.Context(), rec)
	s.core.Report(updates)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) refreshAll(w http.ResponseWriter, r *http.Request, _ caller) {
	s.scans.Record(r.Context(), Scan{})
	s.core.RefreshAll()
	w.WriteHeader(http.StatusNoContent)
}

// refreshItem is Emby's item refresh. On a library's item with
// Recursive it scans that library, shown in its RefreshStatus.
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
func (s *server) asUser(c caller, uid string) (string, bool) {
	if uid != "" {
		u, ok := s.userByID(uid)
		return u.Name, ok
	}
	if c.user != "" {
		return c.user, true
	}
	return s.owner(), true
}

func (s *server) items(w http.ResponseWriter, r *http.Request, c caller) {
	s.listItems(w, r, c, r.URL.Query().Get("userId"))
}

func (s *server) userItems(w http.ResponseWriter, r *http.Request, c caller) {
	s.listItems(w, r, c, r.PathValue("user"))
}

// listItems is GET /Items: by ParentId (a library or a collection),
// IncludeItemTypes, and on Emby an exact Path. Jellyfin has no Path
// filter and lists alternate versions only in the primary item's
// MediaSources.
func (s *server) listItems(w http.ResponseWriter, r *http.Request, c caller, uid string) {
	user, ok := s.asUser(c, uid)
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
		if !s.picked(it, parent, types, q) {
			continue
		}
		out = append(out, s.renderItem(it, user))
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
			return slices.ContainsFunc(it.Parts, func(p mediaserver.Part) bool { return p.File.String() == paths[0] })
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

func (s *server) renderUserData(user string, it mediaserver.Item) map[string]any {
	d := s.core.UserData(user, it.Key)
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

func (s *server) renderItem(it mediaserver.Item, user string) map[string]any {
	var sources []map[string]any
	for _, p := range it.Parts {
		if p.Missing {
			continue
		}
		file := p.File.String()
		var streams []map[string]any
		for _, sc := range p.Sidecars {
			streams = append(streams, map[string]any{
				"Type": "Subtitle", "Codec": strings.TrimPrefix(filepath.Ext(sc.String()), "."), "IsExternal": true, "Path": sc.String(),
			})
		}
		sources = append(sources, map[string]any{
			"Protocol": "File", "Id": id("source " + file), "Path": file, "Type": "Default",
			"Container": strings.TrimPrefix(filepath.Ext(file), "."), "Size": p.Size,
			"Name":         strings.TrimSuffix(filepath.Base(filepath.FromSlash(strings.ReplaceAll(file, `\`, "/"))), filepath.Ext(file)),
			"MediaStreams": nonNil(streams),
		})
	}
	path := ""
	if len(sources) > 0 {
		path, _ = sources[0]["Path"].(string)
	}
	m := map[string]any{
		"Name": it.Title, "ServerId": s.srv.ID, "Id": it.Key, "Path": path, "Type": itemType(it), "IsFolder": false,
		"MediaType": "Video", "LocationType": "FileSystem", "VideoType": "VideoFile", "ProviderIds": providerIDs(it),
		"MediaSources": nonNil(sources), "UserData": s.renderUserData(user, it), "ImageTags": map[string]any{},
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
	user, ok := s.asUser(c, uid)
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, s.renderItem(it, user))
}

func (s *server) getUserData(w http.ResponseWriter, r *http.Request, c caller) {
	user, ok := s.asUser(c, r.URL.Query().Get("userId"))
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, s.renderUserData(user, it))
}

func (s *server) setUserData(w http.ResponseWriter, r *http.Request, c caller) {
	s.writeUserData(w, r, c, r.URL.Query().Get("userId"))
}

func (s *server) userItemData(w http.ResponseWriter, r *http.Request, c caller) {
	s.writeUserData(w, r, c, r.PathValue("user"))
}

// writeUserData sets the fields the body names on the user's data. Emby
// keeps the play count and last played date it had (#101).
func (s *server) writeUserData(w http.ResponseWriter, r *http.Request, c caller, uid string) {
	user, ok := s.asUser(c, uid)
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
	d := s.core.UserData(user, it.Key)
	if in.PlaybackPositionTicks != nil {
		d.Position = time.Duration(*in.PlaybackPositionTicks) * 100
	}
	if in.PlayCount != nil {
		d.PlayCount = *in.PlayCount
	}
	if in.IsFavorite != nil {
		d.Favorite = *in.IsFavorite
	}
	if in.Played != nil {
		d.Played = *in.Played
	}
	if in.LastPlayedDate != nil {
		d.LastPlayed = *in.LastPlayedDate
	}
	s.core.SetUserData(user, it.Key, d)
	writeJSON(w, s.renderUserData(user, it))
}

// markPlayed marks the item played. Jellyfin counts a play and stamps
// it; Emby sets neither (#101).
func (s *server) markPlayed(w http.ResponseWriter, r *http.Request, c caller) {
	uid := r.URL.Query().Get("userId")
	if s.p.Flavor == Emby {
		uid = r.PathValue("user")
	}
	user, ok := s.asUser(c, uid)
	it, found := s.core.Item(r.PathValue("id"))
	if !ok || !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	d := s.core.UserData(user, it.Key)
	d.Played, d.PlayCount, d.LastPlayed = true, d.PlayCount+1, time.Now()
	s.core.SetUserData(user, it.Key, d)
	writeJSON(w, s.renderUserData(user, it))
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
