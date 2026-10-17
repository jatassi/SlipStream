package release

// CutKey names a cut: a built-in's key, a Content Type's, or a custom
// cut's slug. Never "any".
type CutKey string

// Standard is the cut of a release or file that names none. It can't be
// renamed or removed.
const Standard CutKey = "standard"

// Cut is a cut and the phrases that name it in a title or file name. Only
// Aliases are matched, so every built-in but Standard lists its own name.
type Cut struct {
	Key     CutKey
	Name    string // the {edition-<Name>} tag uses it
	Aliases []string
}

// BuiltIn is every cut SlipStream knows without a Content Type or an Admin.
func BuiltIn() []Cut {
	return []Cut{
		{Standard, "Standard", nil},
		{"theatrical", "Theatrical", []string{"Theatrical", "Theatrical Cut", "Theatrical Edition", "Theatrical Version"}},
		{"extended", "Extended", []string{"Extended", "Extended Cut", "Extended Edition", "Extended Version"}},
		{"directors-cut", "Director's Cut", []string{"Director's Cut", "Director.s.Cut", "Director Cut", "Director's Edition", "Directors Edition"}},
		{"unrated", "Unrated", []string{"Unrated"}},
		{"uncut", "Uncut", []string{"Uncut"}},
		{"imax", "IMAX", []string{"IMAX", "IMAX Edition"}},
		{"remastered", "Remastered", []string{"Remastered"}},
		{"special-edition", "Special Edition", []string{"Special Edition"}},
		{"final-cut", "Final Cut", []string{"Final Cut"}},
		{"ultimate", "Ultimate", []string{"Ultimate", "Ultimate Cut", "Ultimate Edition"}},
	}
}
