package formatsapi

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"golang.org/x/text/cases"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
)

type paste struct {
	Text    string   `json:"text" minLength:"1" doc:"One Custom Format's JSON or a list of them, as TRaSH Guides, Radarr and Sonarr export them or as Sonarr's API returns them"`
	App     string   `json:"app,omitempty" enum:"radarr,sonarr" doc:"The app the text was written for; left out, the app its kinds settle: Edition, Quality modifier or Year mean Radarr and Release type means Sonarr"`
	Replace []string `json:"replace,omitempty" format:"uuid" doc:"The Custom Formats of a pasted one's name that it replaces; a pasted one whose name is taken and not listed here is added beside the other, with a number after its name"`
}

type pasteInput struct {
	targetsapi.Changing
	Body paste
}

type formatRef struct {
	ID      string `json:"id" format:"uuid"`
	Name    string `json:"name"`
	Dialect string `json:"dialect" enum:"slipstream,radarr,sonarr"`
}

type pastedFormat struct {
	Name     string     `json:"name" doc:"Its name in the text"`
	Outcome  string     `json:"outcome" enum:"add,replace,keep_both" doc:"Added, replacing sameName, or added beside sameName"`
	SavedAs  string     `json:"savedAs" doc:"The name it is saved under: its own, or with a number after it when another Custom Format has its name"`
	SameName *formatRef `json:"sameName,omitempty" doc:"The Custom Format that already has its name"`
}

type pasted struct {
	App     string            `json:"app" enum:"radarr,sonarr" doc:"The app the formats are read for: the one given, or the one their kinds settle"`
	Formats []pastedFormat    `json:"formats" nullable:"false" doc:"In the text's order"`
	Impact  targetsapi.Impact `json:"impact"`
}

type pastedOutput struct{ Body pasted }

func (h handlers) pasteCustomFormats(ctx context.Context, by identity.Admin, in *pasteInput) (*pastedOutput, error) {
	fs, err := formats.Paste([]byte(in.Body.Text))
	if err != nil {
		return nil, pasteRefused("", err)
	}
	app := formats.Dialect(in.Body.App)
	if app == "" {
		settled, ok := formats.AppOf(fs)
		if !ok {
			return nil, appNeeded
		}
		app = settled
	}
	for _, f := range fs {
		f.Dialect = app
		if err := formats.Check(f, formats.Timings{}); err != nil {
			return nil, pasteRefused(f.Name, err)
		}
	}
	held, err := open(h.goals).ListFormats(ctx, by)
	if err != nil {
		return nil, err
	}
	p, err := plan(fs, held, in.Body.Replace)
	if err != nil {
		return nil, err
	}
	i, err := shelf(h.goals).Change(ctx, by, targets.AddFormats{Dialect: app, Add: p.add, Replace: p.replace}, in.Mode())
	if err != nil {
		return nil, pasteRefused("", err)
	}
	return &pastedOutput{Body: pasted{App: string(app), Formats: p.formats, Impact: targetsapi.ImpactOf(i)}}, nil
}

type planned struct {
	add     []formats.Format
	replace map[formats.ID]formats.Format
	formats []pastedFormat
}

// plan names each of fs: it replaces the held format of its name when
// replace lists that format, and otherwise takes the first free name of
// its own and its own with " (2)", " (3)" and on after it.
func plan(fs []formats.Format, held []targets.StoredFormat, replace []string) (planned, error) {
	fold := cases.Fold()
	byName := map[string]targets.StoredFormat{}
	taken := map[string]bool{}
	for _, h := range held {
		byName[fold.String(h.Name)] = h
		taken[fold.String(h.Name)] = true
	}
	chosen := map[formats.ID]bool{}
	for i, s := range replace {
		id, err := targets.ParseFormatID(s)
		if err == nil && slices.ContainsFunc(fs, func(f formats.Format) bool { return byName[fold.String(f.Name)].ID == id }) {
			chosen[id] = true
			continue
		}
		return planned{}, api.Invalid.On(fmt.Sprintf("replace[%d]", i), "No pasted Custom Format has this one's name. Leave it out and paste again.")
	}
	p := planned{replace: map[formats.ID]formats.Format{}, formats: []pastedFormat{}}
	for _, f := range fs {
		out := pastedFormat{Name: f.Name, Outcome: "add", SavedAs: f.Name}
		same, clash := byName[fold.String(f.Name)]
		if clash {
			out.SameName = &formatRef{ID: same.ID.String(), Name: same.Name, Dialect: string(same.Dialect)}
			out.Outcome = "keep_both"
		}
		if _, replaced := p.replace[same.ID]; clash && chosen[same.ID] && !replaced {
			out.Outcome = "replace"
			p.replace[same.ID] = f
			p.formats = append(p.formats, out)
			continue
		}
		for n := 2; taken[fold.String(out.SavedAs)]; n++ {
			out.SavedAs = fmt.Sprintf("%s (%d)", f.Name, n)
		}
		taken[fold.String(out.SavedAs)] = true
		f.ID, f.Name = targets.NewFormatID(), out.SavedAs
		p.add = append(p.add, f)
		p.formats = append(p.formats, out)
	}
	return p, nil
}

// pasteRefused answers err from reading or saving the pasted format
// name, or from the paste as a whole when name is empty.
func pasteRefused(name string, err error) error {
	if e, ok := errors.AsType[*formats.PatternError](err); ok {
		of := ""
		if name != "" {
			of = fmt.Sprintf(" of the Custom Format %q", name)
		}
		return patternNotCompiled.On("text", patternWords(e, of)+" Fix the pattern and paste again.")
	}
	if e, ok := errors.AsType[*formats.KindError](err); ok {
		return kindUnknown.On("text", kindWords(e)+" Remove the condition and paste again.")
	}
	if e, ok := errors.AsType[*formats.EmptyError](err); ok {
		return formatEmpty.On("text", fmt.Sprintf("The Custom Format %q has no conditions. Add a condition to it or leave it out, and paste again.", e.Name))
	}
	if errors.Is(err, formats.ErrNotJSON) {
		return api.Invalid.On("text", "The text isn't Custom Format JSON. Copy the whole format from TRaSH Guides, Radarr or Sonarr and paste again.")
	}
	return refused(err, "text", "paste")
}
