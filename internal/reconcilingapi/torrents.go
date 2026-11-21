package reconcilingapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

type rehomeInput struct{ Body rehomeRequest }

type rehomeRequest struct {
	Roots []saveRoot `json:"roots" nullable:"false" maxItems:"1000" doc:"Each library root to save. A root left out stays for later: no scan reads it and Reconciling waits on it"`
	Done  bool       `json:"done" doc:"Whether this ends the step, so SlipStream stops waiting for the old Controller's folders"`
}

type saveRoot struct {
	RootID string `json:"rootId" format:"uuid"`
	NodeID string `json:"nodeId,omitempty" format:"uuid" doc:"The Storage Node whose disk holds it now; left out, its Node"`
	Path   string `json:"path,omitempty" maxLength:"4096" doc:"The folder that holds it now, in its Node's path style; left out, its folder"`
	Forget *int   `json:"forget,omitempty" minimum:"0" doc:"The missing file count the Admin confirmed, for Save anyway: those files are forgotten"`
}

type rehomeOutput struct{ Body rehomed }

type rehomed struct {
	Roots []savedRoot `json:"roots" nullable:"false" doc:"How each root's save went, in the order asked"`
}

type savedRoot struct {
	RootID  string               `json:"rootId" format:"uuid"`
	Refused api.Nullable[string] `json:"refused" doc:"Why the root edit refused it, null once it is saved and its scan starts"`
	Missing api.Nullable[int]    `json:"missing" doc:"How many recorded files aren't at the place, when that is why. Send it as forget to save anyway"`
}

func rehome(ctx context.Context, s *reconciling.Service, by identity.Admin, in *rehomeInput) (*rehomeOutput, error) {
	save := make([]reconciling.Save, 0, len(in.Body.Roots))
	for i, r := range in.Body.Roots {
		id, err := storage.ParseUUID(r.RootID)
		if err != nil {
			return nil, field(fmt.Sprintf("body.roots[%d].rootId", i), "Use a library root's id.")
		}
		sv := reconciling.Save{Root: roots.ID{UUID: id}, Rehome: roots.Rehome{Forget: r.Forget}}
		if r.NodeID != "" {
			node, err := storage.ParseUUID(r.NodeID)
			if err != nil {
				return nil, field(fmt.Sprintf("body.roots[%d].nodeId", i), "Use a Storage Node's id.")
			}
			sv.Node = &nodes.ID{UUID: node}
		}
		if r.Path != "" {
			sv.Path = &r.Path
		}
		save = append(save, sv)
	}
	saved, err := s.Rehome(ctx, by, save, in.Body.Done)
	if err != nil {
		return nil, answer(err)
	}
	out := &rehomeOutput{}
	out.Body.Roots = make([]savedRoot, len(saved))
	for i, sv := range saved {
		row := savedRoot{RootID: sv.Root.String()}
		if sv.Err != nil {
			row.Refused = api.Some(sv.Err.Error())
			if missing, ok := sv.Err.(*roots.MissingError); ok { //nolint:errorlint // Rehome returns it bare
				row.Missing = api.Some(missing.Count)
			}
		}
		out.Body.Roots[i] = row
	}
	return out, nil
}

func field(location, message string) error {
	return api.Invalid.Fields(api.FieldError{Location: location, Message: message})
}

type listInput struct {
	Reason string `query:"reason" enum:"not_held,conflict" doc:"Only this group's reason; left out, any"`
	Found  string `query:"found" enum:"not_looked,no_files,files" doc:"Only torrents whose files were found so; left out, any"`
	After  string `query:"after" maxLength:"200" doc:"The next of the previous page; left out, the first page"`
	Limit  int    `query:"limit" minimum:"1" maximum:"500" default:"100"`
}

type listOutput struct{ Body reviewPage }

type reviewPage struct {
	Torrents []reviewed           `json:"torrents" nullable:"false"`
	Next     api.Nullable[string] `json:"next" doc:"The after of the next page; null on the last page. Rows added or removed meanwhile never move it"`
}

type reviewed struct {
	ID     string               `json:"id" format:"uuid"`
	Name   string               `json:"name"`
	NodeID string               `json:"nodeId" format:"uuid" doc:"The Node it was recorded on"`
	Reason string               `json:"reason" enum:"not_held,conflict"`
	Found  string               `json:"found" enum:"not_looked,no_files,files"`
	Since  time.Time            `json:"since" doc:"When it joined the review list"`
	Readd  api.Nullable[string] `json:"readd" format:"uuid" doc:"The Node Re-add puts it on once Reconciling finishes; null until Re-add is pressed"`
}

func filterOf(reason, found string) reconciling.Filter {
	var f reconciling.Filter
	if reason != "" {
		f.Reason = new(reconciling.Reason(reason))
	}
	for k, w := range foundWords {
		if w == found {
			f.Found = new(k)
		}
	}
	return f
}

func list(ctx context.Context, s *reconciling.Service, in *listInput) (*listOutput, error) {
	var after *reconciling.Cursor
	if in.After != "" {
		c, err := cursorOf(in.After)
		if err != nil {
			return nil, field("query.after", "Use the next of a page this list gave.")
		}
		after = &c
	}
	rows, next, err := s.ReviewPage(ctx, filterOf(in.Reason, in.Found), after, in.Limit)
	if err != nil {
		return nil, err
	}
	out := &listOutput{}
	out.Body.Torrents = make([]reviewed, len(rows))
	for i, r := range rows {
		row := reviewed{ID: r.Torrent.String(), Name: r.Name, NodeID: r.Node.String(), Reason: string(r.Reason), Found: foundWords[r.Found], Since: r.Since.Time()}
		if r.Readd != nil {
			row.Readd = api.Some(r.Readd.String())
		}
		out.Body.Torrents[i] = row
	}
	if next != nil {
		out.Body.Next = api.Some(cursorText(*next))
	}
	return out, nil
}

// cursorText is c as one opaque token.
func cursorText(c reconciling.Cursor) string {
	raw := strings.Join([]string{string(c.Reason), strconv.Itoa(int(c.Found)), strconv.FormatInt(c.Since.Time().UnixMicro(), 10), c.Torrent.String()}, " ")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func cursorOf(s string) (reconciling.Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return reconciling.Cursor{}, fmt.Errorf("read a review list cursor: %w", err)
	}
	parts := strings.Split(string(raw), " ")
	if len(parts) != 4 {
		return reconciling.Cursor{}, fmt.Errorf("a cursor has 4 parts, this one %d", len(parts))
	}
	found, err := strconv.Atoi(parts[1])
	if err != nil {
		return reconciling.Cursor{}, fmt.Errorf("read a review list cursor: %w", err)
	}
	since, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return reconciling.Cursor{}, fmt.Errorf("read a review list cursor: %w", err)
	}
	id, err := storage.ParseUUID(parts[3])
	if err != nil {
		return reconciling.Cursor{}, err
	}
	return reconciling.Cursor{
		Reason: reconciling.Reason(parts[0]), Found: reconciling.Found(found), //nolint:gosec // a Found is -1, 0 or 1
		Since: storage.FromTime(time.UnixMicro(since)), Torrent: downloadclients.ID{UUID: id},
	}, nil
}

// pick is which review list torrents an action takes: the ids given, or
// every torrent in a group.
type pick struct {
	IDs    []string `json:"ids,omitempty" maxItems:"1000" doc:"The torrents; left out, every torrent in the group reason and found name"`
	Reason string   `json:"reason,omitempty" enum:"not_held,conflict" doc:"With no ids, the group's reason; left out, any"`
	Found  string   `json:"found,omitempty" enum:"not_looked,no_files,files" doc:"With no ids, how the group's files were found; left out, any"`
}

const groupPage = 500

func (p pick) ids(ctx context.Context, s *reconciling.Service) ([]downloadclients.ID, error) {
	if len(p.IDs) > 0 {
		out := make([]downloadclients.ID, len(p.IDs))
		for i, raw := range p.IDs {
			u, err := storage.ParseUUID(raw)
			if err != nil {
				return nil, field(fmt.Sprintf("body.ids[%d]", i), "Use a torrent's id from the review list.")
			}
			out[i] = downloadclients.ID{UUID: u}
		}
		return out, nil
	}
	var out []downloadclients.ID
	var after *reconciling.Cursor
	for {
		rows, next, err := s.ReviewPage(ctx, filterOf(p.Reason, p.Found), after, groupPage)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, r.Torrent)
		}
		if next == nil {
			return out, nil
		}
		after = next
	}
}

type readdInput struct{ Body readdRequest }

type readdRequest struct {
	pick
	NodeID string `json:"nodeId,omitempty" format:"uuid" doc:"The Node to re-add them on; left out, each on the Node it was recorded on"`
}

func readd(ctx context.Context, s *reconciling.Service, in *readdInput) error {
	ids, err := in.Body.ids(ctx, s)
	if err != nil {
		return err
	}
	var onto *nodes.ID
	if in.Body.NodeID != "" {
		u, err := storage.ParseUUID(in.Body.NodeID)
		if err != nil {
			return field("body.nodeId", "Use a Node's id.")
		}
		onto = &nodes.ID{UUID: u}
	}
	return answer(s.Readd(ctx, ids, onto))
}

type forgetInput struct{ Body forgetRequest }

type forgetRequest struct{ pick }

func forget(ctx context.Context, s *reconciling.Service, in *forgetInput) error {
	if len(in.Body.IDs) == 0 {
		in.Body.Reason = string(reconciling.NotHeld)
	}
	ids, err := in.Body.ids(ctx, s)
	if err != nil {
		return err
	}
	return answer(s.Forget(ctx, ids))
}
