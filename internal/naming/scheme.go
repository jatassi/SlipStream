package naming

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SchemeOf is d's scheme under s, as a type's stored settings make it.
func SchemeOf(d Declaration, s Settings) (*Scheme, error) { return newScheme(d, s) }

// newScheme binds d's templates under s, d's defaults where s is nil. A
// Declaration whose tokens don't make a catalog is an error. Templates
// that don't parse or bind, and a custom colon text a name can't hold,
// are a *RefusedError with a Problem for each.
func newScheme(d Declaration, s Settings) (*Scheme, error) {
	cat, err := newCatalog(d)
	if err != nil {
		return nil, err
	}
	sc := &Scheme{
		decl:        d,
		cat:         cat,
		rename:      or(s.Rename, true),
		colon:       or(s.Colon, ColonSmart),
		colonCustom: or(s.ColonCustom, ""),
		multi:       or(s.MultiEpisode, PrefixedRange),
	}
	var problems []Problem
	bindField := func(key, text string, f field) bound {
		if t := s.Templates[key]; t != nil {
			text = *t
		}
		b, err := parseBound(text, cat, f)
		if err != nil {
			problems = append(problems, Problem{Key: key, Sentence: err.Sentence})
		}
		return b
	}
	sc.folder = bindField(folderKey, d.Folder, folderField)
	for i, text := range d.Levels {
		sc.levels = append(sc.levels, bindField(levelKey(i), text, levelField))
	}
	if d.Aside != "" {
		sc.aside = bindField(asideKey, d.Aside, levelField)
	}
	for _, v := range d.Files {
		sc.files = append(sc.files, bindField(fileKey(v), v.File, fileField))
	}
	if sc.colon == ColonCustom {
		if i := strings.IndexFunc(sc.colonCustom, func(r rune) bool { return strings.ContainsRune(unstorable+`\/`, r) || unicode.IsControl(r) }); i >= 0 {
			c, _ := utf8.DecodeRuneInString(sc.colonCustom[i:])
			problems = append(problems, Problem{Key: colonCustomKey, Sentence: fmt.Sprintf("The colon text can't contain %q because a name on Windows can't hold it. Remove it.", c)})
		}
	}
	if problems != nil {
		return nil, &RefusedError{Problems: problems}
	}
	for i, v := range d.Files {
		sc.files[i] = sc.anchored(sc.files[i], v.Anchor)
	}
	return sc, nil
}

const (
	folderKey      = "folder"
	asideKey       = "aside"
	colonCustomKey = "colonCustom"
)

func levelKey(i int) string { return "levels." + strconv.Itoa(i+1) }

func fileKey(v Variant) string { return "file." + v.Key }

func or[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

func parseBound(text string, c *catalog, f field) (bound, *ParseError) {
	t, err := Parse(text)
	if err == nil {
		var b bound
		if b, err = bind(t, c, f); err == nil {
			return b, nil
		}
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		panic(fmt.Sprintf("naming: Parse and bind refuse with a *ParseError, not %T", err))
	}
	return bound{}, pe
}

func (s *Scheme) style() style { return style{colon: s.colon, custom: s.colonCustom, multi: s.multi} }

func (s *Scheme) Renames() bool { return s.rename }

// Reads reports the costly facts the scheme's templates read, so files
// matches formats only when a template prints them.
func (s *Scheme) Reads() Reads {
	if !s.rename {
		return Reads{}
	}
	for _, b := range s.files {
		for _, p := range b.parts {
			if r, ok := p.(*boundRef); ok && r.entry.Scope == FileScope && r.entry.Kind == KindList && (r.entry.list == nameList || r.entry.list == pickList) {
				return Reads{Formats: true}
			}
		}
	}
	return Reads{}
}

// Folder renders the Work folder of a Work from its values, for a Work
// with no files.
func (s *Scheme) Folder(work Set) (string, error) {
	name := segment(s.style().render(s.folder, Subject{Work: work}, FileFacts{}))
	if name == "" {
		return "", errors.New("the Work folder name is empty because the folder template prints nothing for this Work")
	}
	return name, nil
}

// Name names one file: the Work folder, the level folders and the file
// name, with a FolderAnchor swapped for the Work folder it lands in,
// Item-scope titles cut to fit Budget. A path that still doesn't fit is a
// *TooLongError. With renaming off the file segment is Keep under the
// cross-OS map.
func (s *Scheme) Name(p Placing) (Named, error) {
	folders, err := s.folders(p)
	if err != nil {
		return Named{}, err
	}
	if !s.rename {
		named, err := fits(folders, segment(crossOS(p.File.Keep)+p.Ext), p.Budget)
		if err != nil {
			return Named{}, err
		}
		return named, nil
	}
	v := s.variant(p.Subject)
	b := s.files[v]
	anchor := ""
	if s.decl.Files[v].Anchor.kind == folderAnchor && b.anchorEnd > 0 {
		if anchor, err = s.workFolder(folders, p.Subject.Work); err != nil {
			return Named{}, err
		}
	}
	return s.fit(b, p, folders, anchor)
}

// Dir is the folder p's file lands in, root-relative: its Work folder and
// level folders as Name puts them, "" for the root.
func (s *Scheme) Dir(p Placing) (string, error) {
	folders, err := s.folders(p)
	if err != nil {
		return "", err
	}
	return path.Join(folders...), nil
}

// folders are the folders p's file lands in, Work folder first.
func (s *Scheme) folders(p Placing) ([]string, error) {
	if p.Landing.straight {
		return p.Landing.dir, nil
	}
	work := p.Landing.work
	if work == "" {
		var err error
		if work, err = s.Folder(p.Subject.Work); err != nil {
			return nil, err
		}
	}
	folders := []string{work}
	if !p.LevelFolders || len(s.levels) == 0 {
		return folders, nil
	}
	first := Subject{Work: p.Subject.Work, Units: p.Subject.Units[:1]}
	for i, b := range s.levels {
		if i == 0 && first.Units[0].Aside && s.decl.Aside != "" {
			b = s.aside
		}
		name := segment(s.style().render(b, first, FileFacts{}))
		if name == "" {
			return nil, errors.New("a level folder name is empty because its template prints nothing for this Item")
		}
		folders = append(folders, name)
	}
	return folders, nil
}

// workFolder is the Work folder a file lands in: the first of folders,
// or rendered when the file lands at the root.
func (s *Scheme) workFolder(folders []string, work Set) (string, error) {
	if len(folders) > 0 {
		return folders[0], nil
	}
	return s.Folder(work)
}

// variant is the Declaration.Files index sub's file renders: its first
// Unit's variant, or the first variant when that Unit lacks a value the
// variant's anchor or episode blocks read.
func (s *Scheme) variant(sub Subject) int {
	if len(sub.Units) == 0 || sub.Units[0].Item.Variant == "" {
		return 0
	}
	key := sub.Units[0].Item.Variant
	v := slices.IndexFunc(s.decl.Files, func(v Variant) bool { return v.Key == key })
	if v < 0 {
		panic(fmt.Sprintf("naming: no file variant %q", key))
	}
	for _, e := range s.needs(v) {
		if e.read(sub, 0, FileFacts{}).kind == 0 {
			return 0
		}
	}
	return v
}

// needs are the entries variant v can't render its anchor without: its
// TokenAnchor's token, or the numbers of its episode blocks.
func (s *Scheme) needs(v int) []*entry {
	a := s.decl.Files[v].Anchor
	switch a.kind {
	case tokenAnchor:
		if e := s.cat.byKey[KeyOf(a.token)]; e != nil {
			return []*entry{e}
		}
	case blockAnchor:
		var out []*entry
		for _, p := range s.files[v].parts {
			if r, ok := p.(*boundRef); ok && r.entry.Seq != 0 {
				out = append(out, r.entry)
			}
		}
		return out
	case folderAnchor:
	}
	return nil
}

// fileText is b rendered for sub and f, with parts [0, anchorEnd) printed
// as anchor when anchor is set.
func (st style) fileText(b bound, sub Subject, f FileFacts, anchor string) string {
	if anchor == "" {
		return st.render(b, sub, f)
	}
	return collapse(append([]piece{{text: anchor}}, st.pieces(b, sub, f)[b.anchorEnd:]...))
}
