package imports

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ActionKind is what a file's row offers. The last three are links to
// where the Admin acts.
type ActionKind string

const (
	ActConfirm        ActionKind = "confirm"
	ActRetry          ActionKind = "retry"
	ActDecline        ActionKind = "decline"
	ActForce          ActionKind = "force"
	ActMatch          ActionKind = "match"
	ActNotThisRelease ActionKind = "notThisRelease"
	ActChangeEditions ActionKind = "changeEditions"
	ActTurnOnRenaming ActionKind = "turnOnRenaming"
	ActChangeSetting  ActionKind = "changeSetting"
)

// ActionKinds is every ActionKind, in the wire's order.
var ActionKinds = []ActionKind{
	ActConfirm, ActRetry, ActDecline, ActForce, ActMatch, ActNotThisRelease, ActChangeEditions, ActTurnOnRenaming, ActChangeSetting,
}

// rule is what a reason's row leads with: its sentence, the action it
// recommends, none when empty, and the others it offers.
type rule struct {
	sentence    func(Reason, wording) string
	recommended ActionKind
	also        []ActionKind
}

// rules is the spec's table of waiting reasons, and the sentence of every
// other reason a download or a manual import gives. A root Import's S22
// reasons are worded by their own fields.
var rules = map[ReasonCode]rule{
	NotAnUpgrade: {
		sentence: func(r Reason, w wording) string {
			return "This isn't better than the file you have for " + w.targets(r.Targets) + "."
		},
		recommended: ActDecline, also: []ActionKind{ActForce, ActNotThisRelease},
	},
	MeetsNothing: {
		sentence:    func(Reason, wording) string { return "This doesn't meet any Target this Work wants." },
		recommended: ActDecline, also: []ActionKind{ActNotThisRelease},
	},
	EditionNotWanted: {
		sentence: func(r Reason, w wording) string {
			return "This Work stopped wanting the " + w.edition(r.Edition) + " after this was grabbed."
		},
		recommended: ActDecline, also: []ActionKind{ActChangeEditions},
	},
	OtherItems: {
		sentence: func(r Reason, w wording) string {
			return fmt.Sprintf("This file's name says %s. The download was for %s.", w.items(r.Reading), w.items(r.Of))
		},
		recommended: ActConfirm, also: []ActionKind{ActMatch, ActDecline},
	},
	NoNumber: {
		sentence: func(r Reason, w wording) string {
			return fmt.Sprintf("This file's name has no %s number. The folder says %s.", w.item(), w.level(r.Level))
		},
		recommended: ActMatch, also: []ActionKind{ActDecline},
	},
	Parts: {
		sentence: func(r Reason, w wording) string {
			return fmt.Sprintf("This %s is split into %d files. SlipStream imports a %s as one file.", w.work(), r.Parts, w.work())
		},
		recommended: ActDecline, also: []ActionKind{ActNotThisRelease},
	},
	NameTaken: {
		sentence: func(r Reason, _ wording) string {
			return "Another file already has the name " + r.Name + ". Rename or move that file, then retry."
		},
		recommended: ActRetry, also: []ActionKind{ActTurnOnRenaming, ActDecline},
	},
	NameTooLong: {
		sentence: func(r Reason, _ wording) string {
			return fmt.Sprintf("This name is %d characters, and names on this Node can be %d at most. Shorten the template, then retry.", r.Length, r.Limit)
		},
		recommended: ActRetry, also: []ActionKind{ActDecline},
	},
	Settings: {sentence: settingsSentence, recommended: ActChangeSetting, also: []ActionKind{ActDecline}},
	Busy: {sentence: func(r Reason, w wording) string {
		return "Another import is placing " + w.items(r.Reading) + ". This one continues after it."
	}, also: []ActionKind{ActDecline}},
	Confirm: {sentence: func(r Reason, w wording) string { return "Fills " + w.targets(r.Targets) + " for " + w.label() + "." }, recommended: ActConfirm, also: []ActionKind{ActMatch, ActDecline}},
	NoMatch: {
		sentence: func(_ Reason, w wording) string {
			return "No " + w.work() + " in the library matches this file's name."
		},
		recommended: ActMatch, also: []ActionKind{ActDecline},
	},
	NoRule: {
		sentence: func(_ Reason, w wording) string {
			return "This file's name doesn't name " + article(w.item()) + " of " + w.label() + "."
		},
		recommended: ActMatch, also: []ActionKind{ActDecline},
	},
	Sample: {sentence: fixed("This is a sample.")},
	Bonus:  {sentence: fixed("This is a bonus feature.")},
	TooSmall: {sentence: func(r Reason, _ wording) string {
		return "This is smaller than " + sizeWords(r.Value) + ", the smallest video file to import."
	}},
	NotAVideoType: {sentence: fixed("Its extension isn't one of the video file extensions.")},
	RuntimeSample: {sentence: func(r Reason, _ wording) string {
		return fmt.Sprintf("This runs shorter than %d%% of its runtime, which makes it a sample.", r.Value)
	}},
	HiddenFolder:  {sentence: fixed("This is in a hidden, system or disc folder.")},
	SiblingBetter: {sentence: fixed("Another file in this download is better.")},
	KeptBetter:    {sentence: fixed("These Items keep their files, which are as good or better.")},
	Unreadable: {sentence: func(r Reason, _ wording) string {
		return fmt.Sprintf("ffprobe can't read this file. It printed %q.", r.Line)
	}},
	ImportFail: {
		sentence: func(r Reason, _ wording) string {
			return fmt.Sprintf("Placing this file failed with %q. Retry once that is fixed.", r.Line)
		},
		recommended: ActRetry, also: []ActionKind{ActDecline},
	},
	RootChanged: {sentence: fixed("The Work moved to another library root while this was placed.")},
	ByAdmin:     {sentence: fixed("You chose not to import this.")},
	Removed:     {sentence: fixed("The download was removed.")},
	Canceled:    {sentence: fixed("The Grab was canceled.")},
	Blocklisted: {sentence: fixed("The Release is on the Blocklist.")},
	WorkDeleted: {sentence: fixed("The Work was deleted.")},
}

func fixed(s string) func(Reason, wording) string { return func(Reason, wording) string { return s } }

func settingsSentence(r Reason, _ wording) string {
	switch r.Setting {
	case SmallestVideo.Key:
		return "Every video here is smaller than " + sizeWords(r.Value) + ", the smallest video file to import."
	case SampleShorter.Key:
		return fmt.Sprintf("Every video here runs shorter than %d%% of its runtime, the sample setting.", r.Value)
	}
	return "Every video here has an extension that isn't one of the video file extensions."
}

func sizeWords(n int64) string {
	const mb, gb = 1 << 20, 1 << 30
	if n > 0 && n%gb == 0 {
		return fmt.Sprintf("%d GB", n/gb)
	}
	return fmt.Sprintf("%d MB", n/mb)
}

// wording is what a reason's sentence names, read for its Import's Work.
type wording interface {
	targets(ids []targets.TargetID) string
	items(ids []works.ItemID) string
	level(id works.LevelID) string
	edition(cut release.CutKey) string
	// item and work are the Content Type's nouns for an Item and a Work,
	// and label the Work's.
	item() string
	work() string
	label() string
}

// sentence is r worded for w, empty for a root Import's S22 reason.
func (r Reason) sentence(w wording) string {
	if rl, ok := rules[r.Code]; ok && r.Scan == nil {
		return rl.sentence(r, w)
	}
	return ""
}

// Actions is what f offers now, recommended first, empty for none: its
// reason's rule while it waits, Retry and Don't import on an offered hold,
// Not this Release only on a grab Import, and Change the match on every
// waiting file of a folder Import.
func (f *File) Actions(im Import) (recommended ActionKind, also []ActionKind) {
	switch {
	case f.Hold != nil && f.Hold.offered():
		return ActRetry, []ActionKind{ActDecline}
	case f.Reason == nil || f.Reason.Scan != nil || f.Outcome != Waiting && f.Outcome != FailedOut:
		return "", nil
	}
	rl := rules[f.Reason.Code]
	for _, a := range rl.also {
		if a != ActNotThisRelease || im.Source == FromGrab {
			also = append(also, a)
		}
	}
	if im.Source == FromFolder && f.Outcome == Waiting && rl.recommended != ActMatch && !slices.Contains(also, ActMatch) {
		also = append(also, ActMatch)
	}
	return rl.recommended, also
}

// article is noun with its indefinite article.
func article(noun string) string {
	if noun != "" && strings.ContainsRune("aeiou", rune(noun[0])) {
		return "an " + noun
	}
	return "a " + noun
}

// joinWords is words in a list: "a", "a and b", "a, b and c".
func joinWords(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}
