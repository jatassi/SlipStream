package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadRegexes(t *testing.T) {
	const src = `namespace X
{
    public static class Parser
    {
        private const string Sep = "[-_. ]";
        private static readonly Regex[] Titles = new[]
            {
                // new Regex(@"commented out"),
                new Regex(@"^(?<title>.+?)" + Sep + @"S(?<season>\d+)""", RegexOptions.IgnoreCase | RegexOptions.Compiled),
                new Regex("tab\there"),
            };
        private static readonly Regex Words = new(@"(?<a>a)|(?<b>b)", RegexOptions.IgnoreCase);
        private static readonly Regex Wrapped = new(@$"(?:{Words})x{{2}}", RegexOptions.None);
        private static readonly RegexReplace Clean = new RegexReplace(@"\W", string.Empty, RegexOptions.Compiled);
        public static string Slug(string v) => Regex.Replace(v, @"\s", "-");
        public static bool Imdb(string id)
        {
            var imdbRegex = new Regex(@"^tt\d+$");
            return imdbRegex.IsMatch(id);
        }
    }
}`
	s, err := lex("Parser.cs", src)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := readRegexes(s, []string{"Slug"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]regexItem{}
	for _, d := range decls {
		got[d.name] = d.items
	}
	want := map[string][]regexItem{
		"Titles": {
			{text: `^(?<title>.+?)[-_. ]S(?<season>\d+)"`, options: []string{"IgnoreCase", "Compiled"}},
			{text: "tab\there"},
		},
		"Words":          {{text: `(?<a>a)|(?<b>b)`, options: []string{"IgnoreCase"}}},
		"Wrapped":        {{text: `(?:(?<a>a)|(?<b>b))x{2}`}},
		"Clean":          {{text: `\W`, options: []string{"Compiled"}}},
		"Imdb.imdbRegex": {{text: `^tt\d+$`}},
	}
	if a, b := mustJSON(t, got), mustJSON(t, want); a != b {
		t.Errorf("read\n%s\nwant\n%s", a, b)
	}
	if _, err := readRegexes(s, nil); err == nil || !strings.Contains(err.Error(), "Regex.Replace") {
		t.Errorf("an inline pattern in a method not skipped read without error: %v", err)
	}
}

func TestReadFixture(t *testing.T) {
	const src = `[TestFixture]
public class QualityFixture : CoreTest
{
    public static object[] Sources =
    {
        new object[] { "SD TV", Quality.SDTV },
    };

    [TestCase("Title.S01E01.720p", false, Description = "plain")]
    // [TestCase("commented out", true)]
    [TestCase(@"C:\Test\Title.mkv".AsOsAgnostic(), true)]
    public void should_parse(string title, bool proper)
    {
    }

    [Test]
    [TestCaseSource(nameof(Sources))]
    public void should_parse_source(string name, Quality quality)
    {
    }

    [TestCase("Title", new[] { 1, 2 }, -1.5)]
    public void should_parse_numbers(string title, int[] episodes, double part = 0)
    {
    }

    [Test]
    public void not_a_case_method()
    {
    }
}`
	s, err := lex("QualityFixture.cs", src)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(s)
	if err != nil {
		t.Fatal(err)
	}
	got := mustJSON(t, f.Methods)
	want := `[{"name":"should_parse","params":[{"name":"title","type":"string"},{"name":"proper","type":"bool"}],"cases":[{"args":["Title.S01E01.720p",false],"named":{"Description":"plain"}},{"args":[{"call":"AsOsAgnostic","on":"C:\\Test\\Title.mkv"},true]}]},` +
		`{"name":"should_parse_source","params":[{"name":"name","type":"string"},{"name":"quality","type":"Quality"}],"cases":[{"args":["SD TV",{"ref":"Quality.SDTV"}]}]},` +
		`{"name":"should_parse_numbers","params":[{"name":"title","type":"string"},{"name":"episodes","type":"int[]"},{"name":"part","type":"double","default":0}],"cases":[{"args":["Title",[1,2],-1.5]}]}]`
	if got != want {
		t.Errorf("read\n%s\nwant\n%s", got, want)
	}
}

func TestReadEnum(t *testing.T) {
	s, err := lex("Modifier.cs", "public enum Modifier { NONE = 0, REGIONAL, [Obsolete] SCREENER = 4, RAWHD }")
	if err != nil {
		t.Fatal(err)
	}
	values, err := readEnum(s, "Modifier")
	if err != nil {
		t.Fatal(err)
	}
	want := []enumValue{{"NONE", 0}, {"REGIONAL", 1}, {"SCREENER", 4}, {"RAWHD", 5}}
	if mustJSON(t, values) != mustJSON(t, want) {
		t.Errorf("read %+v, want %+v", values, want)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
