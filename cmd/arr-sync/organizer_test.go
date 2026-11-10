package main

import (
	"strings"
	"testing"
)

const organizerSrc = `public class ColonFixture : CoreTest
{
    [TestCase("CSI: Vegas", ColonReplacementFormat.Dash, 2010, "CSI- Vegas")]
    [TestCase(null, ColonReplacementFormat.Smart, 0, "")]
    public void should_replace(string title, ColonReplacementFormat format, int year, string expected)
    {
    }

    [TestCase("x")]
    public void should_escape(string format)
    {
    }

    [Test]
    public void should_not_be_read()
    {
    }
}`

func organizerFixture(t *testing.T) fixture {
	t.Helper()
	s, err := lex("ColonFixture.cs", organizerSrc)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(s)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestOrganizerCasesFillArguments(t *testing.T) {
	of := organizerFile{
		File: "src/OrganizerTests/ColonFixture.cs",
		Methods: map[string]organizerMethod{"should_replace": {
			Template: "{Series Title} ({Series Year})",
			Values:   map[string]any{"Series Title": "${title}", "Series Year": "${year}", "Season Title": "Part ${year}?", "Season": 1},
			Settings: map[string]any{"colon": "${format}"},
			Want:     "${expected}!",
		}},
		Skip: map[string]string{"should_escape": "Sonarr's escape, which SlipStream refuses."},
	}
	rows, err := of.cases(organizerFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		line, err := marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, line)
	}
	want := []string{
		`{"template":"{Series Title} ({Series Year})","values":{"Season":1,"Season Title":"Part 2010?","Series Title":"CSI: Vegas","Series Year":2010},"settings":{"colon":"Dash"},"want":"CSI- Vegas!","source":"ColonFixture.cs should_replace #1"}`,
		`{"template":"{Series Title} ({Series Year})","values":{"Season":1,"Season Title":"Part 0?","Series Year":0},"settings":{"colon":"Smart"},"want":"!","source":"ColonFixture.cs should_replace #2"}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("cases\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestOrganizerCasesRefuseWhatThePinsDontCover(t *testing.T) {
	f := organizerFixture(t)
	for _, c := range []struct {
		of   organizerFile
		want string
	}{
		{organizerFile{Skip: map[string]string{"should_replace": "x"}}, "method should_escape is neither ported nor skipped"},
		{organizerFile{Skip: map[string]string{"should_replace": "x", "should_escape": "x", "should_go": "x"}}, "skips method should_go, which the file doesn't have"},
		{organizerFile{Skip: map[string]string{"should_escape": "x"}, Methods: map[string]organizerMethod{"should_replace": {Template: "${name}"}}}, "should_replace row 1: no parameter is named name"},
	} {
		if _, err := c.of.cases(f); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("cases(%+v) = %v; want an error naming %q", c.of, err, c.want)
		}
	}
}
