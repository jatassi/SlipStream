package release_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/jatassi/slipstream/internal/release"
)

func TestQualitySQLAgreesWithFile(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(t.Context(), `CREATE TABLE c (title TEXT, width INTEGER, height INTEGER)`); err != nil {
		t.Fatal(err)
	}
	titles := []string{
		"Orchard - WEB -- DL - 1080p", "Orchard.WEB - Rip", "Orchard [Blu]-[Ray]", "Orchard [WEBRip]", "Orchard (BluRay)", "Orchard{Remux}", "Orchard+HDTV",
		"Orchard_web_dl", "Orchard.WEB'DL", "Orchard’s.TS", "Orchard/ts\\cam", "ORCHARD.HDCAM", "Orchard.Tele.Sync",
		"Orchard\tTS", "Örchard.WEB-DL", "Orchard.ΤS", "Orchard.web dl", "  web  ", "web", "Orchard.dvd~rip", "Orchard.DVD",
	}
	for title := range titleQualities {
		titles = append(titles, title)
	}
	frames := [][2]int{{3840, 2160}, {3840, 1600}, {1920, 1080}, {1280, 720}, {1024, 576}, {720, 480}}
	for _, title := range titles {
		for _, f := range frames {
			if _, err := db.ExecContext(t.Context(), `INSERT INTO c VALUES (?, ?, ?)`, title, f[0], f[1]); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.ExecContext(t.Context(), `INSERT INTO c VALUES (?, NULL, NULL)`, title); err != nil {
			t.Fatal(err)
		}
	}
	query := `SELECT title, coalesce(width, 0), coalesce(height, 0), ` + release.QualitySQL("title", "width", "height") + ` FROM c` //nolint:gosec // the expression under test, built from constants
	rows, err := db.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	n := 0
	for rows.Next() {
		var title, got string
		var width, height int
		if err := rows.Scan(&title, &width, &height, &got); err != nil {
			t.Fatal(err)
		}
		n++
		if want := qualityKey(release.File(title, nil, release.ResolutionOf(width, height)).Quality); got != want {
			t.Errorf("QualitySQL(%q, %d, %d) = %q, File says %q", title, width, height, got, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := len(titles) * (len(frames) + 1); n != want {
		t.Fatalf("checked %d rows, want %d", n, want)
	}
}
