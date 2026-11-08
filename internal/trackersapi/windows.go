package trackersapi

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

type attention struct {
	Level string `json:"level" enum:"warn,info"`
	Text  string `json:"text" doc:"One sentence"`
}

func attentionOf(as []trackers.Attention) []attention {
	out := make([]attention, len(as))
	for i, a := range as {
		out[i] = attention{Level: string(a.Level), Text: a.Text}
	}
	return out
}

const windowsPage = 50

type windowsInput struct {
	ID     string `path:"id" format:"uuid"`
	List   string `query:"list" enum:"open,history" doc:"open lists the windows that owe. history lists those met, missed or released, a missed one grabbed again included. Every window when absent."`
	Offset int    `query:"offset" minimum:"0" doc:"How many of the newest to skip"`
}

type owed struct {
	Seconds int64 `json:"seconds" minimum:"0" doc:"Seeding time still needed, 0 when the rule doesn't count it or it is reached"`
	Bytes   int64 `json:"bytes" minimum:"0" doc:"Upload still needed, 0 when the rule doesn't count ratio or it is reached"`
}

type window struct {
	InfoHash    string                  `json:"infoHash" pattern:"^[0-9a-f]{40}$"`
	Name        string                  `json:"name" doc:"Its latest torrent's name"`
	State       string                  `json:"state" enum:"owing,met,missed,released"`
	Torrents    []string                `json:"torrents" nullable:"false" doc:"Its live Torrents' ids, in added order; empty once none is live"`
	Rule        api.Nullable[ruleShape] `json:"rule" doc:"The rule it snapshot when it opened; null while none is known"`
	OpenedAt    time.Time               `json:"openedAt" doc:"Its torrent's first downloaded byte"`
	MetAt       *time.Time              `json:"metAt" doc:"When its Tracker's rule and the margin were met; null unless met"`
	ClosedAt    *time.Time              `json:"closedAt" doc:"Its latest miss or release; a regrab keeps it"`
	RegrabbedAt *time.Time              `json:"regrabbedAt" doc:"When a torrent of it came back after a miss"`
	Owed        api.Nullable[owed]      `json:"owed" doc:"What its latest miss still owed; null before a miss"`
	Owes        api.Nullable[owed]      `json:"owes" doc:"What it still needs to meet its Tracker's rule and the margin; null unless it owes under a known rule"`
	AtRisk      string                  `json:"atRisk" enum:"no,yes,unknown" doc:"Whether it is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline"`
	QuietSince  *time.Time              `json:"quietSince" doc:"When it stopped seeding while it owes; null while it seeds"`
}

type ruleShape struct {
	Clears      string   `json:"clears" enum:"time,ratio,either,both"`
	Ratio       *float64 `json:"ratio" doc:"null when the rule doesn't count ratio"`
	SeedSeconds *int64   `json:"seedSeconds" doc:"Seconds; null when the rule doesn't count seeding time"`
}

type windowList struct {
	Windows    []window `json:"windows" nullable:"false" doc:"Newest opened first, each as it reads now"`
	NextOffset *int     `json:"nextOffset" doc:"The offset of the next page; null on the last"`
}

type windowsOutput struct{ Body windowList }

func (h handlers) listWindows(ctx context.Context, by identity.Admin, in *windowsInput) (*windowsOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	all, err := h.trackers().Windows(ctx, by, id, trackers.WindowList(in.List))
	if err != nil {
		return nil, trackerAnswer(err)
	}
	out := windowList{Windows: []window{}}
	end := min(in.Offset+windowsPage, len(all))
	for _, w := range all[min(in.Offset, len(all)):end] {
		out.Windows = append(out.Windows, windowOf(w))
	}
	if end < len(all) {
		out.NextOffset = &end
	}
	return &windowsOutput{Body: out}, nil
}

func windowOf(v trackers.WindowView) window {
	w := v.Window
	out := window{
		InfoHash: hex.EncodeToString(v.InfoHash[:]), Name: v.Name, State: string(w.State), OpenedAt: w.OpenedAt,
		MetAt: when(w.MetAt), ClosedAt: when(w.ClosedAt), RegrabbedAt: when(w.RegrabbedAt), QuietSince: when(w.QuietSince),
		AtRisk: string(v.Risk), Torrents: make([]string, len(v.Torrents)),
	}
	for i, id := range v.Torrents {
		out.Torrents[i] = id.String()
	}
	if w.Rule != nil {
		ratio, seed := partsOf(w.Rule.Parts())
		out.Rule = api.Some(ruleShape{Clears: string(w.Rule.Clears()), Ratio: ratio, SeedSeconds: seed})
	}
	if w.Owed.Clears != "" {
		out.Owed = owedOf(w.Owed)
	}
	if v.Owes != nil {
		out.Owes = owedOf(*v.Owes)
	}
	return out
}

func owedOf(o seeding.Owes) api.Nullable[owed] {
	return api.Some(owed{Seconds: int64(o.Time / time.Second), Bytes: o.Bytes})
}

func when(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type plannedTorrent struct {
	ID      string                   `json:"id" format:"uuid"`
	Name    string                   `json:"name"`
	Tracker api.Nullable[trackerRef] `json:"tracker" doc:"null for a torrent with no known Tracker"`
}

type trackerRef struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type act struct {
	Torrent plannedTorrent `json:"torrent"`
	Action  string         `json:"action" enum:"pause,remove,remove-delete"`
}

type kept struct {
	Torrent plannedTorrent `json:"torrent"`
	Why     string         `json:"why" enum:"not-imported,draining" doc:"not-imported: a Grab on it didn't import, so removing it keeps the download copy; draining: its Node is draining Downloading, so it is removed keeping its download copy in place of a pause"`
}

type wait struct {
	Torrent plannedTorrent `json:"torrent"`
	Why     string         `json:"why" enum:"removing,resumed,ran,incomplete,grab-downloading,grab-importing,reconciling" doc:"What holds its met goal back: removing, it is being removed; resumed, an Admin resumed it after its goal paused it, so it seeds with no end; ran, its goal already acted on it; incomplete, its selected files aren't all there; grab-downloading or grab-importing, a Grab on it is still downloading or importing; reconciling, SlipStream is reconciling the library after a restore"`
}

type goalsPreview struct {
	Acts  []act  `json:"acts" nullable:"false" doc:"The torrents whose goal acts, in added order"`
	Kept  []kept `json:"kept" nullable:"false" doc:"The torrents removed keeping their download copy, in added order"`
	Waits []wait `json:"waits" nullable:"false" doc:"The torrents whose goal is met and waits, in added order"`
}

type goalsPreviewOutput struct{ Body goalsPreview }

func (h handlers) previewGoals(ctx context.Context, by identity.Admin, _ *struct{}) (*goalsPreviewOutput, error) {
	planned, err := h.trackers().Preview(ctx, by)
	if err != nil {
		return nil, err
	}
	out := goalsPreview{Acts: []act{}, Kept: []kept{}, Waits: []wait{}}
	for _, p := range planned {
		t := plannedOf(p)
		switch plan := p.Plan.(type) {
		case seeding.Act:
			out.Acts = append(out.Acts, act{Torrent: t, Action: string(plan.Do)})
		case seeding.Keep:
			out.Kept = append(out.Kept, kept{Torrent: t, Why: string(plan.Why)})
		case seeding.Wait:
			out.Waits = append(out.Waits, wait{Torrent: t, Why: string(plan.Why)})
		}
	}
	return &goalsPreviewOutput{Body: out}, nil
}

func plannedOf(p trackers.Planned) plannedTorrent {
	out := plannedTorrent{ID: p.ID.String(), Name: p.Name}
	if r := p.Tracker; r != nil {
		out.Tracker = api.Some(trackerRef{ID: r.ID.String(), Name: r.Name})
	}
	return out
}
