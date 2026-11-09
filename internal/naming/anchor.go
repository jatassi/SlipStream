package naming

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// dash is the literal a file name's anchor ends with, where Plex,
// Jellyfin and Emby stop reading the item and start reading the file.
const dash = " - "

// atom is one byte of literal text, or one ref, of a template as it
// prints when every value is there. An untagged ref's prefix and suffix
// are text, so {(Year)} starts like ({Year}) and only a missing year tells
// them apart.
type atom struct {
	ref *boundRef
	c   byte
	// part is the index of the part the atom comes from, and at is its
	// byte offset there when that part is a literal, else -1.
	part, at int
}

func atoms(parts []part) []atom {
	var out []atom
	text := func(s string, part int, inLiteral bool) {
		for i := range len(s) {
			at := -1
			if inLiteral {
				at = i
			}
			out = append(out, atom{c: s[i], part: part, at: at})
		}
	}
	for i, p := range parts {
		switch p := p.(type) {
		case literal:
			text(string(p), i, true)
		case *boundRef:
			if p.tag != noTag {
				out = append(out, atom{ref: p, part: i, at: -1})
				continue
			}
			text(p.prefix, i, false)
			out = append(out, atom{ref: p, part: i, at: -1})
			text(p.suffix, i, false)
		}
	}
	return out
}

func (a atom) same(b atom) bool {
	if a.ref == nil || b.ref == nil {
		return a.ref == nil && b.ref == nil && a.c == b.c
	}
	x, y := a.ref, b.ref
	return x.entry == y.entry && x.tag == y.tag && x.sep == y.sep && x.casing == y.casing &&
		reflect.DeepEqual(x.mod, y.mod) && (x.tag == noTag || x.prefix == y.prefix && x.suffix == y.suffix)
}

// matched is how many of folder's atoms file starts with.
func matched(folder, file []atom) int {
	n := 0
	for n < len(folder) && n < len(file) && folder[n].same(file[n]) {
		n++
	}
	return n
}

// anchored is b with its anchorEnd set for a. A FolderAnchor's run ends
// where b stops matching the folder template, which may be inside a
// literal: b then has that literal split in two there.
func (s *Scheme) anchored(b bound, a Anchor) bound {
	switch a.kind {
	case folderAnchor:
		folder, file := atoms(s.folder.parts), atoms(b.parts)
		n := matched(folder, file)
		switch {
		case n == 0 || n < len(folder):
		case n == len(file):
			b.anchorEnd = len(b.parts)
		case file[n].at > 0:
			b = split(b, file[n].part, file[n].at)
			b.anchorEnd = file[n].part + 1
		case file[n].at == 0 || file[n-1].part != file[n].part:
			b.anchorEnd = file[n].part
		}
	case blockAnchor:
		if spans := b.blockSpans(); len(spans) > 0 {
			b.anchorEnd = spans[0].to
		}
	case tokenAnchor:
		e := s.cat.byKey[KeyOf(a.token)]
		for i, p := range b.parts {
			if r, ok := p.(*boundRef); ok && r.entry == e {
				b.anchorEnd = i + 1
				break
			}
		}
	}
	return b
}

// split is b with its literal part i split in two at byte at.
func split(b bound, i, at int) bound {
	l, ok := b.parts[i].(literal)
	if !ok {
		panic(fmt.Sprintf("naming: split part %d, a %T", i, b.parts[i]))
	}
	b.parts = slices.Concat(b.parts[:i], []part{l[:at], l[at:]}, b.parts[i+1:])
	b.index()
	return b
}

// anchorProblem is what keeps file variant v's anchor from grouping its
// files, as a sentence, or "" when nothing does: the anchor is missing,
// an Item or file token comes before the dash, or the dash doesn't follow
// the anchor.
func (s *Scheme) anchorProblem(v int) string {
	b, a := s.files[v], s.decl.Files[v].Anchor
	why := groupItem
	if a.kind == folderAnchor {
		why = groupWork
	}
	if b.anchorEnd == 0 {
		return s.noAnchor(b, a) + why + "."
	}
	for i, p := range b.parts {
		switch p := p.(type) {
		case literal:
			if i == b.anchorEnd && strings.HasPrefix(string(p), dash) {
				return ""
			}
			if i >= b.anchorEnd && strings.Contains(string(p), dash) {
				return s.dashAfter(b, a) + why + "."
			}
		case *boundRef:
			if (p.entry.Scope == ItemScope || p.entry.Scope == FileScope) && (i >= b.anchorEnd || !anchors(p, a)) {
				return p.text + " comes before ' - '. Move it after ' - ', because it differs between files."
			}
		}
	}
	return s.dashAfter(b, a) + why + "."
}

const (
	groupWork = "because Jellyfin and Emby show a Work's files as one item only when every name does"
	groupItem = "because Plex, Jellyfin and Emby show an Item's files as one item only when every name does"
)

// anchors reports that r is part of a's anchor, so it may vary by Item.
func anchors(r *boundRef, a Anchor) bool {
	switch a.kind {
	case blockAnchor:
		return r.entry.Seq != 0
	case tokenAnchor:
		return r.entry.key == KeyOf(a.token)
	case folderAnchor:
	}
	return false
}

func (s *Scheme) noAnchor(b bound, a Anchor) string {
	switch a.kind {
	case folderAnchor:
		file := atoms(b.parts)
		if n := matched(atoms(s.folder.parts), file); n < len(file) && file[n].at < 0 {
			return "The file name starts with " + typed(b.parts[:file[n].part+1]) + ". Start it with the folder name and ' - ', "
		}
		return "The file name doesn't start with the folder template " + typed(s.folder.parts) + ". Start it with the folder name and ' - ', "
	case blockAnchor:
		var names []string
		for _, t := range s.decl.Tokens {
			if t.Seq == SeqOuter || t.Seq == SeqInner {
				names = append(names, "{"+t.Name+"}")
			}
		}
		return "The file name has no " + strings.Join(names, " and ") + ". Add them and then ' - ', "
	case tokenAnchor:
	}
	return "The file name has no {" + a.token + "}. Add it and then ' - ', "
}

func (s *Scheme) dashAfter(b bound, a Anchor) string {
	after := typed(b.parts[b.anchorEnd-1 : b.anchorEnd])
	if a.kind == folderAnchor {
		after = "the folder name"
	}
	return "' - ' doesn't come right after " + after + ". Put it there, "
}

// typed is parts as the Admin typed them.
func typed(parts []part) string {
	var sb strings.Builder
	for _, p := range parts {
		switch p := p.(type) {
		case literal:
			sb.WriteString(string(p))
		case *boundRef:
			sb.WriteString(p.text)
		}
	}
	return sb.String()
}

// anchorText is how file variant v's anchor prints for sub and f: the
// folder template's text for a FolderAnchor, else the file template's
// parts before anchorEnd.
func (s *Scheme) anchorText(v int, sub Subject, f FileFacts) string {
	if s.decl.Files[v].Anchor.kind == folderAnchor {
		return s.style().render(s.folder, Subject{Work: sub.Work}, f)
	}
	b := s.files[v]
	head := bound{field: b.field, parts: b.parts[:b.anchorEnd]}
	head.index()
	return s.style().render(head, sub, f)
}
