package trackersapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// maxSeed is the longest seeding time a rule or goal takes.
const maxSeed = 3650 * 24 * time.Hour

type ruleBody struct {
	Clears      string   `json:"clears" enum:"time,ratio,either,both" doc:"What meets the rule: its seeding time, its ratio, either of them or both"`
	Ratio       *float64 `json:"ratio" doc:"At most two decimals; null when the rule doesn't count ratio"`
	SeedSeconds *int64   `json:"seedSeconds" doc:"Seconds; null when the rule doesn't count seeding time"`
}

type rule struct {
	Clears      string   `json:"clears" enum:"time,ratio,either,both" doc:"What meets the rule: its seeding time, its ratio, either of them or both"`
	Ratio       *float64 `json:"ratio" doc:"At most two decimals; null when the rule doesn't count ratio"`
	SeedSeconds *int64   `json:"seedSeconds" doc:"Seconds; null when the rule doesn't count seeding time"`
	Source      string   `json:"source" enum:"releases,admin" doc:"releases is the latest rule the Tracker's Releases stated; admin is the rule an Admin set, which Releases never replace"`
}

func ruleOf(rs trackers.Rules) api.Nullable[rule] {
	r, source := rs.InForce()
	if r == nil {
		return api.Nullable[rule]{}
	}
	ratio, seed := partsOf(r.Parts())
	return api.Some(rule{Clears: string(r.Clears()), Ratio: ratio, SeedSeconds: seed, Source: string(source)})
}

// partsSentence is what a rule that clears each way gives.
var partsSentence = map[seeding.Clears]string{
	seeding.ClearsTime:   "Give a seeding time and no ratio for a rule that clears by time.",
	seeding.ClearsRatio:  "Give a ratio and no seeding time for a rule that clears by ratio.",
	seeding.ClearsEither: "Give a ratio and a seeding time for a rule that clears by either.",
	seeding.ClearsBoth:   "Give a ratio and a seeding time for a rule that clears by both.",
}

func parseRule(b ruleBody) (seeding.Rule, error) {
	ratio, err := parseRatio(b.Ratio)
	if err != nil {
		return nil, err
	}
	seed, err := parseSeed(b.SeedSeconds)
	if err != nil {
		return nil, err
	}
	r, err := seeding.RuleOf(seeding.Clears(b.Clears), ratio, seed)
	if err != nil {
		return nil, seedingValueRefused.On("clears", partsSentence[seeding.Clears(b.Clears)])
	}
	return r, nil
}

type goal struct {
	Combine     string   `json:"combine" enum:"first,both,forever" doc:"first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent"`
	Ratio       *float64 `json:"ratio" doc:"At most two decimals; null when combine is forever"`
	SeedSeconds *int64   `json:"seedSeconds" doc:"Seconds; null when combine is forever"`
	Action      *string  `json:"action" enum:"pause,remove,remove-delete" doc:"What happens to the torrent once the goal is met, where remove-delete also deletes its download copy; null when combine is forever"`
}

func goalOf(g seeding.Goal) goal {
	f, ok := g.(seeding.Finite)
	if !ok {
		return goal{Combine: "forever"}
	}
	ratio, seed := partsOf(f.Ratio, f.Seed)
	action := string(f.Then)
	return goal{Combine: string(f.Combine), Ratio: ratio, SeedSeconds: seed, Action: &action}
}

func parseGoal(b goal) (seeding.Goal, error) {
	if b.Combine == "forever" {
		if b.Ratio != nil || b.SeedSeconds != nil || b.Action != nil {
			return nil, seedingValueRefused.On("combine", "Leave out the ratio, seeding time and action of a goal that seeds forever.")
		}
		return seeding.Forever{}, nil
	}
	for _, missing := range []struct {
		absent         bool
		field, mistake string
	}{
		{b.Ratio == nil, "ratio", "Give the goal's ratio."},
		{b.SeedSeconds == nil, "seedSeconds", "Give the goal's seeding time."},
		{b.Action == nil, "action", "Choose what happens to a torrent once the goal is met."},
	} {
		if missing.absent {
			return nil, seedingValueRefused.On(missing.field, missing.mistake)
		}
	}
	ratio, err := parseRatio(b.Ratio)
	if err != nil {
		return nil, err
	}
	seed, err := parseSeed(b.SeedSeconds)
	if err != nil {
		return nil, err
	}
	return seeding.Finite{Combine: seeding.Combine(b.Combine), Ratio: ratio, Seed: seed, Then: seeding.Action(*b.Action)}, nil
}

func parseRatio(v *float64) (seeding.Ratio, error) {
	if v == nil {
		return 0, nil
	}
	r, err := seeding.ParseRatio(*v)
	if err != nil {
		return 0, seedingValueRefused.On("ratio", "Use a ratio of 0.01 or more with at most two decimals.")
	}
	return r, nil
}

func parseSeed(v *int64) (time.Duration, error) {
	if v == nil {
		return 0, nil
	}
	if *v < 1 || *v > int64(maxSeed/time.Second) {
		return 0, seedingValueRefused.On("seedSeconds", fmt.Sprintf("Use a seeding time between 1 second and %d days.", maxSeed/(24*time.Hour)))
	}
	return time.Duration(*v) * time.Second, nil
}

func partsOf(r seeding.Ratio, d time.Duration) (ratio *float64, seed *int64) {
	if r > 0 {
		v := r.Float()
		ratio = &v
	}
	if d > 0 {
		v := int64(d / time.Second)
		seed = &v
	}
	return ratio, seed
}

func seedingAnswer(err error) error {
	if public, ok := errors.AsType[*trackers.PublicError](err); ok {
		return trackerPublic.Because(public.Name + " is public, so its torrents follow your seeding goal alone.")
	}
	return trackerAnswer(err)
}

type ruleInput struct {
	ID   string `path:"id" format:"uuid"`
	Body ruleBody
}

func (h handlers) setRule(ctx context.Context, by identity.Admin, in *ruleInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	r, err := parseRule(in.Body)
	if err != nil {
		return nil, err
	}
	s, err := h.trackers().SetRule(ctx, by, id, r)
	if err != nil {
		return nil, seedingAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(s)}, nil
}

func (h handlers) clearRule(ctx context.Context, by identity.Admin, in *trackerInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	s, err := h.trackers().ClearRule(ctx, by, id)
	if err != nil {
		return nil, seedingAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(s)}, nil
}

type goalInput struct {
	ID   string `path:"id" format:"uuid"`
	Body goal
}

func (h handlers) setGoal(ctx context.Context, by identity.Admin, in *goalInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	g, err := parseGoal(in.Body)
	if err != nil {
		return nil, err
	}
	s, err := h.trackers().SetGoal(ctx, by, id, g)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(s)}, nil
}

func (h handlers) clearGoal(ctx context.Context, by identity.Admin, in *trackerInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	s, err := h.trackers().ClearGoal(ctx, by, id)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(s)}, nil
}

type goalOutput struct{ Body goal }

func (h handlers) getDefaultGoal(ctx context.Context, by identity.Admin, _ *struct{}) (*goalOutput, error) {
	g, err := h.trackers().DefaultGoal(ctx, by)
	if err != nil {
		return nil, err
	}
	return &goalOutput{Body: goalOf(g)}, nil
}

type defaultGoalInput struct{ Body goal }

func (h handlers) setDefaultGoal(ctx context.Context, by identity.Admin, in *defaultGoalInput) (*goalOutput, error) {
	g, err := parseGoal(in.Body)
	if err != nil {
		return nil, err
	}
	if err := h.trackers().SetDefaultGoal(ctx, by, g); err != nil {
		return nil, err
	}
	return h.getDefaultGoal(ctx, by, nil)
}

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The margin added to every rule's seeding time, and how long a torrent that owes may go without announcing before Health warns"`
}

type settingsOutput struct{ Body settings }

type settingsInput struct{ Body settingsChange }

type settingsChange struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return settingsOf(ctx, h.trackers().Settings())
}

func (h handlers) updateSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	p := h.trackers().Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func settingsOf(ctx context.Context, p setting.Panel) (*settingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{Settings: settingapi.FieldsOf(fields)}}, nil
}
