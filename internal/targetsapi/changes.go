package targetsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	impactChanged    = api.NewCarrying[Impact](http.StatusConflict, "impact_changed", "The change now does more than its preview showed", "impact")
	ImpactChanged    = impactChanged.Code
	profileIsDefault = api.NewCode(http.StatusConflict, "profile_is_default", "The default Profile can't be deleted")
	profileWrongType = api.NewCode(http.StatusUnprocessableEntity, "profile_wrong_type", "The Profile is for another Content Type")
	profileNotFound  = api.NewCode(http.StatusUnprocessableEntity, "profile_not_found", "No Profile has that id")
)

type unwantedFiles struct {
	Files int   `json:"files" minimum:"0"`
	Bytes int64 `json:"bytes" minimum:"0"`
}

type Impact struct {
	Works       int           `json:"works" minimum:"0" doc:"Works whose Targets, files or file names change"`
	OwnList     int           `json:"ownList" minimum:"0" doc:"Works that keep their own Edition list through a change to their Profile's"`
	Unwanted    unwantedFiles `json:"unwanted" doc:"Files a Target uses now and none uses after: deleted, or kept as Kept files, as the save chooses"`
	Renamed     int           `json:"renamed" minimum:"0" doc:"Files whose path changes, Kept files included"`
	Renames     []workRenames `json:"renames" nullable:"false" doc:"The first 500 of the renamed files, by Work in title order"`
	NewlyWanted int           `json:"newlyWanted" minimum:"0" doc:"How many more files SlipStream will look for"`
	Downloading int           `json:"downloading" minimum:"0" doc:"Downloads in progress that are no longer needed. Each waits in Pending imports once it finishes"`
	Subtitles   subtitles     `json:"subtitles" doc:"What a change of a Profile's subtitle languages does"`
	Made        madeImpact    `json:"made" doc:"The files a change of a Profile's made-from Targets has SlipStream make"`
	Remake      remakeImpact  `json:"remake" doc:"The made files a Target's older Conversion settings made"`
}

type subtitles struct {
	Gained  []gainedLanguage  `json:"gained" nullable:"false" doc:"Each subtitle language the change adds, in the Profile's order"`
	Dropped []droppedLanguage `json:"dropped" nullable:"false" doc:"Each subtitle language the change removes"`
	Days    int               `json:"days" minimum:"0" doc:"About how many days the gained searches take at the providers' last reported download allowance, 0 when none reported one"`
}

type gainedLanguage struct {
	Language string `json:"language"`
	Forced   bool   `json:"forced"`
	Searches int    `json:"searches" minimum:"0" doc:"Files SlipStream searches subtitles for"`
	NeedScan int    `json:"needScan" minimum:"0" doc:"Of those, files whose audio is scanned first"`
}

type droppedLanguage struct {
	Language string `json:"language"`
	Forced   bool   `json:"forced"`
	Deleted  int    `json:"deleted" minimum:"0" doc:"Subtitles SlipStream downloaded that the save deletes. Uploads and subtitles found beside the files stay"`
}

func subtitlesOf(s files.SubtitlesImpact) subtitles {
	out := subtitles{Gained: []gainedLanguage{}, Dropped: []droppedLanguage{}, Days: s.Days}
	for _, g := range s.Gained {
		out.Gained = append(out.Gained, gainedLanguage{Language: g.Language.String(), Forced: g.Forced, Searches: g.Files, NeedScan: g.NeedScan})
	}
	for _, d := range s.Dropped {
		out.Dropped = append(out.Dropped, droppedLanguage{Language: d.Language.String(), Forced: d.Forced, Deleted: d.Deleted})
	}
	return out
}

type workRenames struct {
	WorkID string        `json:"workId" format:"uuid"`
	Title  string        `json:"title"`
	Files  []renamedFile `json:"files" nullable:"false" doc:"In the order the files were added"`
}

type renamedFile struct {
	FileID string `json:"fileId" format:"uuid"`
	From   string `json:"from" doc:"The file's path in its root now"`
	To     string `json:"to" doc:"The file's path in its root after the change"`
}

func ImpactOf(i files.Impact) Impact {
	renames := make([]workRenames, len(i.Renames))
	for n, w := range i.Renames {
		renames[n] = workRenames{WorkID: w.Work.String(), Title: w.Title, Files: make([]renamedFile, len(w.Rows))}
		for k, r := range w.Rows {
			renames[n].Files[k] = renamedFile{FileID: r.File.String(), From: r.From, To: r.To}
		}
	}
	return Impact{
		Works: i.Works, OwnList: i.OwnList, Unwanted: unwantedFiles{Files: i.Unwanted, Bytes: i.Bytes},
		Renamed: i.Renamed, Renames: renames, NewlyWanted: i.NewlyWanted, Downloading: i.Downloading, Subtitles: subtitlesOf(i.Subtitles),
		Made: madeOf(i.Made), Remake: remakeOf(i.Remake),
	}
}

// Changing is the query every change that can leave files unwanted takes.
// It is exported so the document carries the parameters it embeds.
type Changing struct {
	Preview     bool                `query:"preview" doc:"Return the change's Impact and save nothing"`
	Unwanted    string              `query:"unwanted" enum:"delete,keep" default:"delete" doc:"What happens to the files no Target uses after the change"`
	Confirm     api.Optional[int64] `query:"confirm" minimum:"0" doc:"The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed"`
	ConfirmMade api.Optional[int64] `query:"confirmMade" minimum:"0" doc:"The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed"`
}

func (c Changing) Mode() files.Mode {
	m := files.Mode{Preview: c.Preview, Unwanted: files.Unwanted(c.Unwanted)}
	if c.Confirm.Set {
		m.Confirmed = new(int(c.Confirm.Value))
	}
	if c.ConfirmMade.Set {
		m.ConfirmedMade = new(int(c.ConfirmMade.Value))
	}
	return m
}

// changed answers err from a Change.
func changed(err error) error {
	if e, ok := errors.AsType[*files.ImpactChangedError](err); ok {
		return ImpactRefused(e)
	}
	return answer(err)
}

// ImpactRefused is the problem a save that would leave more files
// unwanted, delete more subtitles, or make or remake more files than its
// preview showed is refused with. It carries the Impact the save has now.
func ImpactRefused(e *files.ImpactChangedError) error {
	var words []string
	if e.Files {
		words = append(words, fmt.Sprintf("The save now leaves %s unwanted, and the preview showed %d.", filesOf(e.Impact.Unwanted), e.Confirmed))
	}
	if e.Subtitles {
		words = append(words, fmt.Sprintf("The save now deletes %s, and the preview showed %d.", subtitlesWords(e.Impact.Subtitles.Deleted()), e.ConfirmedSubtitles))
	}
	if e.Made {
		words = append(words, fmt.Sprintf("The save now makes %s, and the preview showed %d.", filesOf(e.Impact.Made.Files), e.ConfirmedMade))
	}
	if e.Remake {
		words = append(words, fmt.Sprintf("The save now makes %s again, and the preview showed %d.", filesOf(e.Impact.Remake.Files()), e.ConfirmedRemake))
	}
	return impactChanged.Explained(strings.Join(append(words, "Preview the change again."), " "), ImpactOf(e.Impact))
}

func subtitlesWords(n int) string {
	if n == 1 {
		return "1 subtitle"
	}
	return fmt.Sprintf("%d subtitles", n)
}

func filesOf(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

type updateTargetInput struct {
	TargetID      string              `path:"targetId" format:"uuid"`
	ConfirmRemake api.Optional[int64] `query:"confirmRemake" minimum:"0" doc:"The remade file count the Admin confirmed from the preview: a save that would remake more is refused with impact_changed"`
	Changing
	Body newTarget
}

type targetChange struct {
	Impact Impact `json:"impact"`
	Target target `json:"target"`
}

type targetChangeOutput struct{ Body targetChange }

func (h handlers) updateTarget(ctx context.Context, by identity.Admin, in *updateTargetInput) (*targetChangeOutput, error) {
	id, err := targetID(in.TargetID)
	if err != nil {
		return nil, err
	}
	s, err := h.specOf(in.Body)
	if err != nil {
		return nil, err
	}
	m := in.Mode()
	if in.ConfirmRemake.Set {
		m.ConfirmedRemake = new(int(in.ConfirmRemake.Value))
	}
	if m.Remake, err = workIDs("remake", in.Body.Remake); err != nil {
		return nil, err
	}
	i, err := shelf(h.goals).Change(ctx, by, targets.UpdateTarget{ID: id, Spec: s}, m)
	switch {
	case errors.Is(err, targets.ErrNoTarget):
		return nil, noSuchTarget
	case err != nil:
		return nil, changed(err)
	}
	t, err := open(h.goals).GetTarget(ctx, by, id)
	if err != nil {
		return nil, err
	}
	view, err := h.targetOf(ctx, t)
	if err != nil {
		return nil, err
	}
	impact, err := h.impactOf(ctx, i)
	if err != nil {
		return nil, err
	}
	return &targetChangeOutput{Body: targetChange{Impact: impact, Target: view}}, nil
}

type updateProfileInput struct {
	ProfileID        string              `path:"profileId" format:"uuid"`
	ConfirmSubtitles api.Optional[int64] `query:"confirmSubtitles" minimum:"0" doc:"The deleted subtitle count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed"`
	Changing
	Body newProfile
}

type profileChange struct {
	Impact  Impact  `json:"impact"`
	Profile Profile `json:"profile"`
}

type profileChangeOutput struct{ Body profileChange }

func (h handlers) updateProfile(ctx context.Context, by identity.Admin, in *updateProfileInput) (*profileChangeOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	s, err := profileSpecOf(in.Body)
	if err != nil {
		return nil, err
	}
	m := in.Mode()
	if in.ConfirmSubtitles.Set {
		m.ConfirmedSubtitles = new(int(in.ConfirmSubtitles.Value))
	}
	c, err := h.profileChange(ctx, by, id, s, in.Body.Backfill)
	if err != nil {
		return nil, err
	}
	i, err := shelf(h.goals).Change(ctx, by, c, m)
	if err != nil {
		return nil, wrongType(changed(err))
	}
	return h.profileChanged(ctx, by, i, id)
}

type deleteProfileInput struct {
	ProfileID string `path:"profileId" format:"uuid"`
	MoveTo    string `query:"moveTo" format:"uuid" required:"true" doc:"The Profile of the same Content Type that its Works move to"`
	Changing
}

func (h handlers) deleteProfile(ctx context.Context, by identity.Admin, in *deleteProfileInput) (*profileChangeOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	u, err := storage.ParseUUID(in.MoveTo)
	if err != nil {
		return nil, profileNotFound
	}
	to := targets.ProfileID{UUID: u}
	i, err := shelf(h.goals).Change(ctx, by, targets.DeleteProfile{ID: id, To: to}, in.Mode())
	switch {
	case errors.Is(err, targets.ErrNoReplacement):
		return nil, profileNotFound
	case errors.Is(err, targets.ErrProfileIsDefault):
		return nil, profileIsDefault
	case errors.Is(err, targets.ErrMovesToItself):
		return nil, api.Invalid.Fields(api.FieldError{Location: "query.moveTo", Message: "Choose a Profile other than the one you're deleting."})
	case err != nil:
		return nil, wrongType(changed(err))
	}
	return h.profileChanged(ctx, by, i, to)
}

func (h handlers) profileChanged(ctx context.Context, by identity.Admin, i files.Impact, id targets.ProfileID) (*profileChangeOutput, error) {
	p, err := open(h.goals).GetProfile(ctx, by, id)
	if err != nil {
		return nil, err
	}
	impact, err := h.impactOf(ctx, i)
	if err != nil {
		return nil, err
	}
	return &profileChangeOutput{Body: profileChange{Impact: impact, Profile: profileOf(p)}}, nil
}

// wrongType answers a Profile of another Content Type, which the body or
// query names.
func wrongType(err error) error {
	if errors.Is(err, targets.ErrProfileWrongType) {
		return profileWrongType
	}
	return err
}

type newWants struct {
	Editions api.Patch[api.OneOf[editions]] `json:"editions" doc:"The Work's own Edition list, or null for its Profile's"`
}

type setWantsInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Changing
	Body newWants
}

type wantsChange struct {
	Impact Impact `json:"impact"`
	Wants  wants  `json:"wants"`
}

type wantsChangeOutput struct{ Body wantsChange }

func (h handlers) setWorkWants(ctx context.Context, by identity.Admin, in *setWantsInput) (*wantsChangeOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	work := works.WorkID{UUID: u}
	c := targets.SetWants{Work: work}
	if own := in.Body.Editions.Value(); own != nil {
		list, err := editionsFrom(*own)
		if err != nil {
			return nil, err
		}
		c.Own = &list
	}
	i, err := shelf(h.goals).Change(ctx, by, c, in.Mode())
	if err != nil {
		return nil, changed(err)
	}
	g, err := open(h.goals).Goals(ctx, by, work)
	if err != nil {
		return nil, err
	}
	impact, err := h.impactOf(ctx, i)
	if err != nil {
		return nil, err
	}
	return &wantsChangeOutput{Body: wantsChange{Impact: impact, Wants: wantsOf(g)}}, nil
}

// impactOf is i on the wire, with the Works of its made rows named.
func (h handlers) impactOf(ctx context.Context, i files.Impact) (Impact, error) {
	out := ImpactOf(i)
	err := h.titled(ctx, &out)
	return out, err
}

func workIDs(field string, raw []string) ([]works.WorkID, error) {
	var out []works.WorkID
	for k, s := range raw {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, api.Invalid.On(fmt.Sprintf("%s[%d]", field, k), "No Work has this id.")
		}
		out = append(out, works.WorkID{UUID: u})
	}
	return out, nil
}

// profileChange is the Change a Profile save makes: an UpdateProfile,
// or with a backfill a SplitProfile whose rest are the Works the save
// would make files for that the backfill doesn't convert.
func (h handlers) profileChange(ctx context.Context, by identity.Admin, id targets.ProfileID, s targets.ProfileSpec, b *backfill) (targets.Change, error) {
	update := targets.UpdateProfile{ID: id, Spec: s}
	if b == nil {
		return update, nil
	}
	convert, err := workIDs("backfill.convert", b.Convert)
	if err != nil {
		return nil, err
	}
	split := targets.SplitProfile{Profile: id, Spec: s, To: targets.ProfileTarget{NewName: b.Rest.NewName}}
	switch {
	case b.Rest.ProfileID != "":
		to, err := profileID(b.Rest.ProfileID)
		if err != nil {
			return nil, profileNotFound
		}
		split.To.Profile = &to
	case b.Rest.NewName == "":
		return nil, api.Invalid.On("backfill.rest", "Choose a Profile for the other Works, or name a new one.")
	}
	if b.Default != "" {
		d, err := profileID(b.Default)
		if err != nil {
			return nil, profileNotFound
		}
		split.Default = &d
	}
	preview, err := shelf(h.goals).Change(ctx, by, update, files.Mode{Preview: true})
	if err != nil {
		return nil, wrongType(changed(err))
	}
	for _, r := range preview.Made.Rows {
		if !slices.Contains(convert, r.Work) {
			split.Rest = append(split.Rest, r.Work)
		}
	}
	return split, nil
}
