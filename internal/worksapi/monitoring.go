package worksapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type presetCounts struct {
	Asides  int           `json:"asides" doc:"Items in aside levels, which Include adds to a preset"`
	Choices []presetCount `json:"choices" nullable:"false" doc:"One per preset, in the order the presets show"`
}

type presetCount struct {
	Preset      string `json:"preset" enum:"all,future,missing,existing,first,latest,none"`
	Monitors    int    `json:"monitors" doc:"Live Items the preset monitors, asides left out"`
	MonitorsNew bool   `json:"monitorsNew" doc:"Whether the preset monitors Items a refresh adds"`
}

type monitoringInput struct {
	Body monitoringChange
}

type monitoringChange struct {
	WorkIDs       []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
	Preset        string   `json:"preset" enum:"all,future,missing,existing,first,latest,none"`
	IncludeAsides bool     `json:"includeAsides" doc:"Monitor the aside Items the preset picks, and new ones when it monitors new Items"`
	MonitorNew    *bool    `json:"monitorNew,omitempty" doc:"Whether to monitor Items a refresh adds; left out, the preset decides"`
	Preview       bool     `json:"preview" doc:"Answer what the change would move and write nothing"`
}

type monitoringOutcome struct{}

func (monitoringOutcome) Variants() (discriminator string, variants []any) {
	return "outcome", []any{monitoringDoneOutcome{}, monitoringSkippedOutcome{}, failedOutcome{}}
}

type monitoringDoneOutcome struct {
	WorkID  string                     `json:"workId" format:"uuid"`
	Outcome string                     `json:"outcome" enum:"done"`
	Start   int                        `json:"start" doc:"Items the change starts monitoring"`
	Stop    int                        `json:"stop" doc:"Items the change stops monitoring"`
	Counts  api.Nullable[presetCounts] `json:"counts" doc:"What each preset would monitor of the Work; null unless a preview"`
}

type monitoringSkippedOutcome struct {
	WorkID  string                     `json:"workId" format:"uuid"`
	Outcome string                     `json:"outcome" enum:"skipped"`
	Reason  string                     `json:"reason" enum:"already" doc:"Why the change left the Work as it was"`
	Counts  api.Nullable[presetCounts] `json:"counts" doc:"What each preset would monitor of the Work; null unless a preview"`
}

type monitoringList struct {
	Outcomes []api.OneOf[monitoringOutcome] `json:"outcomes" nullable:"false" doc:"One per Work, in request order"`
}

type monitoringListOutput struct{ Body monitoringList }

type workMonitoredInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   workMonitored
}

type workMonitored struct {
	LevelIDs []string `json:"levelIds" format:"uuid" maxItems:"20000" nullable:"false" doc:"Each level stands for every Item under it"`
	ItemIDs  []string `json:"itemIds" format:"uuid" maxItems:"20000" nullable:"false"`
	On       bool     `json:"on" doc:"Monitor the levels and Items, or stop"`
}

type optionsInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   optionsChange
}

type optionsChange struct {
	LevelFolders *bool                      `json:"levelFolders,omitempty" doc:"Whether the Work's files go in a folder per level; left out, it keeps its own"`
	Type         map[string]json.RawMessage `json:"type,omitempty" doc:"The Content Type's own options to change, by key"`
	Preview      bool                       `json:"preview" doc:"Answer what the change would withdraw and write nothing"`
}

type optionsChanged struct {
	LevelFolders bool                          `json:"levelFolders"`
	Options      []api.OneOf[settingapi.Field] `json:"options" nullable:"false" doc:"The Content Type's own options for the Work"`
	Moves        bool                          `json:"moves" doc:"The change fetches the Work again, so its Items may move between levels"`
	Withdraws    int                           `json:"withdraws" doc:"Live Items the change withdraws: the source doesn't list them under the new options, and they keep their files"`
}

type optionsChangedOutput struct{ Body optionsChanged }

func (h handlers) setWorksMonitoring(ctx context.Context, by identity.Admin, in *monitoringInput) (*monitoringListOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	asides := in.Body.IncludeAsides
	c := works.Choice{Preset: works.Preset(in.Body.Preset), Asides: &asides, New: in.Body.MonitorNew}
	os, err := composer(h.shelf).SetMonitoring(ctx, by, ws, c, in.Body.Preview)
	if err != nil {
		return nil, answer(err)
	}
	out := monitoringList{Outcomes: make([]api.OneOf[monitoringOutcome], len(os))}
	for i, o := range os {
		counts := api.Nullable[presetCounts]{}
		if in.Body.Preview {
			counts = api.Some(presetCountsOf(o.Counts))
		}
		id := o.Work.String()
		switch o.Result {
		case library.Done:
			out.Outcomes[i] = api.NewOneOf[monitoringOutcome](monitoringDoneOutcome{WorkID: id, Outcome: "done", Start: o.Delta.Start, Stop: o.Delta.Stop, Counts: counts})
		case library.Skipped:
			out.Outcomes[i] = api.NewOneOf[monitoringOutcome](monitoringSkippedOutcome{WorkID: id, Outcome: "skipped", Reason: reasonOf(o.Why), Counts: counts})
		case library.Failed:
			p, err := problemOf(o.Why)
			if err != nil {
				return nil, err
			}
			out.Outcomes[i] = api.NewOneOf[monitoringOutcome](failedOutcome{WorkID: id, Outcome: "failed", Problem: p})
		}
	}
	return &monitoringListOutput{Body: out}, nil
}

func presetCountsOf(c works.PresetCounts) presetCounts {
	out := presetCounts{Asides: c.Asides, Choices: make([]presetCount, len(c.Choices))}
	for i, p := range c.Choices {
		out.Choices[i] = presetCount{Preset: string(p.Preset), Monitors: p.Monitors, MonitorsNew: p.MonitorsNew}
	}
	return out
}

func (h handlers) setWorkMonitored(ctx context.Context, by identity.Admin, in *workMonitoredInput) (*struct{}, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	if len(in.Body.LevelIDs) == 0 && len(in.Body.ItemIDs) == 0 {
		return nil, api.Invalid.On("levelIds", "Send at least one level or Item id.")
	}
	var part works.Within
	for _, s := range in.Body.LevelIDs {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, api.Invalid
		}
		part.Levels = append(part.Levels, works.LevelID{UUID: u})
	}
	for _, s := range in.Body.ItemIDs {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, api.Invalid
		}
		part.Items = append(part.Items, works.ItemID{UUID: u})
	}
	err = composer(h.shelf).SetMonitored(ctx, by, id, part, in.Body.On)
	if e, ok := errors.AsType[*works.NotInWorkError](err); ok {
		field, noun := "itemIds", "Item"
		if e.Level {
			field, noun = "levelIds", "level"
		}
		return nil, api.Invalid.On(fmt.Sprintf("%s[%d]", field, e.Index), "The Work has no "+noun+" with that id.")
	}
	if err != nil {
		return nil, answer(err)
	}
	return nil, nil
}

func (h handlers) setWorkOptions(ctx context.Context, by identity.Admin, in *optionsInput) (*optionsChangedOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	o := works.Optioning{LevelFolders: in.Body.LevelFolders, Type: in.Body.Type}
	c, err := composer(h.shelf).SetOptions(ctx, by, id, o, in.Body.Preview)
	if err != nil {
		return nil, settingapi.Refused(answer(err))
	}
	return &optionsChangedOutput{Body: optionsChanged{LevelFolders: c.LevelFolders, Options: settingapi.FieldsOf(c.Fields), Moves: c.Moves, Withdraws: c.Withdraws}}, nil
}
