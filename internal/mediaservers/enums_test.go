package mediaservers_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/release"
)

func enumOf[T any](t *testing.T, field string) string {
	t.Helper()
	f, ok := reflect.TypeFor[T]().FieldByName(field)
	if !ok {
		t.Fatalf("%s has no field %s", reflect.TypeFor[T](), field)
	}
	return f.Tag.Get("enum")
}

func joined[S ~string](values []S) string {
	words := make([]string, len(values))
	for i, v := range values {
		words[i] = string(v)
	}
	return strings.Join(words, ",")
}

var (
	accesses = []link.Access{link.AccessPlexTV, link.AccessUsers}
	problems = []mediaservers.ProblemCode{
		mediaservers.ProblemPlexRefused, mediaservers.ProblemPlexLibrariesShort, mediaservers.ProblemUsernameTaken,
		mediaservers.ProblemServerRefused, mediaservers.ProblemMediaServerUnreached, mediaservers.ProblemFailed,
	}
)

// qualityResolutions is every resolution a Quality has, highest first.
func qualityResolutions() []release.Resolution {
	var out []release.Resolution
	for _, q := range release.Qualities() {
		if r := q.Info().Resolution; r != 0 && !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	slices.Sort(out)
	return out
}

func names[T interface{ String() string }](values []T) string {
	words := make([]string, len(values))
	for i, v := range values {
		words[i] = v.String()
	}
	return strings.Join(words, ",")
}

// startedBy is every event the access table lets by start, sorted.
func startedBy(by access.By) string {
	var events []access.Event
	for _, t := range access.Table {
		if t.By&by != 0 && !slices.Contains(events, t.On) {
			events = append(events, t.On)
		}
	}
	slices.Sort(events)
	return joined(events)
}

func sorted(enum string) string {
	values := strings.Split(enum, ",")
	slices.Sort(values)
	return strings.Join(values, ",")
}

func TestTheWireEnumsNameEveryDomainValueInOrder(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"ChangeView.Kind", enumOf[mediaservers.ChangeView](t, "Kind"), joined(link.ChangeKinds())},
		{"ChangeView.Step", enumOf[mediaservers.ChangeView](t, "Step"), joined(link.Steps())},
		{"ChangeView.Result", enumOf[mediaservers.ChangeView](t, "Result"), joined(link.Results())},
		{"ChangeView.Lost", enumOf[mediaservers.ChangeView](t, "Lost"), joined(link.Losses())},
		{"ServerView.Contact", enumOf[mediaservers.ServerView](t, "Contact"), joined(link.Contacts())},
		{"SessionView.Decision", enumOf[mediaservers.SessionView](t, "Decision"), joined(link.Decisions())},
		{"SessionView.Video", enumOf[mediaservers.SessionView](t, "Video"), joined(link.StreamDecisions())},
		{"SessionView.Audio", enumOf[mediaservers.SessionView](t, "Audio"), joined(link.StreamDecisions())},
		{"SessionView.Subtitles", enumOf[mediaservers.SessionView](t, "Subtitles"), joined(link.SubtitleDecisions())},
		{"PlayView.Decision", enumOf[mediaservers.PlayView](t, "Decision"), joined(link.Decisions())},
		{"PlayView.Video", enumOf[mediaservers.PlayView](t, "Video"), joined(link.StreamDecisions())},
		{"PlayView.Audio", enumOf[mediaservers.PlayView](t, "Audio"), joined(link.StreamDecisions())},
		{"PlayView.Subtitles", enumOf[mediaservers.PlayView](t, "Subtitles"), joined(link.SubtitleDecisions())},
		{"Blocked.Dimension", enumOf[playback.Blocked](t, "Dimension"), joined(playback.Dimensions())},
		{"Outcome.Kind", enumOf[playback.Outcome](t, "Kind"), joined(playback.OutcomeKinds())},
		{"WorkLine.State", enumOf[playback.WorkLine](t, "State"), joined(playback.WorkStates())},
		{"FactValue.Fact", enumOf[playback.FactValue](t, "Fact"), joined(playback.FactKinds())},
		{"Suggestion.Highest", enumOf[playback.Suggestion](t, "Highest"), names(qualityResolutions())},
		{"Suggestion.VideoCodecs", enumOf[playback.Suggestion](t, "VideoCodecs"), joined(release.VideoCodecs())},
		{"Suggestion.NotHDR", enumOf[playback.Suggestion](t, "NotHDR"), joined(release.HDRs())},
		{"Suggestion.NotAudioCodecs", enumOf[playback.Suggestion](t, "NotAudioCodecs"), joined(release.AudioCodecs())},
		{"Suggestion.NotChannels", enumOf[playback.Suggestion](t, "NotChannels"), joined(release.ChannelLayouts())},
		{"KindView.SignIn", enumOf[mediaservers.KindView](t, "SignIn"), joined([]link.SignIn{link.SignInPlexPIN, link.SignInKeyOrAdmin})},
		{"AccessView.State", enumOf[mediaservers.AccessView](t, "State"), joined(access.States())},
		{"AccessView.Actions", sorted(enumOf[mediaservers.AccessView](t, "Actions")), startedBy(access.ByAdmin)},
		{"AccessServer.Access", enumOf[mediaservers.AccessServer](t, "Access"), joined(accesses)},
		{"AccessProblem.Code", enumOf[mediaservers.AccessProblem](t, "Code"), joined(problems)},
		{"MyAccessRow.State", enumOf[mediaservers.MyAccessRow](t, "State"), joined(slices.DeleteFunc(access.States(), func(s access.State) bool { return !s.MemberSees() }))},
		{"MyAccessRow.Actions", sorted(enumOf[mediaservers.MyAccessRow](t, "Actions")), startedBy(access.ByMember)},
		{"MyServer.Access", enumOf[mediaservers.MyServer](t, "Access"), joined(accesses)},
		{"MemberPinState.State", enumOf[mediaservers.MemberPinState](t, "State"), joined([]mediaservers.PinState{mediaservers.PinWaiting, mediaservers.PinExpired, mediaservers.PinSignedIn, mediaservers.PinRefused})},
		{"MemberPinState.Refusal", enumOf[mediaservers.MemberPinState](t, "Refusal"), joined([]mediaservers.PinRefusal{mediaservers.PlexAccountTaken, mediaservers.PlexAccountOwner, mediaservers.PlexAccountOther})},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}
