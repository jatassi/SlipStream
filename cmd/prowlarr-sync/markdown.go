package main

import (
	"fmt"
	"strings"
)

func (rep Report) markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Prowlarr sync report\n\nProwlarr at `%s` (%s). %d change sets, %d units up to date.\n", rep.Ref, rep.Date, len(rep.ChangeSets), len(rep.UpToDate))
	for i, s := range sections {
		var in []ChangeSet
		for _, c := range rep.ChangeSets {
			if section(c) == i {
				in = append(in, c)
			}
		}
		if len(in) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n", s.title)
		for _, c := range in {
			writeChangeSet(&b, c)
		}
	}
	if len(rep.UpToDate) > 0 {
		b.WriteString("\n## Up to date\n\n")
		for _, u := range rep.UpToDate {
			fmt.Fprintf(&b, "- %s at `%s`\n", u.Unit, short(u.Commit))
		}
	}
	return b.String()
}

func writeChangeSet(b *strings.Builder, c ChangeSet) {
	fmt.Fprintf(b, "\n### %s\n\n", c.Key)
	switch c.Kind {
	case Upstream:
		fmt.Fprintf(b, "Synced to `%s`. Run `prowlarr-sync bump --unit %s --to %s` once the diff is ported.\n", short(c.From), c.Unit, c.To)
	case Obsolete:
		fmt.Fprintf(b, "Upstream marked %s obsolete: %q. Delete the port and add it to the retired list in internal/ports/all/all.go.\n", c.Unit, c.Reason)
	case Moved:
		fmt.Fprintf(b, "Upstream moved %s to a Cardigann definition: %q. Keep the port until that definition passes the runner tier, then add it to the moved list in internal/ports/all/all.go and delete it.\n", c.Unit, c.Reason)
	case Deleted:
		fmt.Fprintf(b, "Upstream deleted every file %s maps. Delete the port and add it to the retired list in internal/ports/all/all.go.\n", c.Unit)
	case NewSite:
		fmt.Fprintf(b, "%s derives from %s and carries %s. Start its port with `prowlarr-sync new --site %s`, or list it in skipped.json with why.\n\n", c.Site.Name, c.Site.Base, mediaWords(c.Site.Media), c.Site.Name)
		for _, f := range c.Site.Files {
			fmt.Fprintf(b, "- `%s`\n", f)
		}
	case HiddenList:
		fmt.Fprintf(b, "The hidden list changed since `%s`. Added: %s. Removed: %s.\n", short(c.From), ids(c.Added), ids(c.Removed))
	}
	writeCommits(b, c.Commits)
	if len(c.Files) > 0 {
		b.WriteString("\n| Upstream file | Change | Go files |\n|---|---|---|\n")
		for _, f := range c.Files {
			fmt.Fprintf(b, "| `%s` | %s | %s |\n", f.Path, f.Status, ids(f.GoFiles))
		}
	}
	if c.Diff != "" {
		fence := strings.Repeat("`", max(3, longestRun(c.Diff, '`')+1))
		fmt.Fprintf(b, "\n%sdiff\n%s%s\n", fence, c.Diff, fence)
	}
}

func writeCommits(b *strings.Builder, commits []Commit) {
	if len(commits) == 0 {
		return
	}
	b.WriteString("\n| Commit | Date | Subject |\n|---|---|---|\n")
	for _, c := range commits {
		fmt.Fprintf(b, "| `%s` | %s | %s |\n", short(c.SHA), c.Date, strings.ReplaceAll(c.Subject, "|", `\|`))
	}
}

func short(sha string) string { return sha[:min(len(sha), 7)] }

func ids(xs []string) string {
	if len(xs) == 0 {
		return "none"
	}
	return "`" + strings.Join(xs, "`, `") + "`"
}

func mediaWords(m []string) string {
	if len(m) == 0 {
		return "no category mapping"
	}
	return strings.Join(m, ", ")
}

func longestRun(s string, r byte) int {
	longest, run := 0, 0
	for i := range len(s) {
		if s[i] == r {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}
