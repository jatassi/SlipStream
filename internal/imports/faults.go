package imports

import "slices"

// settingCodes are the left-out reasons only an Admin's setting gives, each
// with its setting.
var settingCodes = map[ReasonCode]string{TooSmall: SmallestVideo.Key, NotAVideoType: VideoExtensions.Key, RuntimeSample: SampleShorter.Key}

// faults applies the release faults once nothing of im is left to decide
// or place and nothing was placed. When every file failed unreadable, im
// fails unreadable. When a file was left out by a setting alone, those
// files wait settings, never blocklisted. Else every video was left out by
// the name and folder rules, and im fails no_video_file.
func faults(im *Import, rules Rules) {
	if im.Failure != nil || slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome != LeftOut && f.Outcome != FailedOut }) {
		return
	}
	if !slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome != FailedOut || f.Reason.Code != Unreadable }) {
		im.Failure = new(UnreadableFile)
		return
	}
	waits := false
	for i, f := range im.Files {
		if f.Outcome != LeftOut {
			continue
		}
		key, ok := settingCodes[f.Reason.Code]
		if !ok {
			continue
		}
		r := Reason{Code: Settings, Setting: key}
		switch key {
		case SmallestVideo.Key:
			r.Value = rules.Smallest
		case SampleShorter.Key:
			r.Value = int64(rules.Sample)
		}
		im.Files[i].Outcome, im.Files[i].Reason, waits = Waiting, &r, true
	}
	if !waits {
		im.Failure = new(NoVideoFile)
	}
}
