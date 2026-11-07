package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/provenance"
)

// Report is what changed upstream since each unit's recorded commit, up
// to one Prowlarr commit. One commit and one set of manifests always give
// the same Report.
type Report struct {
	// Ref is the Prowlarr commit the report reads, and Date its
	// committer date.
	Ref  string `json:"ref"`
	Date string `json:"date"`
	// ChangeSets are one draft PR each, in the order the Markdown lists
	// them.
	ChangeSets []ChangeSet `json:"changeSets"`
	UpToDate   []Synced    `json:"upToDate"`
}

// ChangeSet is one PR's worth of upstream change.
type ChangeSet struct {
	// Key is Subject@<To's short hash>. It stays the same until upstream
	// changes the subject again, so a routine finds the PR it opened.
	Key     string `json:"key"`
	Subject string `json:"subject"`
	Kind    Kind   `json:"kind"`
	// Unit is the unit an upstream or lifecycle change set is about.
	Unit string `json:"unit,omitempty"`
	// From is the unit's recorded commit, or skipped.json's for the
	// hidden list. To is the newest upstream commit in the set, which
	// bump moves the unit to.
	From    string       `json:"from,omitempty"`
	To      string       `json:"to"`
	Commits []Commit     `json:"commits"`
	Files   []FileChange `json:"files,omitempty"`
	// Diff is git diff From To over the unit's files. A deleted port has
	// none.
	Diff   string `json:"diff,omitempty"`
	Site   *Site  `json:"site,omitempty"`
	Reason string `json:"reason,omitempty"`
	// Added and Removed are hidden-list ids.
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

type Kind string

const (
	Upstream   Kind = "upstream"    // a unit's files changed
	Obsolete   Kind = "obsolete"    // upstream marked a ported site Obsolete
	Moved      Kind = "moved"       // ... with a message naming YML, YAML or Cardigann
	Deleted    Kind = "deleted"     // every file a port maps is gone
	NewSite    Kind = "new-site"    // a live C# torrent site no unit maps and skipped.json doesn't list
	HiddenList Kind = "hidden-list" // _definitionBlocklist changed
)

type Commit struct {
	SHA     string `json:"sha"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

// FileChange is one upstream file a change set touches and the Go files
// it maps to.
type FileChange struct {
	Path    string   `json:"path"`
	Status  string   `json:"status"`
	GoFiles []string `json:"goFiles,omitempty"`
}

// Site is a new site upstream.
type Site struct {
	Name  string   `json:"name"`
	Base  string   `json:"base"`
	Files []string `json:"files"`
	// Media are what its AddCategoryMapping lines carry.
	Media []string `json:"media"`
}

type Synced struct {
	Unit   string `json:"unit"`
	Commit string `json:"commit"`
}

type reporter struct {
	ctx      context.Context
	tree     *provenance.Tree
	target   string
	upstream map[string][]byte // every .cs file under Definitions at target, skipped ones left out
	sites    []class
}

func buildReport(ctx context.Context, repo string, tree *provenance.Tree, ref string) (Report, error) {
	problems, err := provenance.Check(repo, nil)
	if err != nil {
		return Report{}, err
	}
	if len(problems) > 0 {
		return Report{}, problemsError(problems)
	}
	units, _, err := provenance.Units(repo)
	if err != nil {
		return Report{}, err
	}
	skipped, err := provenance.ReadSkipped(repo)
	if err != nil {
		return Report{}, err
	}
	head, err := resolve(ctx, tree, ref)
	if err != nil {
		return Report{}, err
	}
	r := &reporter{ctx: ctx, tree: tree, target: head.SHA}
	if err := r.readDefinitions(skipped); err != nil {
		return Report{}, err
	}
	rep := Report{Ref: head.SHA, Date: head.Date}
	mapped := map[string]bool{}
	for _, u := range units {
		for _, f := range u.Upstream() {
			mapped[f] = true
		}
		cs, err := r.unitChange(u)
		if err != nil {
			return Report{}, err
		}
		if cs == nil {
			rep.UpToDate = append(rep.UpToDate, Synced{u.ID, u.Commit})
			continue
		}
		rep.ChangeSets = append(rep.ChangeSets, *cs)
	}
	newSites, err := r.newSites(mapped)
	if err != nil {
		return Report{}, err
	}
	hidden, err := r.hiddenListChange(skipped.Commit)
	if err != nil {
		return Report{}, err
	}
	rep.ChangeSets = append(rep.ChangeSets, newSites...)
	if hidden != nil {
		rep.ChangeSets = append(rep.ChangeSets, *hidden)
	}
	slices.SortStableFunc(rep.ChangeSets, func(a, b ChangeSet) int {
		return cmp.Compare(section(a), section(b))
	})
	return rep, nil
}

func problemsError(problems []provenance.Problem) error {
	lines := make([]string, len(problems))
	for i, p := range problems {
		lines[i] = p.String()
	}
	return errors.New(strings.Join(lines, "\n"))
}

// sections order the Markdown's headings and the change sets under them.
type reportSection struct {
	title string
	in    func(ChangeSet) bool
}

var sections = []reportSection{
	{"Ports", func(c ChangeSet) bool { return c.Kind == Upstream && strings.HasPrefix(c.Unit, "port:") }},
	{"Family base files", func(c ChangeSet) bool { return c.Kind == Upstream && strings.HasPrefix(c.Unit, "family:") }},
	{"Port kit and shared parsing", func(c ChangeSet) bool { return c.Kind == Upstream }},
	{"Obsolete, moved and deleted ports", func(c ChangeSet) bool { return c.Kind == Obsolete || c.Kind == Moved || c.Kind == Deleted }},
	{"New sites", func(c ChangeSet) bool { return c.Kind == NewSite }},
	{"Hidden list", func(c ChangeSet) bool { return c.Kind == HiddenList }},
}

func section(c ChangeSet) int {
	return slices.IndexFunc(sections, func(s reportSection) bool { return s.in(c) })
}

func resolve(ctx context.Context, tree *provenance.Tree, ref string) (Commit, error) {
	commits, err := log(ctx, tree, "-1", ref+"^{commit}", "--")
	if err != nil {
		return Commit{}, err
	}
	if len(commits) != 1 {
		return Commit{}, fmt.Errorf("%s names no commit in the Prowlarr clone", ref)
	}
	return commits[0], nil
}

// log is the commits git log args lists, oldest first.
func log(ctx context.Context, tree *provenance.Tree, args ...string) ([]Commit, error) {
	out, err := tree.Git(ctx, append([]string{"log", "--topo-order", "--format=%H%x00%cs%x00%s%x00"}, args...)...)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSpace(string(out)), "\x00")
	var commits []Commit
	for i := 0; i+2 < len(fields); i += 3 {
		commits = append(commits, Commit{strings.TrimSpace(fields[i]), fields[i+1], fields[i+2]})
	}
	slices.Reverse(commits)
	return commits, nil
}

func (r *reporter) readDefinitions(skipped provenance.Skipped) error {
	all, err := r.tree.Files(r.ctx, r.target, definitionsDir)
	if err != nil {
		return err
	}
	var paths []string
	for _, f := range all {
		if strings.HasSuffix(f, ".cs") && !skipped.Skips(f) {
			paths = append(paths, f)
		}
	}
	r.upstream, err = r.tree.Read(r.ctx, r.target, paths)
	if err != nil {
		return err
	}
	var declared []class
	for _, f := range paths {
		declared = append(declared, classes(f, r.upstream[f])...)
	}
	r.sites = sites(declared)
	return nil
}

// unitChange is u's change set, nil when nothing it maps changed.
func (r *reporter) unitChange(u provenance.Unit) (*ChangeSet, error) {
	files := u.Upstream()
	commits, err := log(r.ctx, r.tree, append([]string{u.Commit + ".." + r.target, "--"}, files...)...)
	if err != nil || len(commits) == 0 {
		return nil, err
	}
	to := commits[len(commits)-1].SHA
	cs := &ChangeSet{Kind: Upstream, Subject: u.ID, Unit: u.ID, From: u.Commit, To: to, Commits: commits}
	if cs.Files, err = r.fileChanges(u, to); err != nil {
		return nil, err
	}
	if u.Kind == provenance.Port {
		r.lifecycle(u, cs)
	}
	if cs.Kind != Deleted {
		diff, err := r.tree.Git(r.ctx, append([]string{"diff", "--no-color", "--no-ext-diff", "--no-renames", "--src-prefix=a/", "--dst-prefix=b/", u.Commit, to, "--"}, files...)...)
		if err != nil {
			return nil, err
		}
		cs.Diff = string(diff)
	}
	cs.Key = cs.Subject + "@" + to[:7]
	return cs, nil
}

var statuses = map[string]string{"A": "added", "M": "modified", "D": "deleted", "T": "modified"}

func (r *reporter) fileChanges(u provenance.Unit, to string) ([]FileChange, error) {
	out, err := r.tree.Git(r.ctx, append([]string{"diff", "--name-status", "--no-renames", "-z", u.Commit, to, "--"}, u.Upstream()...)...)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var changes []FileChange
	for i := 0; i+1 < len(fields); i += 2 {
		changes = append(changes, FileChange{Path: fields[i+1], Status: statuses[fields[i]], GoFiles: u.GoFiles(fields[i+1])})
	}
	slices.SortFunc(changes, func(a, b FileChange) int { return strings.Compare(a.Path, b.Path) })
	return changes, nil
}

// lifecycle turns cs into a lifecycle change set when every file the
// port maps is gone, or upstream marked its site Obsolete.
func (r *reporter) lifecycle(u provenance.Unit, cs *ChangeSet) {
	files := u.Upstream()
	if !slices.ContainsFunc(files, func(f string) bool { _, ok := r.upstream[f]; return ok }) {
		cs.Kind, cs.Subject = Deleted, "deleted:"+u.ID
		return
	}
	for _, s := range r.sites {
		if !s.IsObsolete || !slices.Contains(files, s.File) {
			continue
		}
		cs.Kind, cs.Reason = Obsolete, s.Obsolete
		if movedWords.MatchString(s.Obsolete) {
			cs.Kind = Moved
		}
		cs.Subject = string(cs.Kind) + ":" + u.ID
		return
	}
}

func (r *reporter) newSites(mapped map[string]bool) ([]ChangeSet, error) {
	var out []ChangeSet
	for _, s := range r.sites {
		if s.IsObsolete || mapped[s.File] {
			continue
		}
		files := []string{s.File}
		if dir := path.Dir(s.File) + "/"; dir != definitionsDir {
			files = nil
			for f := range r.upstream {
				if strings.HasPrefix(f, dir) {
					files = append(files, f)
				}
			}
			slices.Sort(files)
		}
		srcs := make([][]byte, len(files))
		for i, f := range files {
			srcs[i] = r.upstream[f]
		}
		commits, err := log(r.ctx, r.tree, append([]string{"-1", r.target, "--"}, files...)...)
		if err != nil {
			return nil, err
		}
		to := commits[0].SHA
		out = append(out, ChangeSet{
			Key: "new:" + s.Name + "@" + to[:7], Subject: "new:" + s.Name, Kind: NewSite, To: to, Commits: commits,
			Site: &Site{Name: s.Name, Base: s.Parent, Files: files, Media: mediaOf(srcs...)},
		})
	}
	return out, nil
}

func (r *reporter) hiddenListChange(from string) (*ChangeSet, error) {
	both, err := r.tree.Read(r.ctx, from, []string{hiddenListFile})
	if err != nil {
		return nil, err
	}
	was := hiddenList(both[hiddenListFile])
	now, err := r.tree.Read(r.ctx, r.target, []string{hiddenListFile})
	if err != nil {
		return nil, err
	}
	is := hiddenList(now[hiddenListFile])
	if slices.Equal(was, is) {
		return nil, nil
	}
	commits, err := log(r.ctx, r.tree, from+".."+r.target, "--", hiddenListFile)
	if err != nil {
		return nil, err
	}
	to := r.target
	if len(commits) > 0 {
		to = commits[len(commits)-1].SHA
	}
	return &ChangeSet{
		Key: "hidden-list@" + to[:7], Subject: "hidden-list", Kind: HiddenList, From: from, To: to, Commits: commits,
		Added: without(is, was), Removed: without(was, is),
	}, nil
}

// without is the members of a that b lacks.
func without(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}
