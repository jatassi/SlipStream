package subtitlesapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
)

var kindNames = map[subtitling.Kind]string{subtitling.Feature: "feature", subtitling.Episode: "episode"}

var standings = map[subtitles.Standing]string{
	subtitles.Off: "off", subtitles.NeedsKey: "needs_key", subtitles.Rejected: "rejected", subtitles.Out: "out", subtitles.Failing: "failing", subtitles.Ready: "ready",
}

type window struct {
	Used      int       `json:"used" minimum:"0"`
	Limit     int       `json:"limit" minimum:"0"`
	Remaining int       `json:"remaining" minimum:"0"`
	Resets    time.Time `json:"resets"`
	Exact     bool      `json:"exact" doc:"Whether the provider counts the window up to the last call, or reports it with a lag"`
}

type allowance struct {
	Downloads api.Nullable[window] `json:"downloads" doc:"Null when the provider reported no download count"`
	Searches  api.Nullable[window] `json:"searches" doc:"Null when the provider reported no search count"`
	At        time.Time            `json:"at" doc:"When the provider reported it"`
}

type standing struct {
	Kind  string                  `json:"kind" enum:"ready,needs_key,out,rejected,failing,off" doc:"The first that applies: off, a key is needed, the key was rejected, out of downloads or rate limited, failing, ready"`
	Since api.Nullable[time.Time] `json:"since" doc:"When the provider rejected the key, or when a failing provider started failing"`
	Until api.Nullable[time.Time] `json:"until" doc:"When a provider that is out takes calls again"`
	Words api.Nullable[string]    `json:"words" doc:"The provider's own words for a rejected key, or the last error of a failing provider"`
}

type provider struct {
	ID        string                  `json:"id" doc:"The provider, such as subdl"`
	Name      string                  `json:"name"`
	Kinds     []string                `json:"kinds" nullable:"false" enum:"feature,episode" doc:"What it searches: films by their ids, episodes by their series and Aired numbers"`
	Position  int                     `json:"position" minimum:"1" doc:"Its place in the search order"`
	On        bool                    `json:"on"`
	NeedsKey  bool                    `json:"needsKey" doc:"The provider needs the Admin's key"`
	HasKey    bool                    `json:"hasKey"`
	KeyPage   api.Nullable[string]    `json:"keyPage" format:"uri" doc:"Where an Admin gets a key; null for a provider that needs none"`
	Account   api.Nullable[string]    `json:"account" doc:"The account the key belongs to, as the provider names it"`
	Plan      api.Nullable[string]    `json:"plan" doc:"The account's plan, as the provider names it"`
	Allowance api.Nullable[allowance] `json:"allowance" doc:"What the provider last reported of the key's allowance; null until it reports one"`
	State     standing                `json:"state"`
}

func windowOf(w *calls.Window) api.Nullable[window] {
	if w == nil {
		return api.Nullable[window]{}
	}
	return api.Some(window{Used: w.Used, Limit: w.Limit, Remaining: w.Remaining, Resets: w.Resets, Exact: w.Exact})
}

func some(s string) api.Nullable[string] {
	if s == "" {
		return api.Nullable[string]{}
	}
	return api.Some(s)
}

func providerOf(p subtitles.Provider, now time.Time) provider {
	out := provider{
		ID: string(p.Spec.ID), Name: p.Spec.Label, Kinds: []string{}, Position: p.Position, On: p.On, NeedsKey: p.Spec.Keyed, HasKey: p.HasKey,
		KeyPage: some(p.Spec.KeyPage), Account: some(p.Account), Plan: some(p.Plan),
	}
	for _, k := range p.Spec.Kinds {
		out.Kinds = append(out.Kinds, kindNames[k])
	}
	if a := p.Allowance; a != nil {
		out.Allowance = api.Some(allowance{Downloads: windowOf(a.Downloads), Searches: windowOf(a.Searches), At: p.AllowanceAt})
	}
	st := p.Standing(now)
	out.State = standing{Kind: standings[st]}
	switch st {
	case subtitles.Rejected:
		out.State.Since = api.Some(p.KeyCheckedAt)
		out.State.Words = some(p.Paused.Words)
	case subtitles.Out:
		out.State.Until = api.Some(p.Paused.Until)
	case subtitles.Failing:
		if !p.FailingSince.IsZero() {
			out.State.Since = api.Some(p.FailingSince)
		}
		out.State.Words = some(p.LastError)
	case subtitles.Off, subtitles.NeedsKey, subtitles.Ready:
	}
	return out
}

type providerList struct {
	Providers []provider `json:"providers" nullable:"false" doc:"In search order"`
}

type providerListOutput struct{ Body providerList }

type providerOutput struct{ Body provider }

type providerInput struct {
	ID string `path:"id"`
}

type providerChange struct {
	On bool `json:"on"`
}

type providerOnInput struct {
	ID   string `path:"id"`
	Body providerChange
}

type providerOrder struct {
	IDs []string `json:"ids" nullable:"false" doc:"Every provider's id once, in the new search order"`
}

type orderInput struct{ Body providerOrder }

type providerKey struct {
	Key string `json:"key" minLength:"1" maxLength:"256"`
}

type keyInput struct {
	ID   string `path:"id"`
	Body providerKey
}

func (h handlers) list(ctx context.Context, by identity.Admin) (*providerListOutput, error) {
	ps, err := open(h.o).Providers(ctx, by)
	if err != nil {
		return nil, err
	}
	return listOf(ps), nil
}

func listOf(ps []subtitles.Provider) *providerListOutput {
	now := time.Now()
	out := providerList{Providers: make([]provider, len(ps))}
	for i, p := range ps {
		out.Providers[i] = providerOf(p, now)
	}
	return &providerListOutput{Body: out}
}

func one(p subtitles.Provider, err error) (*providerOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	return &providerOutput{Body: providerOf(p, time.Now())}, nil
}
