// Package jellyfin is the wire client for Jellyfin and Emby, which share
// one API family and differ in how a call signs in and in some routes: the
// library listing, the one-library scan, UserData, played items and the
// path lookup. Every type here is the wire's, and the kinds translate.
// Written against Jellyfin 12.2.0 and 12.4.0 (#71, recorded run
// 2026-11-14) and Emby 4.10.1.0 (#71, #101), checked against Jellyfin's
// source at v12.4 and the Emby SDK's OpenAPI document.
package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Flavor is which product a Client speaks to.
type Flavor int

const (
	Jellyfin Flavor = iota
	Emby
)

func (f Flavor) String() string {
	if f == Emby {
		return "emby"
	}
	return "jellyfin"
}

// App names SlipStream to the server. DeviceID is stable per server,
// since Jellyfin revokes a token when the same DeviceId signs in again.
type App struct {
	Client   string
	Device   string
	DeviceID string
	Version  string
}

// Client is one Jellyfin or Emby server at base, with an API key or a
// user's access token.
type Client struct {
	base   *url.URL
	flavor Flavor
	token  string
	app    App
	http   *http.Client
}

func New(base *url.URL, f Flavor, token string, app App, c *http.Client) *Client {
	return &Client{base: base, flavor: f, token: token, app: app, http: c}
}

// Error is a response other than 2xx.
type Error struct {
	Status int
	// Detail is the start of the response body.
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("responded with status %d: %s", e.Status, e.Detail)
}

// Refused is whether err is the server refusing the credential, or its
// absence.
func Refused(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusUnauthorized
}

// NotFound is whether err is a 404.
func NotFound(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusNotFound
}

// authorization is the header value that names the app, and the token
// when there is one. Jellyfin 12 honors only the MediaBrowser scheme
// while legacy authorization is off, its default; Emby takes its own.
func (c *Client) authorization(token string) string {
	scheme := "MediaBrowser"
	if c.flavor == Emby {
		scheme = "Emby"
	}
	fields := []string{
		`Client="` + c.app.Client + `"`, `Device="` + c.app.Device + `"`,
		`DeviceId="` + c.app.DeviceID + `"`, `Version="` + c.app.Version + `"`,
	}
	if token != "" {
		fields = append(fields, `Token="`+token+`"`)
	}
	return scheme + " " + strings.Join(fields, ", ")
}

// sign puts the app and the token on req, each product's way: Jellyfin's
// Authorization header, Emby's X-Emby-Authorization and X-Emby-Token.
func (c *Client) sign(req *http.Request, token bool) {
	t := ""
	if token {
		t = c.token
	}
	if c.flavor == Emby {
		req.Header.Set("X-Emby-Authorization", c.authorization(""))
		if t != "" {
			req.Header.Set("X-Emby-Token", t)
		}
		return
	}
	req.Header.Set("Authorization", c.authorization(t))
}

func (c *Client) url(path string, q url.Values) string {
	u := c.base.JoinPath(path)
	u.RawQuery = q.Encode()
	return u.String()
}

// do sends one request with a JSON body when in isn't nil. With token
// false it carries the app and no token. A 2xx response's JSON body
// decodes into out when out isn't nil.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, token bool, in, out any) error {
	body := io.Reader(http.NoBody)
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("%s %s %s: %w", c.flavor, method, path, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path, q), body)
	if err != nil {
		return fmt.Errorf("%s %s %s: %w", c.flavor, method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.sign(req, token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s %s: %w", c.flavor, method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("%s %s %s: read the response: %w", c.flavor, method, path, err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s %s: %w", c.flavor, method, path, &Error{Status: resp.StatusCode, Detail: detail(raw)})
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s %s %s: decode the response: %w", c.flavor, method, path, err)
	}
	return nil
}

func detail(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// PublicInfo is GET /System/Info/Public, which needs no credential.
type PublicInfo struct {
	ID                     string `json:"Id"`
	ServerName             string `json:"ServerName"`
	Version                string `json:"Version"`
	ProductName            string `json:"ProductName"`
	LocalAddress           string `json:"LocalAddress"`
	StartupWizardCompleted bool   `json:"StartupWizardCompleted"`
}

func (c *Client) PublicInfo(ctx context.Context) (PublicInfo, error) {
	var out PublicInfo
	err := c.do(ctx, http.MethodGet, "/System/Info/Public", nil, false, nil, &out)
	return out, err
}

// Configuration is the part of GET /System/Configuration SlipStream
// reads. It needs an administrator or an API key.
type Configuration struct {
	// MonitorDelay is how long the server waits after a path report:
	// Jellyfin's LibraryMonitorDelay, 60 s by default, and Emby's
	// LibraryMonitorDelaySeconds, 90 s by default.
	MonitorDelay time.Duration
}

func (c *Client) Configuration(ctx context.Context) (Configuration, error) {
	var out struct {
		LibraryMonitorDelay        *int `json:"LibraryMonitorDelay"`
		LibraryMonitorDelaySeconds *int `json:"LibraryMonitorDelaySeconds"`
	}
	if err := c.do(ctx, http.MethodGet, "/System/Configuration", nil, true, nil, &out); err != nil {
		return Configuration{}, err
	}
	delay := out.LibraryMonitorDelay
	if c.flavor == Emby {
		delay = out.LibraryMonitorDelaySeconds
	}
	if delay == nil {
		return Configuration{}, fmt.Errorf("%s /System/Configuration has no monitor delay", c.flavor)
	}
	return Configuration{MonitorDelay: time.Duration(*delay) * time.Second}, nil
}

// User is one server user.
type User struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Policy struct {
		IsAdministrator bool `json:"IsAdministrator"`
	} `json:"Policy"`
}

// Auth is a user signing in: the access token and who it is.
type Auth struct {
	AccessToken string `json:"AccessToken"`
	User        User   `json:"User"`
	ServerID    string `json:"ServerId"`
}

// AuthenticateByName signs user in with the app's DeviceId, which revokes
// a token the same DeviceId had (#71).
func (c *Client) AuthenticateByName(ctx context.Context, user, password string) (Auth, error) {
	var out Auth
	err := c.do(ctx, http.MethodPost, "/Users/AuthenticateByName", nil, false, map[string]string{"Username": user, "Pw": password}, &out)
	return out, err
}

// Logout ends the session the client's token is.
func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/Sessions/Logout", nil, true, nil, nil)
}

func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out []User
	err := c.do(ctx, http.MethodGet, "/Users", nil, true, nil, &out)
	return out, err
}

// Key is one API key.
type Key struct {
	AccessToken string    `json:"AccessToken"`
	AppName     string    `json:"AppName"`
	DateCreated time.Time `json:"DateCreated"`
}

// CreateKey makes an API key named app. Jellyfin answers 204 with no
// body, so Keys reads it back.
func (c *Client) CreateKey(ctx context.Context, app string) error {
	name := "app"
	if c.flavor == Emby {
		name = "App"
	}
	return c.do(ctx, http.MethodPost, "/Auth/Keys", url.Values{name: {app}}, true, nil, nil)
}

func (c *Client) Keys(ctx context.Context) ([]Key, error) {
	var out struct {
		Items []Key `json:"Items"`
	}
	err := c.do(ctx, http.MethodGet, "/Auth/Keys", nil, true, nil, &out)
	return out.Items, err
}

func (c *Client) RevokeKey(ctx context.Context, key string) error {
	return c.do(ctx, http.MethodDelete, "/Auth/Keys/"+url.PathEscape(key), nil, true, nil, nil)
}

// VirtualFolder is one library and its folders as the server's OS names
// them. ItemID is the library's item, the parent of its items.
type VirtualFolder struct {
	Name            string   `json:"Name"`
	CollectionType  string   `json:"CollectionType"`
	ItemID          string   `json:"ItemId"`
	Locations       []string `json:"Locations"`
	RefreshStatus   string   `json:"RefreshStatus"`
	RefreshProgress *float64 `json:"RefreshProgress"`
}

// Libraries are the server's libraries: Jellyfin's GET
// /Library/VirtualFolders, which needs elevation, and Emby's
// /Library/VirtualFolders/Query, which has no plain GET.
func (c *Client) Libraries(ctx context.Context) ([]VirtualFolder, error) {
	if c.flavor == Emby {
		var out struct {
			Items []VirtualFolder `json:"Items"`
		}
		err := c.do(ctx, http.MethodGet, "/Library/VirtualFolders/Query", nil, true, nil, &out)
		return out.Items, err
	}
	var out []VirtualFolder
	err := c.do(ctx, http.MethodGet, "/Library/VirtualFolders", nil, true, nil, &out)
	return out, err
}

// UpdateType is a path report's kind. Jellyfin ignores it; Emby removes a
// Deleted file at once (#101).
type UpdateType string

const (
	Created  UpdateType = "Created"
	Modified UpdateType = "Modified"
	Deleted  UpdateType = "Deleted"
)

// Update is one path report.
type Update struct {
	Path       string     `json:"Path"`
	UpdateType UpdateType `json:"UpdateType"`
}

// ReportUpdated is POST /Library/Media/Updated, answered 204 at once. The
// server scans after its monitor delay.
func (c *Client) ReportUpdated(ctx context.Context, updates []Update) error {
	return c.do(ctx, http.MethodPost, "/Library/Media/Updated", nil, true, map[string][]Update{"Updates": updates}, nil)
}

// RefreshAll queues the RefreshLibrary task, a scan of every library, and
// answers 204 without waiting.
func (c *Client) RefreshAll(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/Library/Refresh", nil, true, nil, nil)
}

// RefreshLibrary is Emby's recursive refresh of one library's item, seen
// in that library's RefreshStatus. Jellyfin 12.4's item refresh has no
// Recursive, so it has no route for one library.
func (c *Client) RefreshLibrary(ctx context.Context, itemID string) error {
	if c.flavor != Emby {
		return fmt.Errorf("%s has no refresh of one library", c.flavor)
	}
	q := url.Values{"Recursive": {"true"}, "MetadataRefreshMode": {"Default"}, "ImageRefreshMode": {"Default"}}
	return c.do(ctx, http.MethodPost, "/Items/"+url.PathEscape(itemID)+"/Refresh", q, true, nil, nil)
}

// RefreshLibraryTask is the Key of the task RefreshAll queues.
const RefreshLibraryTask = "RefreshLibrary"

// Task is one scheduled task. A scan has ended when State is Idle and
// LastExecutionResult.EndTimeUtc is later than it was before the scan was
// asked for: a small library goes from Running to Idle inside half a
// second, between two polls (recorded run 2026-11-14).
type Task struct {
	Name                      string      `json:"Name"`
	State                     string      `json:"State"`
	ID                        string      `json:"Id"`
	Key                       string      `json:"Key"`
	CurrentProgressPercentage *float64    `json:"CurrentProgressPercentage"`
	LastExecutionResult       *TaskResult `json:"LastExecutionResult"`
}

type TaskResult struct {
	StartTimeUtc time.Time `json:"StartTimeUtc"`
	EndTimeUtc   time.Time `json:"EndTimeUtc"`
	Status       string    `json:"Status"`
}

// TaskIdle is a task's State when it isn't running.
const TaskIdle = "Idle"

func (c *Client) Tasks(ctx context.Context) ([]Task, error) {
	var out []Task
	err := c.do(ctx, http.MethodGet, "/ScheduledTasks", nil, true, nil, &out)
	return out, err
}

// Task is the scheduled task with key, false when the server has none.
func (c *Client) Task(ctx context.Context, key string) (Task, bool, error) {
	all, err := c.Tasks(ctx)
	if err != nil {
		return Task{}, false, err
	}
	for _, t := range all {
		if t.Key == key {
			return t, true, nil
		}
	}
	return Task{}, false, nil
}

// Item is one item. Jellyfin lists alternate versions only in the primary
// item's MediaSources, and Emby makes each file its own item, joined by
// PresentationUniqueKey.
type Item struct {
	ID                    string            `json:"Id"`
	Name                  string            `json:"Name"`
	Type                  string            `json:"Type"`
	Path                  string            `json:"Path"`
	ServerID              string            `json:"ServerId"`
	ProductionYear        int               `json:"ProductionYear"`
	SeriesID              string            `json:"SeriesId"`
	SeriesName            string            `json:"SeriesName"`
	ParentIndexNumber     int               `json:"ParentIndexNumber"`
	IndexNumber           int               `json:"IndexNumber"`
	ProviderIDs           map[string]string `json:"ProviderIds"`
	PresentationUniqueKey string            `json:"PresentationUniqueKey"`
	MediaSources          []MediaSource     `json:"MediaSources"`
	UserData              *UserData         `json:"UserData"`
}

// MediaSource is one version of an item.
type MediaSource struct {
	ID           string   `json:"Id"`
	Path         string   `json:"Path"`
	Name         string   `json:"Name"`
	MediaStreams []Stream `json:"MediaStreams"`
}

// Stream is one stream of a version. A Sidecar is an external stream with
// its own Path.
type Stream struct {
	Type       string `json:"Type"`
	Codec      string `json:"Codec"`
	IsExternal bool   `json:"IsExternal"`
	Path       string `json:"Path"`
}

// UserData is one user's data on an item. Positions are 100 ns ticks.
type UserData struct {
	PlaybackPositionTicks int64      `json:"PlaybackPositionTicks"`
	PlayCount             int        `json:"PlayCount"`
	IsFavorite            bool       `json:"IsFavorite"`
	Played                bool       `json:"Played"`
	LastPlayedDate        *time.Time `json:"LastPlayedDate,omitempty"`
}

// Item types a library query asks for.
const (
	TypeMovie   = "Movie"
	TypeEpisode = "Episode"
	TypeBoxSet  = "BoxSet"
)

// Query is a GET /Items listing, as user so each item carries the user's
// data.
type Query struct {
	User      string
	Parent    string
	Recursive bool
	Types     []string
	Start     int
	Limit     int
}

// Page is one page of items and how many the query matches.
type Page struct {
	Items []Item `json:"Items"`
	Total int    `json:"TotalRecordCount"`
}

const fields = "Path,MediaSources,ProviderIds,PresentationUniqueKey"

// Items lists one page. Jellyfin 12.4 has no path parameter, so a lookup
// by path pages through a library and checks every version's path.
func (c *Client) Items(ctx context.Context, q Query) (Page, error) {
	v := url.Values{"Fields": {fields}, "StartIndex": {strconv.Itoa(q.Start)}}
	if q.Limit > 0 {
		v.Set("Limit", strconv.Itoa(q.Limit))
	}
	if q.Parent != "" {
		v.Set("ParentId", q.Parent)
	}
	if q.Recursive {
		v.Set("Recursive", "true")
	}
	if len(q.Types) > 0 {
		v.Set("IncludeItemTypes", strings.Join(q.Types, ","))
	}
	path := "/Items"
	if c.flavor == Emby {
		path = "/Users/" + url.PathEscape(q.User) + "/Items"
	} else {
		v.Set("userId", q.User)
	}
	var out Page
	err := c.do(ctx, http.MethodGet, path, v, true, nil, &out)
	return out, err
}

// ItemsAt is Emby's lookup by file path, GET /Items?Path=, as user. It
// matches the path exactly (conformance case PathLookupExact).
func (c *Client) ItemsAt(ctx context.Context, user, path string) ([]Item, error) {
	if c.flavor != Emby {
		return nil, fmt.Errorf("%s has no lookup by path", c.flavor)
	}
	var out Page
	err := c.do(ctx, http.MethodGet, "/Users/"+url.PathEscape(user)+"/Items", url.Values{"Path": {path}, "Fields": {fields}, "Recursive": {"true"}}, true, nil, &out)
	return out.Items, err
}

// Item is one item as user, false on 404.
func (c *Client) Item(ctx context.Context, user, id string) (Item, bool, error) {
	path, q := "/Items/"+url.PathEscape(id), url.Values{"userId": {user}, "Fields": {fields}}
	if c.flavor == Emby {
		path, q = "/Users/"+url.PathEscape(user)+"/Items/"+url.PathEscape(id), url.Values{"Fields": {fields}}
	}
	var out Item
	err := c.do(ctx, http.MethodGet, path, q, true, nil, &out)
	if NotFound(err) {
		return Item{}, false, nil
	}
	return out, err == nil, err
}

func (c *Client) userDataRoute(user, item string) (string, url.Values) {
	if c.flavor == Emby {
		return "/Users/" + url.PathEscape(user) + "/Items/" + url.PathEscape(item) + "/UserData", nil
	}
	return "/UserItems/" + url.PathEscape(item) + "/UserData", url.Values{"userId": {user}}
}

// UserData is user's data on item: Jellyfin 12's /UserItems route, and
// Emby's /Users/{user}/Items/{item}, where /UserItems answers 404.
func (c *Client) UserData(ctx context.Context, user, item string) (UserData, error) {
	if c.flavor == Emby {
		it, ok, err := c.Item(ctx, user, item)
		if err != nil {
			return UserData{}, err
		}
		if !ok || it.UserData == nil {
			return UserData{}, fmt.Errorf("emby item %s has no user data", item)
		}
		return *it.UserData, nil
	}
	path, q := c.userDataRoute(user, item)
	var out UserData
	err := c.do(ctx, http.MethodGet, path, q, true, nil, &out)
	return out, err
}

// SetUserData writes user's data on item. Emby answers success and keeps
// the play count and last played date it had (#101).
func (c *Client) SetUserData(ctx context.Context, user, item string, d UserData) error {
	path, q := c.userDataRoute(user, item)
	return c.do(ctx, http.MethodPost, path, q, true, d, nil)
}

// MarkPlayed marks item played for user. On Emby it sets no play count
// or last played date either (#101).
func (c *Client) MarkPlayed(ctx context.Context, user, item string) error {
	if c.flavor == Emby {
		return c.do(ctx, http.MethodPost, "/Users/"+url.PathEscape(user)+"/PlayedItems/"+url.PathEscape(item), nil, true, nil, nil)
	}
	return c.do(ctx, http.MethodPost, "/UserPlayedItems/"+url.PathEscape(item), url.Values{"userId": {user}}, true, nil, nil)
}

// CreateCollection makes the collection name holding items and answers
// its id.
func (c *Client) CreateCollection(ctx context.Context, name string, items []string) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := c.do(ctx, http.MethodPost, "/Collections", url.Values{"Name": {name}, "Ids": {strings.Join(items, ",")}}, true, nil, &out)
	return out.ID, err
}

// AddToCollection puts items in the collection id.
func (c *Client) AddToCollection(ctx context.Context, id string, items []string) error {
	return c.do(ctx, http.MethodPost, "/Collections/"+url.PathEscape(id)+"/Items", url.Values{"Ids": {strings.Join(items, ",")}}, true, nil, nil)
}
