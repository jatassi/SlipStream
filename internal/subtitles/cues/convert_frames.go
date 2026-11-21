package cues

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	microDVDLine  = regexp.MustCompile(`^\{(\d{1,9})\}\{(\d{1,9})\}(.*)$`)
	microDVDStyle = regexp.MustCompile(`\{([yY]):([ibus,]+)\}`)
	microDVDCode  = regexp.MustCompile(`\{[a-zA-Z]:[^}]*\}`)
	mpl2Line      = regexp.MustCompile(`^\[(\d{1,9})\]\[(\d{1,9})\](.*)$`)
)

var errNoFrameRate = errors.New("it's MicroDVD with no frame rate line, and the video's frame rate isn't known")

// fromMicroDVD reads {start}{end}text frame lines at fps, or at the rate
// a first {1}{1}23.976 line gives when fps is zero. | breaks a line, and
// {y:i} or {Y:i} italics become <i>.
func fromMicroDVD(text string, _ language.Code, fps float64) ([]item, error) {
	var items []item
	first := true
	for l := range strings.Lines(text) {
		m := microDVDLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		if first {
			first = false
			if rate, err := strconv.ParseFloat(strings.TrimSpace(m[3]), 64); err == nil && rate > 1 && rate < 1000 && m[1] == "1" && m[2] == "1" {
				if fps == 0 {
					fps = rate
				}
				continue
			}
		}
		if fps <= 0 {
			return nil, errNoFrameRate
		}
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		items = append(items, item{
			start: frames(start, fps),
			end:   frames(end, fps),
			text:  microDVDText(m[3]),
		})
	}
	return items, nil
}

func frames(n int, fps float64) time.Duration {
	return time.Duration(float64(n) / fps * float64(time.Second)).Round(time.Millisecond)
}

func microDVDText(s string) string {
	whole := false
	lines := strings.Split(s, "|")
	for i, l := range lines {
		italic := false
		l = microDVDStyle.ReplaceAllStringFunc(l, func(code string) string {
			m := microDVDStyle.FindStringSubmatch(code)
			if strings.Contains(m[2], "i") {
				italic = true
				whole = whole || m[1] == "Y"
			}
			return ""
		})
		l = microDVDCode.ReplaceAllString(l, "")
		if italic || whole {
			l = "<i>" + l + "</i>"
		}
		lines[i] = l
	}
	return keepStyle(strings.Join(lines, "\n"))
}

// fromMPL2 reads [start][end]text lines in tenths of a second. | breaks a
// line, and a line opening with / is italic.
func fromMPL2(text string, _ language.Code, _ float64) ([]item, error) {
	var items []item
	for l := range strings.Lines(text) {
		m := mpl2Line.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		lines := strings.Split(m[3], "|")
		for i, line := range lines {
			if rest, ok := strings.CutPrefix(line, "/"); ok {
				lines[i] = "<i>" + rest + "</i>"
			}
		}
		items = append(items, item{
			start: time.Duration(start) * 100 * time.Millisecond,
			end:   time.Duration(end) * 100 * time.Millisecond,
			text:  keepStyle(strings.Join(lines, "\n")),
		})
	}
	return items, nil
}
