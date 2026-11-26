package nodeapi

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// MediaServerIDs are the Media Servers' ids by name.
func MediaServerIDs(ctx context.Context, c *http.Client, baseURL, apiKey string) (map[string]string, error) {
	var out struct {
		Servers []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"servers"`
	}
	err := call(ctx, c, request{
		what: "list the media servers", method: http.MethodGet, url: baseURL + "/api/v1/media-servers",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	ids := map[string]string{}
	for _, s := range out.Servers {
		ids[s.Name] = s.ID
	}
	return ids, err
}

// WorkFile is a file of a Work: its root and its path in it, '/'
// separated.
type WorkFile struct {
	RootID string
	Path   string
}

// WorkFiles are each Work's files by Work id.
func WorkFiles(ctx context.Context, c *http.Client, baseURL, apiKey string) (map[string][]WorkFile, error) {
	var library struct {
		Rows []struct {
			ID     string `json:"id"`
			RootID string `json:"rootId"`
		} `json:"rows"`
	}
	if err := call(ctx, c, request{
		what: "list the library", method: http.MethodGet, url: baseURL + "/api/v1/library",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &library); err != nil {
		return nil, err
	}
	out := map[string][]WorkFile{}
	for _, w := range library.Rows {
		var listed struct {
			Files []struct {
				Path string `json:"path"`
			} `json:"files"`
		}
		if err := call(ctx, c, request{
			what: "list the files of Work " + w.ID, method: http.MethodGet, url: baseURL + "/api/v1/works/" + w.ID + "/files",
			header: apiKeyHeader(apiKey), want: http.StatusOK,
		}, &listed); err != nil {
			return nil, err
		}
		for _, f := range listed.Files {
			out[w.ID] = append(out[w.ID], WorkFile{RootID: w.RootID, Path: f.Path})
		}
	}
	return out, nil
}

// OpenIn are the ids of the Media Servers the Work page opens the Work
// in. Each read asks a server whose index holds none of its files to
// look them up.
func OpenIn(ctx context.Context, c *http.Client, baseURL, apiKey, work string) ([]string, error) {
	var out struct {
		OpenIn []struct {
			ServerID string `json:"serverId"`
		} `json:"openIn"`
	}
	err := call(ctx, c, request{
		what: "get Work " + work, method: http.MethodGet, url: baseURL + "/api/v1/works/" + work,
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	ids := make([]string, len(out.OpenIn))
	for i, o := range out.OpenIn {
		ids[i] = o.ServerID
	}
	return ids, err
}

// Play is a play as listMediaServerPlays lists it.
type Play struct {
	PlayerKey  string     `json:"playerKey"`
	Path       string     `json:"path"`
	Work       *struct{}  `json:"work"`
	StartedAt  time.Time  `json:"startedAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	EndedAt    *time.Time `json:"endedAt"`
}

// Plays are every play of the server id, newest first.
func Plays(ctx context.Context, c *http.Client, baseURL, apiKey, id string) ([]Play, error) {
	var all []Play
	cursor := ""
	for {
		var page struct {
			Plays []Play `json:"plays"`
			Next  string `json:"next"`
		}
		u := baseURL + "/api/v1/media-servers/" + id + "/plays"
		if cursor != "" {
			u += "?cursor=" + url.QueryEscape(cursor)
		}
		if err := call(ctx, c, request{
			what: "list the plays of media server " + id, method: http.MethodGet, url: u,
			header: apiKeyHeader(apiKey), want: http.StatusOK,
		}, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Plays...)
		if page.Next == "" {
			return all, nil
		}
		cursor = page.Next
	}
}
