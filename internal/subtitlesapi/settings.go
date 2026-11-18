package subtitlesapi

import (
	"context"
	"encoding/json"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targetsapi"
)

var leadKinds = map[subtitles.LeadKind]string{
	subtitles.NoLanguages: "no_languages", subtitles.Uncovered: "uncovered", subtitles.SearchOff: "off", subtitles.SearchReady: "ready",
}

type lead struct {
	Kind      string   `json:"kind" enum:"no_languages,uncovered,off,ready" doc:"Which sentence leads the section, the first that holds: no Profile has a Wanted Language, a Content Type with Wanted Languages has no Subtitle Provider that can search, Subtitle search is off, or ready"`
	Uncovered []string `json:"uncovered" nullable:"false" doc:"The Content Types with Wanted Languages that no Subtitle Provider that is on, with its key, searches"`
	Missing   int      `json:"missing" minimum:"0" doc:"Wanted subtitles that no subtitle file or embedded track meets yet"`
	Interval  int64    `json:"interval" doc:"Seconds between Subtitle search runs"`
	Scans     []scans  `json:"scans" nullable:"false" doc:"Each Storage Node with files whose audio SlipStream still has to scan, in Node order"`
}

type scans struct {
	NodeID string `json:"nodeId" format:"uuid"`
	Name   string `json:"name"`
	Left   int    `json:"left" minimum:"1" doc:"Files whose audio is still to scan"`
}

type profileLanguages struct {
	ID          string                      `json:"id" format:"uuid"`
	Name        string                      `json:"name"`
	ContentType string                      `json:"contentType"`
	Languages   []targetsapi.WantedLanguage `json:"languages" nullable:"false" doc:"In the Admin's order"`
}

type subtitleSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"Subtitle search, upgrades and how often it runs, then the settings under Details"`
	Lead     lead                          `json:"lead"`
	Profiles []profileLanguages            `json:"profiles" nullable:"false" doc:"Each Profile with a Wanted Language, by name"`
}

type subtitleSettingsOutput struct{ Body subtitleSettings }

type valuesBody struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

type subtitleSettingsInput struct{ Body valuesBody }

type nodeInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
}

type nodeSettingsInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
	Body   valuesBody
}

type nodeSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"How many audio scans the Node runs at once"`
}

type nodeSettingsOutput struct{ Body nodeSettings }

func (h handlers) getSettings(ctx context.Context, by identity.Admin, _ *struct{}) (*subtitleSettingsOutput, error) {
	return h.settingsOf(ctx, by)
}

func (h handlers) updateSettings(ctx context.Context, by identity.Admin, in *subtitleSettingsInput) (*subtitleSettingsOutput, error) {
	if err := open(h.o).Settings().Apply(ctx, in.Body.Values); err != nil {
		return nil, answer(err)
	}
	return h.settingsOf(ctx, by)
}

func (h handlers) settingsOf(ctx context.Context, by identity.Admin) (*subtitleSettingsOutput, error) {
	s := open(h.o)
	fields, err := s.Settings().Fields(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.Lead(ctx)
	if err != nil {
		return nil, err
	}
	ps, err := goals(h.o).ListProfiles(ctx, by)
	if err != nil {
		return nil, err
	}
	out := subtitleSettings{Settings: settingapi.FieldsOf(fields), Profiles: []profileLanguages{}, Lead: lead{
		Kind: leadKinds[l.Kind], Uncovered: []string{}, Missing: l.Missing, Interval: int64(l.Interval.Seconds()), Scans: []scans{},
	}}
	for _, t := range l.Uncovered {
		out.Lead.Uncovered = append(out.Lead.Uncovered, string(t))
	}
	for _, sc := range l.Scans {
		out.Lead.Scans = append(out.Lead.Scans, scans{NodeID: sc.Node.String(), Name: sc.Name, Left: sc.Left})
	}
	for _, p := range ps {
		if len(p.Languages) == 0 {
			continue
		}
		pl := profileLanguages{ID: p.ID.String(), Name: p.Name, ContentType: string(p.ContentType), Languages: make([]targetsapi.WantedLanguage, len(p.Languages))}
		for i, wl := range p.Languages {
			pl.Languages[i] = targetsapi.WantedLanguageOf(wl)
		}
		out.Profiles = append(out.Profiles, pl)
	}
	return &subtitleSettingsOutput{Body: out}, nil
}

func nodeOf(s string) (role.NodeID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return role.NodeID{}, notStorage
	}
	return role.NodeID{UUID: u}, nil
}

func (h handlers) getNodeSettings(ctx context.Context, _ identity.Admin, in *nodeInput) (*nodeSettingsOutput, error) {
	n, err := nodeOf(in.NodeID)
	if err != nil {
		return nil, err
	}
	return nodeSettingsOf(ctx, open(h.o).NodeSettings(n))
}

func (h handlers) updateNodeSettings(ctx context.Context, _ identity.Admin, in *nodeSettingsInput) (*nodeSettingsOutput, error) {
	n, err := nodeOf(in.NodeID)
	if err != nil {
		return nil, err
	}
	p := open(h.o).NodeSettings(n)
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, answer(err)
	}
	return nodeSettingsOf(ctx, p)
}

func nodeSettingsOf(ctx context.Context, p setting.Panel) (*nodeSettingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, answer(err)
	}
	return &nodeSettingsOutput{Body: nodeSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}
