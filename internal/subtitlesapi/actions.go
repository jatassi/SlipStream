package subtitlesapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// uploadBody is 7 MiB: a 5 MiB subtitle file in base64, and its name.
const uploadBody = 7 << 20

type wantInput struct {
	FileID string `path:"fileId" format:"uuid"`
	Want   string `path:"want" doc:"en, pt-BR or en-forced"`
}

type searchInput struct {
	FileID   string `path:"fileId" format:"uuid"`
	Want     string `path:"want" doc:"en, pt-BR or en-forced"`
	SearchID string `path:"searchId" format:"uuid"`
}

type pickInput struct {
	FileID   string `path:"fileId" format:"uuid"`
	Want     string `path:"want" doc:"en, pt-BR or en-forced"`
	SearchID string `path:"searchId" format:"uuid"`
	Key      string `path:"key" doc:"The candidate's key, from the search"`
}

type upload struct {
	Name  string `json:"name" minLength:"1" maxLength:"255" doc:"The file's name, whose extension says its format"`
	Bytes []byte `json:"bytes" doc:"The file, in base64"`
}

type uploadInput struct {
	FileID string `path:"fileId" format:"uuid"`
	Want   string `path:"want" doc:"en, pt-BR or en-forced"`
	Body   upload
}

type attemptInput struct {
	FileID    string `path:"fileId" format:"uuid"`
	AttemptID string `path:"attemptId" format:"uuid"`
}

type keepInput struct {
	FileID   string `path:"fileId" format:"uuid"`
	UploadID string `path:"uploadId" format:"uuid"`
}

type started struct {
	SearchID string `json:"searchId" format:"uuid"`
}

type startedOutput struct{ Body started }

type attempting struct {
	AttemptID string `json:"attemptId" format:"uuid" doc:"Read its steps with getSubtitleAttempt, which subtitles.attempt refreshes"`
}

type attemptingOutput struct{ Body attempting }

var reasonNames = map[subtitles.Reason]string{
	subtitles.MadeByPerson: "made_by_person", subtitles.SameGroup: "same_group", subtitles.SameSource: "same_source", subtitles.ProviderOrder: "provider_order",
}

type tried struct {
	Outcome string            `json:"outcome" enum:"passed,failed_timing,marked_bad,unreadable,image,wrong_episode,too_large"`
	Match   api.Nullable[int] `json:"match" doc:"Its timing match, in percent"`
	At      time.Time         `json:"at"`
}

type candidate struct {
	Key      string              `json:"key"`
	Provider string              `json:"provider"`
	Release  string              `json:"release"`
	Reasons  []string            `json:"reasons" nullable:"false" enum:"made_by_person,same_group,same_source,provider_order" doc:"The ranking steps it meets"`
	Tried    api.Nullable[tried] `json:"tried" doc:"Its earlier try for this file and want"`
	Disabled bool                `json:"disabled" doc:"It can't be picked: image subtitles, marked bad, or its provider is out"`
}

type outcome struct {
	Provider string                  `json:"provider"`
	Kind     string                  `json:"kind" enum:"ok,not_found,refused,limited,capped,unreachable"`
	Until    api.Nullable[time.Time] `json:"until" doc:"When a limited or capped provider takes calls again"`
}

type candidates struct {
	State      string      `json:"state" enum:"searching,done"`
	Candidates []candidate `json:"candidates" nullable:"false" doc:"Best first"`
	Outcomes   []outcome   `json:"outcomes" nullable:"false" doc:"How each provider's search ended"`
}

type candidatesOutput struct{ Body candidates }

var stepNames = map[subtitles.Step]string{subtitles.Downloading: "downloading", subtitles.Checking: "checking", subtitles.Finished: "finished"}

var endedNames = map[subtitles.Ended]string{
	subtitles.Added: "added", subtitles.FailedTiming: "failed_timing", subtitles.Unreadable: "unreadable", subtitles.Image: "image",
	subtitles.WrongEpisode: "wrong_episode", subtitles.MarkedBad: "marked_bad",
}

type ended struct {
	Kind      string            `json:"kind" enum:"added,failed_timing,unreadable,image,wrong_episode,marked_bad"`
	Match     api.Nullable[int] `json:"match" doc:"The timing match, in percent"`
	Threshold api.Nullable[int] `json:"threshold" doc:"The timing match a subtitle needs, in percent"`
}

type attempt struct {
	Step     string               `json:"step" enum:"downloading,checking,finished"`
	Provider api.Nullable[string] `json:"provider"`
	Outcome  api.Nullable[ended]  `json:"outcome" doc:"How it ended; null until it finished"`
	UploadID api.Nullable[string] `json:"uploadId" format:"uuid" doc:"A refused upload SlipStream can keep anyway"`
}

type attemptOutput struct{ Body attempt }

var outcomeNames = map[calls.OutcomeKind]string{
	calls.OK: "ok", calls.NotFound: "not_found", calls.Refused: "refused", calls.Limited: "limited", calls.Capped: "capped", calls.Unreachable: "unreachable",
}

func candidatesOf(c subtitles.Candidates) candidates {
	out := candidates{State: "searching", Candidates: []candidate{}, Outcomes: []outcome{}}
	if c.Done {
		out.State = "done"
	}
	for _, x := range c.Candidates {
		cd := candidate{Key: x.Key, Provider: string(x.Provider), Release: x.Release, Reasons: []string{}, Disabled: x.Disabled}
		for _, r := range x.Reasons {
			cd.Reasons = append(cd.Reasons, reasonNames[r])
		}
		if t := x.Tried; t != nil {
			tr := tried{Outcome: string(t.Outcome), At: t.At}
			if t.Match > 0 {
				tr.Match = api.Some(t.Match)
			}
			cd.Tried = api.Some(tr)
		}
		out.Candidates = append(out.Candidates, cd)
	}
	for _, o := range c.Outcomes {
		oc := outcome{Provider: string(o.Provider), Kind: outcomeNames[o.Kind]}
		if !o.Until.IsZero() {
			oc.Until = api.Some(o.Until)
		}
		out.Outcomes = append(out.Outcomes, oc)
	}
	return out
}

func attemptOf(a subtitles.Attempt) attempt {
	out := attempt{Step: stepNames[a.Step], Provider: some(string(a.Provider))}
	if a.Step == subtitles.Finished {
		e := ended{Kind: endedNames[a.Ended]}
		if a.Match > 0 {
			e.Match = api.Some(a.Match)
		}
		if a.Threshold > 0 {
			e.Threshold = api.Some(a.Threshold)
		}
		out.Outcome = api.Some(e)
	}
	if !a.Upload.IsZero() {
		out.UploadID = api.Some(a.Upload.String())
	}
	return out
}

func (h handlers) search(ctx context.Context, by identity.Admin, in *wantInput) (*struct{}, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.o).SearchNowFor(ctx, by, k))
}

func (h handlers) startCandidates(ctx context.Context, by identity.Admin, in *wantInput) (*startedOutput, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	id, err := open(h.o).StartCandidates(ctx, by, k)
	if err != nil {
		return nil, answer(err)
	}
	return &startedOutput{Body: started{SearchID: id.String()}}, nil
}

func (h handlers) candidates(ctx context.Context, by identity.Admin, in *searchInput) (*candidatesOutput, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	id, err := uuidOf(in.SearchID, searchExpired, "These results have expired. Search again.")
	if err != nil {
		return nil, err
	}
	c, err := open(h.o).Candidates(ctx, by, k, id)
	if err != nil {
		return nil, answer(err)
	}
	return &candidatesOutput{Body: candidatesOf(c)}, nil
}

func (h handlers) pick(ctx context.Context, by identity.Admin, in *pickInput) (*attemptingOutput, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	id, err := uuidOf(in.SearchID, searchExpired, "These results have expired. Search again.")
	if err != nil {
		return nil, err
	}
	a, err := open(h.o).Pick(ctx, by, k, id, in.Key)
	return attemptingOf(a, err)
}

func (h handlers) upload(ctx context.Context, by identity.Admin, in *uploadInput) (*attemptingOutput, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	a, err := open(h.o).Upload(ctx, by, k, in.Body.Name, in.Body.Bytes)
	return attemptingOf(a, err)
}

func attemptingOf(a storage.UUID, err error) (*attemptingOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	return &attemptingOutput{Body: attempting{AttemptID: a.String()}}, nil
}

func (h handlers) attempt(ctx context.Context, by identity.Admin, in *attemptInput) (*attemptOutput, error) {
	f, err := fileID(in.FileID)
	if err != nil {
		return nil, err
	}
	id, err := uuidOf(in.AttemptID, attemptExpired, "This download has expired. Search again.")
	if err != nil {
		return nil, err
	}
	a, err := open(h.o).AttemptOf(ctx, by, f, id)
	if err != nil {
		return nil, answer(err)
	}
	return &attemptOutput{Body: attemptOf(a)}, nil
}

func (h handlers) keep(ctx context.Context, by identity.Admin, in *keepInput) (*struct{}, error) {
	f, err := fileID(in.FileID)
	if err != nil {
		return nil, err
	}
	id, err := uuidOf(in.UploadID, uploadExpired, "This upload has expired. Upload the file again.")
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.o).KeepUpload(ctx, by, f, id))
}

func (h handlers) markBad(ctx context.Context, by identity.Admin, in *wantInput) (*struct{}, error) {
	k, err := wantKey(in.FileID, in.Want)
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.o).MarkBadNow(ctx, by, k))
}

// Routes registers every Subtitles operation.
func Routes(r *api.Router, owner Owner) {
	o := r.Owner("subtitles", api.Account)
	h := handlers{owner}
	api.Register(o, api.Admin, api.Op{
		ID: "getSubtitleSettings", Method: http.MethodGet, Path: "/subtitles/settings",
		Summary: "Get the Subtitles settings, the section's lead and the Profiles with Wanted Languages", Status: http.StatusOK,
	}, h.getSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "updateSubtitleSettings", Method: http.MethodPatch, Path: "/subtitles/settings",
		Summary: "Change Subtitles settings by key, all or none", Status: http.StatusOK,
		Errors: []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown, gapTooShort, notBuilt},
	}, h.updateSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "getSubtitleNodeSettings", Method: http.MethodGet, Path: "/subtitles/nodes/{nodeId}/settings",
		Summary: "Get a Storage Node's Subtitles settings", Status: http.StatusOK, Errors: []*api.Code{notStorage},
	}, h.getNodeSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "updateSubtitleNodeSettings", Method: http.MethodPatch, Path: "/subtitles/nodes/{nodeId}/settings",
		Summary: "Change a Storage Node's Subtitles settings by key, all or none", Status: http.StatusOK,
		Errors: []*api.Code{notStorage, settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateNodeSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "listSubtitleProviders", Method: http.MethodGet, Path: "/subtitles/providers",
		Summary: "List the Subtitle Providers in search order", Status: http.StatusOK,
	}, func(ctx context.Context, by identity.Admin, _ *struct{}) (*providerListOutput, error) {
		return h.list(ctx, by)
	})
	api.Register(o, api.Admin, api.Op{
		ID: "updateSubtitleProvider", Method: http.MethodPatch, Path: "/subtitles/providers/{id}",
		Summary: "Turn a Subtitle Provider on or off", Status: http.StatusOK, Errors: []*api.Code{providerNotFound},
	}, func(ctx context.Context, by identity.Admin, in *providerOnInput) (*providerOutput, error) {
		return one(open(h.o).SetProviderOn(ctx, by, subtitling.ProviderID(in.ID), in.Body.On))
	})
	api.Register(o, api.Admin, api.Op{
		ID: "orderSubtitleProviders", Method: http.MethodPut, Path: "/subtitles/providers/order",
		Summary: "Set the Subtitle Providers' search order", Status: http.StatusOK, Errors: []*api.Code{orderMismatch},
	}, func(ctx context.Context, by identity.Admin, in *orderInput) (*providerListOutput, error) {
		ids := make([]subtitling.ProviderID, len(in.Body.IDs))
		for i, id := range in.Body.IDs {
			ids[i] = subtitling.ProviderID(id)
		}
		ps, err := open(h.o).OrderProviders(ctx, by, ids)
		if err != nil {
			return nil, answer(err)
		}
		return listOf(ps), nil
	})
	api.Register(o, api.Admin, api.Op{
		ID: "setSubtitleProviderKey", Method: http.MethodPut, Path: "/subtitles/providers/{id}/key",
		Summary: "Check a Subtitle Provider key on a Subtitles holder, then save it", Status: http.StatusOK,
		Errors: []*api.Code{providerNotFound, keyless, keyRejected, keyUnchecked, noHolder},
	}, func(ctx context.Context, by identity.Admin, in *keyInput) (*providerOutput, error) {
		return one(open(h.o).SetKey(ctx, by, subtitling.ProviderID(in.ID), in.Body.Key))
	})
	api.Register(o, api.Admin, api.Op{
		ID: "deleteSubtitleProviderKey", Method: http.MethodDelete, Path: "/subtitles/providers/{id}/key",
		Summary: "Forget a Subtitle Provider's key", Status: http.StatusOK, Errors: []*api.Code{providerNotFound, keyless},
	}, func(ctx context.Context, by identity.Admin, in *providerInput) (*providerOutput, error) {
		return one(open(h.o).DeleteKey(ctx, by, subtitling.ProviderID(in.ID)))
	})
	api.Register(o, api.Admin, api.Op{
		ID: "getWorkSubtitles", Method: http.MethodGet, Path: "/works/{workId}/subtitles",
		Summary: "Get each library file of a Work with what each of its wanted subtitles reads as", Status: http.StatusOK, Errors: []*api.Code{workNotFound},
	}, h.workSubtitles)
	api.Register(o, api.Admin, api.Op{
		ID: "ignoreSubtitleTrack", Method: http.MethodPut, Path: "/files/{fileId}/subtitles/tracks/{stream}/ignored",
		Summary: "Don't count an embedded subtitle track toward the file's wants", Status: http.StatusNoContent, Errors: []*api.Code{fileNotFound, trackNotFound},
	}, h.ignore(true))
	api.Register(o, api.Admin, api.Op{
		ID: "countSubtitleTrack", Method: http.MethodDelete, Path: "/files/{fileId}/subtitles/tracks/{stream}/ignored",
		Summary: "Count an embedded subtitle track toward the file's wants again", Status: http.StatusNoContent, Errors: []*api.Code{fileNotFound, trackNotFound},
	}, h.ignore(false))
	want := "/files/{fileId}/subtitles/{want}"
	wanting := []*api.Code{fileNotFound, badWant, notBuilt}
	api.Register(o, api.Admin, api.Op{
		ID: "searchSubtitle", Method: http.MethodPost, Path: want + "/search",
		Summary: "Search for one file's wanted subtitle now, ahead of the schedule", Status: http.StatusAccepted, Errors: wanting,
	}, h.search)
	api.Register(o, api.Admin, api.Op{
		ID: "startSubtitleCandidates", Method: http.MethodPost, Path: want + "/candidates",
		Summary: "Start a Manual subtitle search on one file's wanted subtitle", Status: http.StatusAccepted, Errors: wanting,
	}, h.startCandidates)
	api.Register(o, api.Admin, api.Op{
		ID: "getSubtitleCandidates", Method: http.MethodGet, Path: want + "/candidates/{searchId}",
		Summary: "Get a Manual subtitle search's ranked candidates so far", Status: http.StatusOK, Errors: append(wanting, searchExpired),
	}, h.candidates)
	api.Register(o, api.Admin, api.Op{
		ID: "pickSubtitleCandidate", Method: http.MethodPost, Path: want + "/candidates/{searchId}/{key}",
		Summary: "Download a candidate and check its timing against the file", Status: http.StatusAccepted, Errors: append(wanting, searchExpired),
	}, h.pick)
	api.Register(o, api.Admin, api.Op{
		ID: "uploadSubtitle", Method: http.MethodPost, Path: want + "/upload",
		Summary: "Upload a subtitle file and check its timing against the file", Status: http.StatusAccepted, MaxBody: uploadBody,
		Errors: append(wanting, tooLarge, image),
	}, h.upload)
	api.Register(o, api.Admin, api.Op{
		ID: "markSubtitleBad", Method: http.MethodPost, Path: want + "/mark-bad",
		Summary: "Delete a file's subtitle, never use it for the file again and search for another", Status: http.StatusAccepted, Errors: wanting,
	}, h.markBad)
	api.Register(o, api.Admin, api.Op{
		ID: "getSubtitleAttempt", Method: http.MethodGet, Path: "/files/{fileId}/subtitles/attempts/{attemptId}",
		Summary: "Get a subtitle download's or upload's step, and its outcome once finished", Status: http.StatusOK,
		Errors: []*api.Code{fileNotFound, attemptExpired, notBuilt},
	}, h.attempt)
	api.Register(o, api.Admin, api.Op{
		ID: "keepSubtitleUpload", Method: http.MethodPost, Path: "/files/{fileId}/subtitles/uploads/{uploadId}/keep",
		Summary: "Keep a refused upload without a timing check", Status: http.StatusNoContent,
		Errors: []*api.Code{fileNotFound, uploadExpired, notBuilt},
	}, h.keep)
}
