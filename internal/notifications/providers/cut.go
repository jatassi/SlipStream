package providers

import (
	"strings"
	"unicode/utf16"
)

// ellipsis ends a cut string.
const ellipsis = "…"

// cut is s in at most n characters, ending in "…" when it was longer.
// Characters are counted in UTF-16 code units, as Discord and Telegram
// count them, which is never fewer than the runes another provider counts.
// A rune is never split.
func cut(s string, n int) string {
	if units(s) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	room, end := n-1, 0
	for i, r := range s {
		w := utf16.RuneLen(r)
		if w < 0 {
			w = 1
		}
		if room < w {
			break
		}
		room -= w
		end = i + len(string(r))
	}
	return strings.TrimRight(s[:end], " ") + ellipsis
}

// cutBytes is s in at most n bytes of UTF-8, ending in "…" when it was
// longer, for a provider that limits bytes.
func cutBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	end := 0
	for i, r := range s {
		if i+len(string(r)) > n-len(ellipsis) {
			break
		}
		end = i + len(string(r))
	}
	return s[:end] + ellipsis
}

func units(s string) int {
	n := 0
	for _, r := range s {
		if w := utf16.RuneLen(r); w > 0 {
			n += w
		} else {
			n++
		}
	}
	return n
}

// fit cuts parts, kept in order, so they and a sep of sepLen characters
// between each two total at most n. A part with no room left is dropped.
func fit(parts []string, sepLen, n int) []string {
	out := make([]string, 0, len(parts))
	used := 0
	for _, p := range parts {
		if len(out) > 0 {
			used += sepLen
		}
		room := n - used
		if room <= 0 {
			break
		}
		p = cut(p, room)
		out = append(out, p)
		used += units(p)
	}
	return out
}

// fitEscaped is s escaped by esc in at most n characters, cutting s before
// it is escaped so an escape is never split.
func fitEscaped(s string, n int, esc func(string) string) string {
	limit := n
	for {
		out := esc(cut(s, limit))
		if units(out) <= n || limit <= 0 {
			return out
		}
		limit -= units(out) - n
	}
}

// escapeHTML escapes text for Telegram's HTML and an email's HTML part.
var escapeHTML = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace

// escapeMrkdwn escapes the three characters Slack's mrkdwn reserves.
var escapeMrkdwn = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace

// escapeMarkdown escapes the characters Discord's markdown formats with,
// so a title such as *batteries not included reads as typed.
var escapeMarkdown = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "~", `\~`, "`", "\\`", "|", `\|`, ">", `\>`, "#", `\#`, "[", `\[`, "]", `\]`,
).Replace

func joinLines(lines []string) string { return strings.Join(lines, "\n") }
