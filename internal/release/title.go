package release

// Title reads a release title: the type's Name, then every core fact.
// Quality, revision, group, the stream facts and id tags read the whole
// name; languages, the cut and the edition text read Name.Tail.
func Title(name string, names Names, cuts []Cut) Read {
	p := Prepare(name)
	n := names.ReadName(p)
	tail := n.Tail
	if tail == "" {
		tail = p.Title.String()
	}
	f := Facts{
		Revision:  readRevision(p),
		Group:     readGroup(p),
		Cut:       readCut(tail, cuts),
		Edition:   readEdition(tail),
		Languages: readLanguages(tail),
		IDs:       readIDs(p),
	}
	readQuality(p, &f)
	readStream(p, &f)
	return Read{Given: name, Name: n, Facts: f}
}

// File reads a file path's name, and its folder's when the file name
// gives no title and no numbering, in Sonarr's ParsePath order. It reads
// title facts only: a file's probe joins with Facts.WithProbe.
func File(path string, names Names, cuts []Cut) Read {
	return Title(path, names, cuts)
}

// QualityIn is the quality a name reads as, for display callers that
// hold no Content Type.
func QualityIn(name string) Fact[Quality] {
	return Title(name, TitleOnly{}, nil).Facts.Quality
}
