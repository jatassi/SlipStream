package main

import "testing"

func TestReadAdds(t *testing.T) {
	const src = `public static class LanguageParser
{
    public static List<Language> ParseLanguages(string title)
    {
        var lowerTitle = title.ToLower();
        var languages = new List<Language>();
        if (lowerTitle.Contains("mandarin") || lowerTitle.Contains("chinese"))
        {
            languages.Add(Language.Chinese);
        }
        foreach (Match match in LanguageRegex.Matches(title))
        {
            if (match.Groups["english"].Success)
            {
                languages.Add(Language.English);
            }
        }
        if (!languages.Any())
        {
            languages.Add(Language.Unknown);
        }
        return languages;
    }
}`
	s, err := lex("LanguageParser.cs", src)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := readAdds(s, "ParseLanguages")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, d := range decls {
		got[d.name] = mustJSON(t, d.adds[0].words) + " " + d.adds[0].value
		if len(d.adds) != 1 {
			t.Errorf("%s read %d entries, want 1", d.name, len(d.adds))
		}
	}
	want := map[string]string{
		"ParseLanguages.Contains": `["mandarin","chinese"] Chinese`,
		"ParseLanguages.Groups":   `["english"] English`,
	}
	if mustJSON(t, got) != mustJSON(t, want) {
		t.Errorf("read %v, want %v", got, want)
	}
	if _, err := readAdds(s, "Missing"); err == nil {
		t.Error("a method the file doesn't declare read without error")
	}
}
