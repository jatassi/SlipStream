package filesapi

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	templateRefused     = api.NewCode(http.StatusUnprocessableEntity, "naming_template_refused", "The naming settings fail the save check")
	rootNotReady        = api.NewCode(http.StatusConflict, "root_not_ready", "A library root isn't ready")
)

// confirmLimit fits a confirm of 60,000 Works, three times the largest
// library the plan is sized for.
const confirmLimit = 8 << 20

type renameScope struct {
	Kind string `json:"kind" enum:"work,type,library"`
	ID   string `json:"id,omitempty" doc:"The Work's id for work, the Content Type's id for type; left out for library"`
}

type renameDraft struct {
	ContentType string                     `json:"contentType" doc:"The Content Type whose naming settings the draft changes"`
	Values      map[string]json.RawMessage `json:"values" doc:"Each naming setting's unsaved value by key, over the stored ones"`
}

type planInput struct {
	Body struct {
		Scope renameScope  `json:"scope"`
		Draft *renameDraft `json:"draft,omitempty" doc:"Unsaved naming settings to plan under instead of the stored ones"`
	}
}

type workLine struct {
	WorkID      string `json:"workId" format:"uuid"`
	ContentType string `json:"contentType"`
	Title       string `json:"title"`
	Digest      string `json:"digest" doc:"A hash of the Work's rows, which a confirm sends back"`
	Rename      int    `json:"rename" minimum:"0" doc:"Files that can be renamed now"`
	Blocked     int    `json:"blocked" minimum:"0" doc:"Files that can't be renamed now"`
	Moving      int    `json:"moving" minimum:"0" doc:"Files the disk queue still moves, which the plan leaves out"`
}

type renamePlan struct {
	Files   int        `json:"files" minimum:"0" doc:"Files that can be renamed now, in every Work"`
	Blocked int        `json:"blocked" minimum:"0" doc:"Files that can't be renamed now, in every Work"`
	Works   []workLine `json:"works" nullable:"false" doc:"Each Work with a file to rename, a blocked file or a file still moving"`
}

type renamePlanOutput struct{ Body renamePlan }

func (h handlers) planRenames(ctx context.Context, by identity.Admin, in *planInput) (*renamePlanOutput, error) {
	s, err := scopeOf(in.Body.Scope)
	if err != nil {
		return nil, err
	}
	ps, err := open(h.lib).Plan(ctx, by, s, draftOf(in.Body.Draft))
	if err != nil {
		return nil, planRefused(err)
	}
	out := renamePlan{Works: make([]workLine, len(ps))}
	for i, p := range ps {
		line := workLine{WorkID: p.Work.String(), ContentType: string(p.Type), Title: p.Title, Digest: p.Digest.String(), Moving: p.Moving}
		for _, r := range p.Rows {
			if r.Blocked == (files.Blocked{}) {
				line.Rename++
			} else {
				line.Blocked++
			}
		}
		out.Files += line.Rename
		out.Blocked += line.Blocked
		out.Works[i] = line
	}
	return &renamePlanOutput{Body: out}, nil
}

func scopeOf(s renameScope) (files.Scope, error) {
	switch s.Kind {
	case "work":
		u, err := storage.ParseUUID(s.ID)
		if err != nil {
			return files.Scope{}, workNotFound
		}
		return files.WorkScope(works.WorkID{UUID: u}), nil
	case "type":
		return files.TypeScope(works.TypeID(s.ID)), nil
	}
	return files.LibraryScope(), nil
}

func draftOf(d *renameDraft) *files.Draft {
	if d == nil {
		return nil
	}
	return &files.Draft{Type: works.TypeID(d.ContentType), Patch: setting.Patch(d.Values)}
}

// planRefused is a plan's error as a problem: a draft the save check
// refuses has one field error per setting.
func planRefused(err error) error {
	switch {
	case errors.Is(err, works.ErrNoWork):
		return workNotFound
	case errors.Is(err, works.ErrNoType):
		return contentTypeNotFound
	}
	if refused, ok := errors.AsType[*naming.RefusedError](err); ok {
		fields := make([]api.FieldError, len(refused.Problems))
		for i, p := range refused.Problems {
			fields[i] = api.FieldError{Location: "body.draft.values." + p.Key, Message: p.Sentence}
		}
		return templateRefused.Fields(fields...)
	}
	return surveyed(settingapi.Refused(err))
}

// surveyed is err, or rootNotReady when a root's Node can't list the
// files at a plan's new paths now.
func surveyed(err error) error {
	if _, ok := errors.AsType[*storing.NotReady](err); ok {
		return rootNotReady.Because("A plan lists the files at each new path on the root's Node first. Try again once the root is ready.")
	}
	return err
}

type workPlanInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   *struct {
		Draft *renameDraft `json:"draft,omitempty" doc:"Unsaved naming settings to plan under instead of the stored ones"`
	}
}

type rowBlock struct {
	Reason      string `json:"reason" enum:"occupied,too_long,clash"`
	OtherWorkID string `json:"otherWorkId,omitempty" format:"uuid" doc:"For clash, the Work whose file ends at the same path"`
	Detail      string `json:"detail" doc:"Why the file can't be renamed now, then what to do"`
}

type renameRow struct {
	FileID     string    `json:"fileId" format:"uuid"`
	From       string    `json:"from" doc:"Where the file is now, relative to its root"`
	To         string    `json:"to" doc:"Where the naming templates put it; empty when its name doesn't fit"`
	Group      int       `json:"group" minimum:"0" doc:"0 for a row joined to no other; rows with one number trade or chain paths, so they are renamed or left together"`
	Companions []string  `json:"companions" nullable:"false" doc:"Files beside it that move with it, such as its subtitles"`
	Blocked    *rowBlock `json:"blocked,omitempty" doc:"Why it can't be renamed now; absent when it can"`
}

type workRenamePlan struct {
	WorkID      string      `json:"workId" format:"uuid"`
	ContentType string      `json:"contentType"`
	Title       string      `json:"title"`
	Digest      string      `json:"digest" doc:"A hash of the Work's rows, which a confirm sends back"`
	Moving      int         `json:"moving" minimum:"0" doc:"Files the disk queue still moves, which the plan leaves out"`
	Rows        []renameRow `json:"rows" nullable:"false" doc:"Each file whose path differs from what the naming templates make"`
}

type workRenamePlanOutput struct{ Body workRenamePlan }

func (h handlers) planWorkRenames(ctx context.Context, by identity.Admin, in *workPlanInput) (*workRenamePlanOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	var draft *renameDraft
	if in.Body != nil {
		draft = in.Body.Draft
	}
	p, err := open(h.lib).PlanOf(ctx, by, works.WorkID{UUID: u}, draftOf(draft))
	if err != nil {
		return nil, planRefused(err)
	}
	out := workRenamePlan{WorkID: p.Work.String(), ContentType: string(p.Type), Title: p.Title, Digest: p.Digest.String(), Moving: p.Moving, Rows: make([]renameRow, len(p.Rows))}
	for i, r := range p.Rows {
		out.Rows[i] = renameRow{FileID: r.File.String(), From: r.From, To: r.To, Group: r.Group, Companions: append([]string{}, r.Companions...), Blocked: blockOf(r.Blocked)}
	}
	return &workRenamePlanOutput{Body: out}, nil
}

func blockOf(b files.Blocked) *rowBlock {
	switch b.Reason {
	case files.Occupied:
		return &rowBlock{Reason: "occupied", Detail: "A file SlipStream doesn't track is at the new path. Move or delete that file, then plan again."}
	case files.TooLong:
		return &rowBlock{Reason: "too_long", Detail: b.Long.Sentence()}
	case files.Clash:
		return &rowBlock{Reason: "clash", OtherWorkID: b.Other.String(), Detail: "Another Work's file has the same new path. Add an id to the folder template so each Work gets its own folder."}
	}
	return nil
}

type confirmedWork struct {
	Work   string   `json:"work" format:"uuid"`
	Digest string   `json:"digest" pattern:"^[0-9a-f]{16}$" doc:"The digest the Work's plan showed"`
	Leave  []string `json:"leave,omitempty" format:"uuid" doc:"Files to leave as they are; a file that trades or chains paths with another leaves both"`
}

type renameInput struct {
	Body struct {
		Works []confirmedWork `json:"works" minItems:"1" nullable:"false" doc:"Each Work to rename as its plan showed"`
	}
}

type renameResult struct {
	Queued  int      `json:"queued" minimum:"0" doc:"Files queued to move to their new paths"`
	Changed []string `json:"changed" format:"uuid" nullable:"false" doc:"Works left as they are because their plans differ now; plan them again"`
}

type renameOutput struct{ Body renameResult }

func (h handlers) renameFiles(ctx context.Context, by identity.Admin, in *renameInput) (*renameOutput, error) {
	cs := make([]files.Confirm, len(in.Body.Works))
	for i, w := range in.Body.Works {
		u, err := storage.ParseUUID(w.Work)
		if err != nil {
			return nil, api.Invalid.Fields(api.FieldError{Location: fmt.Sprintf("body.works[%d].work", i), Message: "expected a Work id"})
		}
		d, err := files.ParseDigest(w.Digest)
		if err != nil {
			return nil, api.Invalid.Fields(api.FieldError{Location: fmt.Sprintf("body.works[%d].digest", i), Message: "expected 16 hexadecimal characters"})
		}
		cs[i] = files.Confirm{Work: works.WorkID{UUID: u}, Digest: d}
		for j, s := range w.Leave {
			id, err := storage.ParseUUID(s)
			if err != nil {
				return nil, api.Invalid.Fields(api.FieldError{Location: fmt.Sprintf("body.works[%d].leave[%d]", i, j), Message: "expected a file id"})
			}
			cs[i].Leave = append(cs[i].Leave, files.ID{UUID: id})
		}
	}
	done, err := open(h.lib).Rename(ctx, by, cs)
	if err != nil {
		return nil, surveyed(err)
	}
	out := renameResult{Queued: done.Queued, Changed: make([]string, len(done.Changed))}
	for i, w := range done.Changed {
		out.Changed[i] = w.String()
	}
	return &renameOutput{Body: out}, nil
}

type unmatchedType struct {
	ContentType string `json:"contentType"`
	Files       int    `json:"files" minimum:"1"`
}

type unmatchedOutput struct {
	Body struct {
		ContentTypes []unmatchedType `json:"contentTypes" nullable:"false" doc:"Each Content Type with such a file"`
	}
}

func (h handlers) countUnmatched(ctx context.Context, by identity.Admin, _ *struct{}) (*unmatchedOutput, error) {
	counts, err := open(h.lib).Unmatched(ctx, by)
	if err != nil {
		return nil, err
	}
	out := &unmatchedOutput{}
	out.Body.ContentTypes = []unmatchedType{}
	for t, n := range counts {
		if n > 0 {
			out.Body.ContentTypes = append(out.Body.ContentTypes, unmatchedType{ContentType: string(t), Files: n})
		}
	}
	slices.SortFunc(out.Body.ContentTypes, func(a, b unmatchedType) int { return cmp.Compare(a.ContentType, b.ContentType) })
	return out, nil
}

type movingInput struct {
	WorkID      string `query:"workId" format:"uuid" doc:"Count only this Work's files"`
	ContentType string `query:"contentType" doc:"Count only this Content Type's files"`
}

type movingOutput struct {
	Body struct {
		Files int `json:"files" minimum:"0"`
	}
}

func (h handlers) countMoving(ctx context.Context, _ identity.Admin, in *movingInput) (*movingOutput, error) {
	s := files.LibraryScope()
	switch {
	case in.WorkID != "" && in.ContentType != "":
		return nil, api.Invalid.Fields(api.FieldError{Location: "query.contentType", Message: "Send workId or contentType, not both."})
	case in.WorkID != "":
		u, err := storage.ParseUUID(in.WorkID)
		if err != nil {
			return nil, workNotFound
		}
		s = files.WorkScope(works.WorkID{UUID: u})
	case in.ContentType != "":
		s = files.TypeScope(works.TypeID(in.ContentType))
	}
	n, err := open(h.lib).Moving(ctx, s)
	out := &movingOutput{}
	out.Body.Files = n
	return out, err
}
