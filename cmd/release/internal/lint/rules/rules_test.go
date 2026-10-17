package rules_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	_ "github.com/jatassi/slipstream/cmd/release/internal/lint/rules"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func fixtureRepo(t *testing.T, name string) *lint.Repo {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return lint.Open(t.Context(), root)
}

func check(t *testing.T, rule string, repo *lint.Repo) []string {
	t.Helper()
	r, ok := lint.Lookup(rule)
	if !ok {
		t.Fatalf("rule %s is not registered", rule)
	}
	var out strings.Builder
	if err := lint.Run(&out, repo, []lint.Rule{r}); err == nil {
		t.Fatal("lint passed a fixture with findings")
	}
	return strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
}

func assertFindings(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n%q\nwant:\n%q", got, want)
	}
}

func TestCgoConfined(t *testing.T) {
	repo := fixtureRepo(t, "cgo-confined")
	untracked := filepath.Join(repo.Root, "internal", "media", "untracked.go")
	if err := os.WriteFile(untracked, []byte("package media\n\nimport \"C\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertFindings(t, check(t, "cgo-confined", repo), []string{
		`internal/media/probe_windows.go:8: [cgo-confined] import "C" outside the declared native packages`,
		`internal/torrent/libtorrent/session/session.go:3: [cgo-confined] import "C" outside the declared native packages`,
	})
}

func TestReleaseImports(t *testing.T) {
	all := "linux/amd64, linux/arm64, windows/amd64, darwin/arm64"
	assertFindings(t, check(t, "release-imports", fixtureRepo(t, "release-imports")), []string{
		"cmd/slipstream/main.go:3: [release-imports] cmd/slipstream imports internal/app -> cmd/release/stamp on " + all,
		"cmd/slipstream/main.go:3: [release-imports] cmd/slipstream imports internal/app -> fakes/web on " + all,
		"cmd/slipstream/main.go:3: [release-imports] cmd/slipstream imports internal/app -> internal/storage/example/shelf on " + all,
		"cmd/slipstream/tmdb_windows.go:6: [release-imports] cmd/slipstream imports fakes/tmdb on windows/amd64",
	})
}

func TestParityOwners(t *testing.T) {
	assertFindings(t, check(t, "parity-owners", fixtureRepo(t, "parity-owners")), []string{
		"docs/parity-matrix.md:6: [parity-owners] P-002: expected one spec in Covered by, found 0",
		"docs/parity-matrix.md:7: [parity-owners] P-003: Status is open, expected planned or shipped",
		"docs/parity-matrix.md:7: [parity-owners] P-003: expected one spec in Covered by, found 2",
		"docs/parity-matrix.md:9: [parity-owners] P-005: Status is planned, expected -",
		"docs/parity-matrix.md:10: [parity-owners] P-006: unknown disposition open",
	})
}

func TestNativeLock(t *testing.T) {
	hint := "; " + native.PublishHint
	assertFindings(t, check(t, "native-lock", fixtureRepo(t, "native-lock")), []string{
		"native/lock.json:1: [native-lock] pins ffmpeg archive https://example.test/ffmpeg-old.tar.xz for linux-amd64, inputs list https://example.test/ffmpeg-linux.tar.xz; run scripts/native-deps pin",
		"native/lock.json:1: [native-lock] pins ffmpeg archive https://example.test/ffmpeg-win.zip for windows-amd64, which native/inputs.json does not list; run scripts/native-deps pin",
		"native/lock.json:1: [native-lock] pins libs layer 0000000000000000000000000000000000000000000000000000000000000000 for linux-amd64, inputs and recipe build 33c32da85e817d553bca283ba63360455c21ad0be9c02f857261c75b2ecfe946" + hint,
		"native/lock.json:1: [native-lock] pins libs layer 2222222222222222222222222222222222222222222222222222222222222222 for windows-amd64, which native/inputs.json does not list" + hint,
		"native/lock.json:1: [native-lock] pins no ffmpeg archive for darwin-arm64, inputs list https://example.test/ffmpeg-mac.tar.xz; run scripts/native-deps pin",
		"native/lock.json:1: [native-lock] pins no libs layer for darwin-arm64, inputs and recipe build 13fef6af146855c74b4206aa97728129f7231c1f0793afb381abc1310259fc84" + hint,
		"native/lock.json:1: [native-lock] pins no toolchain layer for linux-amd64, inputs and recipe build c871a31bd840d2719733f3d367edc4c1807ebcd6ba999b630294eb6bab409d39" + hint,
	})
}

func TestEndpointLiterals(t *testing.T) {
	const declare = " outside DefaultEndpoints; declare it in the package's Endpoints section"
	assertFindings(t, check(t, "endpoint-literals", fixtureRepo(t, "endpoint-literals")), []string{
		"internal/node/node.go:3: [endpoint-literals] host dns.google" + declare,
		"internal/tmdb/client.go:3: [endpoint-literals] host api.themoviedb.org" + declare,
		"internal/tmdb/client.go:6: [endpoint-literals] host tracker.opentrackr.org" + declare,
		"internal/tmdb/client.go:7: [endpoint-literals] host stun.l.google.com" + declare,
		"internal/tmdb/client.go:8: [endpoint-literals] host relay.slipstream.app" + declare,
		"internal/tmdb/client.go:9: [endpoint-literals] host router.bittorrent.com" + declare,
		"internal/tmdb/client.go:10: [endpoint-literals] host 8.8.8.8" + declare,
		"internal/tmdb/client.go:11: [endpoint-literals] host 2001:db8::1" + declare,
		"internal/tmdb/client.go:12: [endpoint-literals] host Docs.SlipStream.app" + declare,
		"internal/tmdb/client.go:13: [endpoint-literals] host tracker" + declare,
		"internal/tmdb/endpoints.go:16: [endpoint-literals] host image.tmdb.org" + declare,
		"internal/torrent/defaults.go:6: [endpoint-literals] host router.utorrent.com" + declare,
		"internal/torrent/defaults.go:10: [endpoint-literals] host dht.transmissionbt.com" + declare,
	})
}

func TestProcessGlobals(t *testing.T) {
	const (
		stream = " is process-global; draw from the subsystem's seeded stream from its Node"
		logger = " is process-global; log through the *slog.Logger the Node hands the subsystem"
		config = " is process-global; read the value from the Node's Config"
		listen = " is process-global; bind through the Node so readiness reports the address"
		client = " is process-global; build an *http.Client from the Node's config"
		serve  = " is process-global; serve on a listener the Node binds"
		mux    = " is process-global; register an operation with api.Register"
	)
	assertFindings(t, check(t, "process-globals", fixtureRepo(t, "process-globals")), []string{
		"internal/media/media.go:4: [process-globals] package log" + logger,
		"internal/media/media.go:6: [process-globals] package math/rand" + stream,
		"internal/media/media.go:14: [process-globals] math/rand/v2.IntN" + stream,
		"internal/media/media.go:15: [process-globals] math/rand/v2.NewPCG" + stream,
		"internal/media/media.go:16: [process-globals] math/rand/v2.N" + stream,
		"internal/media/media.go:17: [process-globals] math/rand/v2.NewChaCha8" + stream,
		"internal/media/media.go:20: [process-globals] os.Getenv" + config,
		"internal/media/media.go:21: [process-globals] os.LookupEnv" + config,
		"internal/media/media.go:22: [process-globals] os.Environ" + config,
		"internal/media/media.go:24: [process-globals] log/slog.Info" + logger,
		"internal/media/media.go:25: [process-globals] log/slog.SetDefault" + logger,
		"internal/media/media.go:29: [process-globals] net.Listen" + listen,
		"internal/media/media.go:30: [process-globals] net.ListenPacket" + listen,
		"internal/media/media.go:31: [process-globals] net.ListenConfig" + listen,
		"internal/media/media.go:33: [process-globals] net/http.Get" + client,
		"internal/media/media.go:34: [process-globals] net/http.DefaultClient" + client,
		"internal/media/media.go:35: [process-globals] net/http.ListenAndServe" + serve,
		"internal/media/media.go:37: [process-globals] net/http.Handle" + mux,
		"internal/media/media.go:38: [process-globals] net/http.HandleFunc" + mux,
		"internal/media/media.go:39: [process-globals] net/http.DefaultServeMux" + mux,
		"internal/media/shadow.go:15: [process-globals] os.Getenv" + config,
		"internal/media/shadow.go:16: [process-globals] net/http.Post" + client,
		"internal/seed/seed.go:3: [process-globals] package math/rand" + stream,
	})
}

func TestInstallMarker(t *testing.T) {
	const (
		read  = " [install-marker] install.Read named outside its owners; ProcessEnv reads the marker once; use Env.Install"
		write = " [install-marker] install.Write named outside its owners; only whoever places a binary writes its marker"
	)
	assertFindings(t, check(t, "install-marker", fixtureRepo(t, "install-marker")), []string{
		"cmd/slipstream/main.go:5:" + write,
		"internal/node/node.go:11:" + read,
		"internal/node/node.go:17:" + write,
		"internal/node/node.go:19:" + read,
		"internal/nodes/nodes.go:5:" + read,
	})
}

func TestDesktopConfined(t *testing.T) {
	const linux = " in a file the Linux build compiles; constrain it to windows || darwin"
	assertFindings(t, check(t, "desktop-confined", fixtureRepo(t, "desktop-confined")), []string{
		"internal/api/api_test.go:3: [desktop-confined] imports the desktop package, which only internal/node/app.go imports",
		"internal/media/media.go:3: [desktop-confined] imports fyne.io/systray outside internal/node/internal/desktop",
		"internal/node/internal/desktop/menu.go:3: [desktop-confined] imports fyne.io/systray" + linux,
		"internal/node/internal/desktop/notwindows.go:8: [desktop-confined] imports fyne.io/systray/internal/generated/menu" + linux,
		"internal/node/run.go:6: [desktop-confined] imports the desktop package, which only internal/node/app.go imports",
	})
}

func TestAppMain(t *testing.T) {
	const only = " [app-main] imports %s; the trampoline imports only std and golang.org/x/sys"
	files := " [app-main] the trampoline holds only main.go, launch_windows.go, launch_darwin.go"
	assertFindings(t, check(t, "app-main", fixtureRepo(t, "app-main")), []string{
		"cmd/slipstream-app/helper.go:1:" + files,
		"cmd/slipstream-app/launch_windows.go:4:" + fmt.Sprintf(only, "C"),
		"cmd/slipstream-app/launch_windows.go:7:" + fmt.Sprintf(only, "github.com/example/shell"),
		"cmd/slipstream-app/main.go:6:" + fmt.Sprintf(only, "example.com/app/internal/buildinfo"),
		"cmd/slipstream-app/stub_amd64.s:1:" + files,
	})
}

func TestAppMainWithoutMain(t *testing.T) {
	repo := fixtureRepo(t, "app-main")
	cmd := exec.CommandContext(t.Context(), "git", "rm", "-q", "-r", "-f", "cmd/slipstream-app/main.go", "cmd/slipstream-app/helper.go", "cmd/slipstream-app/stub_amd64.s", "cmd/slipstream-app/launch_windows.go")
	cmd.Dir = repo.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git rm: %v\n%s", err, out)
	}
	assertFindings(t, check(t, "app-main", repo), []string{
		"cmd/slipstream-app/main.go:1: [app-main] does not exist",
	})
}

func TestConfigSections(t *testing.T) {
	assertFindings(t, check(t, "config-sections", fixtureRepo(t, "config-sections")), []string{
		"fakes/contenttypes/music/music.go:9: [config-sections] DefaultTimings is never called by internal/node/config.go; wire the section into node.Config",
		"internal/contenttypes/tv/tv.go:5: [config-sections] DefaultEndpoints is never called by internal/node/config.go; wire the section into node.Config",
		"internal/media/media.go:5: [config-sections] DefaultSettings is never called by internal/node/config.go; wire the section into node.Config",
		"internal/node/timings.go:9: [config-sections] engine.DefaultSettings called outside internal/node/config.go; read the section from the Node's Config",
		"internal/tmdb/client.go:3: [config-sections] DefaultEndpoints called outside internal/node/config.go; read the section from the Node's Config",
		"internal/torrent/engine/engine.go:9: [config-sections] DefaultTimings is never called by internal/node/config.go; wire the section into node.Config",
	})
}

func TestConfigWriter(t *testing.T) {
	assertFindings(t, check(t, "config-writer", fixtureRepo(t, "config-writer")), []string{
		"internal/node/settings.go:12: [config-writer] WriteFile writes config.yaml; change keys through config.Edit",
		"internal/node/settings.go:15: [config-writer] Write writes config.yaml; change keys through config.Edit",
		"internal/node/settings.go:19: [config-writer] WriteFile writes config.yaml; change keys through config.Edit",
	})
}

func TestCredentialColumns(t *testing.T) {
	assertFindings(t, check(t, "credential-columns", fixtureRepo(t, "credential-columns")), []string{
		"internal/archive/schema/00001_archive.sql:4: [credential-columns] archive_feed.token holds a credential in plain text; seal it as token_sealed or keep only its digest as token_hash",
		"internal/vault/schema/00001_vault.sql:4: [credential-columns] vault_login.password holds a credential in plain text; seal it as password_sealed or keep only its digest as password_hash",
		"internal/vault/schema/00002_vault_cookie.sql:2: [credential-columns] vault_login.tracker_cookie holds a credential in plain text; seal it as tracker_cookie_sealed or keep only its digest as tracker_cookie_hash",
	})
}

func TestMigrationVersions(t *testing.T) {
	repo := fixtureRepo(t, "migration-versions")
	untracked := filepath.Join(repo.Root, "internal", "library", "schema", "00003_untracked.sql")
	if err := os.WriteFile(untracked, []byte("-- +goose Down\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertFindings(t, check(t, "migration-versions", repo), []string{
		"internal/library/schema/00002_library_tags.sql:1: [migration-versions] version 2 is also 00002_library_genres.sql",
		"internal/library/schema/00004_library_rename.sql:1: [migration-versions] version 4 follows 2; version 3 is missing",
		"internal/library/schema/00005_library_down.sql:4: [migration-versions] -- +goose Down found; migrations are forward-only (ADR 0015)",
		"internal/library/schema/00006_Library-Size.sql:1: [migration-versions] name must be NNNNN_slug.sql",
		"internal/library/schema/6_library_size.sql:1: [migration-versions] name must be NNNNN_slug.sql",
		"internal/media/schema/00002_media_late.sql:1: [migration-versions] version 2 follows 0; version 1 is missing",
	})
}

func TestQueryOrder(t *testing.T) {
	assertFindings(t, check(t, "query-order", fixtureRepo(t, "query-order")), []string{
		"internal/library/queries/library.sql:4: [query-order] ListWorks returns many rows without a top-level ORDER BY",
		"internal/library/queries/library.sql:15: [query-order] AnyWork uses LIMIT without a top-level ORDER BY",
		"internal/library/queries/library.sql:18: [query-order] ListRecentWorks returns many rows without a top-level ORDER BY",
		"internal/library/queries/library.sql:23: [query-order] ListRankedWorks returns many rows without a top-level ORDER BY",
		"internal/library/queries/library.sql:42: [query-order] ListFirstWorks returns many rows without a top-level ORDER BY",
		"internal/library/queries/library.sql:42: [query-order] ListFirstWorks uses LIMIT without a top-level ORDER BY",
	})
}

func TestSQLConfined(t *testing.T) {
	const statement = " outside generated queries; add the statement to queries/*.sql"
	const outside = " outside internal/storage and generated queries"
	assertFindings(t, check(t, "sql-confined", fixtureRepo(t, "sql-confined")), []string{
		"internal/media/late_header.go:5: [sql-confined] import database/sql" + outside,
		"internal/media/media.go:7: [sql-confined] import github.com/pressly/goose/v3" + outside,
		"internal/media/media.go:8: [sql-confined] import modernc.org/sqlite/lib" + outside,
		"internal/media/media.go:17: [sql-confined] PrepareContext" + statement,
		"internal/media/media.go:19: [sql-confined] QueryContext" + statement,
		"internal/media/mock.go:5: [sql-confined] import database/sql" + outside,
		"internal/media/mock.go:7: [sql-confined] ExecContext" + statement,
		"internal/storage/example/clip/clip.go:15: [sql-confined] ExecContext" + statement,
		"internal/storage/storagetest/audit.go:3: [sql-confined] import database/sql" + outside,
	})
}

func TestSqlcGenerated(t *testing.T) {
	const (
		typedID = " give the column a typed id override in sqlc.yaml"
		at      = " _at columns map to storage.Time"
		outside = " generated output outside <owner>/internal/db; set out: internal/db in the owner's sqlc.yaml"
	)
	assertFindings(t, check(t, "sqlc-generated", fixtureRepo(t, "sqlc-generated")), []string{
		"internal/library/db/models.go:1: [sqlc-generated]" + outside,
		"internal/library/internal/db/models.go:15: [sqlc-generated]" + typedID,
		"internal/library/internal/db/models.go:16: [sqlc-generated]" + typedID,
		"internal/library/internal/db/models.go:17: [sqlc-generated]" + typedID,
		"internal/library/internal/db/models.go:18: [sqlc-generated]" + typedID,
		"internal/library/internal/db/models.go:21: [sqlc-generated]" + at,
		"internal/library/internal/db/models.go:22: [sqlc-generated]" + at,
		"internal/library/internal/db/models.go:23: [sqlc-generated]" + at,
		"internal/library/internal/db/models.go:24: [sqlc-generated]" + at,
		"internal/library/internal/db/works.sql.go:20: [sqlc-generated]" + typedID,
		"internal/media/internal/db/models.go:1: [sqlc-generated]" + outside,
		"internal/media/internal/db/models.go:8: [sqlc-generated]" + at,
	})
}

func TestHumaConfined(t *testing.T) {
	const outside = " huma imported outside internal/api; register operations with api.Register"
	assertFindings(t, check(t, "huma-confined", fixtureRepo(t, "huma-confined")), []string{
		"fakes/web/web.go:3: [huma-confined]" + outside,
		"internal/api/apitest/walk.go:3: [huma-confined]" + outside,
		"internal/nodes/nodes_test.go:3: [huma-confined]" + outside,
		"internal/nodes/nodesapi/routes.go:6: [huma-confined]" + outside,
	})
}

func TestMailConfined(t *testing.T) {
	const outside = " go-mail imported outside internal/mail; send through a mail.Sender"
	assertFindings(t, check(t, "mail-confined", fixtureRepo(t, "mail-confined")), []string{
		"fakes/smtp/smtp_test.go:3: [mail-confined]" + outside,
		"internal/identity/emails.go:6: [mail-confined]" + outside,
		"internal/mail/mailtest/mailtest.go:3: [mail-confined]" + outside,
	})
}

func TestWebauthnConfined(t *testing.T) {
	const outside = " go-webauthn imported outside internal/identity/passkey; run ceremonies through passkey.Ceremonies"
	assertFindings(t, check(t, "webauthn-confined", fixtureRepo(t, "webauthn-confined")), []string{
		"internal/identity/passkey/passkeytest/passkeytest.go:3: [webauthn-confined]" + outside,
		"internal/identity/passkeys.go:6: [webauthn-confined]" + outside,
		"internal/identityapi/passkeys.go:3: [webauthn-confined]" + outside,
	})
}

func TestRoutesPlacement(t *testing.T) {
	assertFindings(t, check(t, "routes-placement", fixtureRepo(t, "routes-placement")), []string{
		"internal/nodes/nodes.go:6: [routes-placement] api.RegisterStream outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:13: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:19: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:22: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:24: [routes-placement] api.RegisterStream outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:27: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:28: [routes-placement] api.RegisterStream outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:29: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/nodes/nodesapi/routes.go:35: [routes-placement] api.Register outside <owner>api.Routes",
		"internal/tmdb/api/routes.go:6: [routes-placement] api.Register outside <owner>api.Routes",
	})
}

func TestSurfaceWired(t *testing.T) {
	assertFindings(t, check(t, "surface-wired", fixtureRepo(t, "surface-wired")), []string{
		"internal/nodes/nodes.go:16: [surface-wired] nodes.ChangedEvent is not listed in surface.Events",
		"internal/nodes/nodes.go:19: [surface-wired] nodes.RemovedEvent is not listed in surface.Events",
		"internal/nodes/nodesapi/routes.go:3: [surface-wired] nodesapi.Routes is never mounted in internal/node/surface",
	})
}

func TestSurfaceWiredWithoutSurface(t *testing.T) {
	repo := fixtureRepo(t, "surface-wired")
	if err := os.RemoveAll(filepath.Join(repo.Root, "internal", "node")); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "git", "add", "-A")
	cmd.Dir = repo.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	assertFindings(t, check(t, "surface-wired", repo), []string{
		"internal/identity/identity.go:7: [surface-wired] identity.SessionsChangedEvent is not listed in surface.Events",
		"internal/identity/identityapi/routes.go:3: [surface-wired] identityapi.Routes is never mounted in internal/node/surface",
		"internal/live/liveapi/routes.go:3: [surface-wired] liveapi.Routes is never mounted in internal/node/surface",
		"internal/nodes/nodes.go:15: [surface-wired] nodes.ListedEvent is not listed in surface.Events",
		"internal/nodes/nodes.go:16: [surface-wired] nodes.ChangedEvent is not listed in surface.Events",
		"internal/nodes/nodes.go:19: [surface-wired] nodes.RemovedEvent is not listed in surface.Events",
		"internal/nodes/nodesapi/routes.go:3: [surface-wired] nodesapi.Routes is never mounted in internal/node/surface",
	})
}

func TestLinkConfined(t *testing.T) {
	const (
		connect = " [link-confined] connectrpc.com/connect imported outside the Node link; call the service through its rpc descriptor"
		link    = " [link-confined] internal/link imported outside the Node runtime; call the service through its rpc descriptor"
	)
	assertFindings(t, check(t, "link-confined", fixtureRepo(t, "link-confined")), []string{
		"fakes/world/world.go:4:" + connect,
		"internal/node/node.go:4:" + connect,
		"internal/node/node.go:11: [link-confined] nodev1connect.RegisterSyncServiceHandler called outside the Node link; call the service through its rpc descriptor",
		"internal/node/surface/surface.go:3:" + link,
		"internal/role/roletest/roletest.go:6:" + link,
		"internal/search/client.go:12:" + connect,
		"internal/search/client.go:17: [link-confined] linkv1connect.NewLinkServiceClient called outside the Node link; call the service through its rpc descriptor",
		"internal/search/search_test.go:6:" + connect,
		"internal/search/transport.go:4: [link-confined] github.com/libp2p/go-yamux/v5 imported outside the Node link; call the service through its rpc descriptor",
		"internal/search/transport.go:5: [link-confined] golang.org/x/net/http2 imported outside the Node link; call the service through its rpc descriptor",
		"internal/search/transport.go:7:" + link,
	})
}

func TestRoleDefsWired(t *testing.T) {
	assertFindings(t, check(t, "role-defs-wired", fixtureRepo(t, "role-defs-wired")), []string{
		"internal/converting/converting.go:5: [role-defs-wired] internal/converting.Def is not held in internal/node",
		"internal/downloading/downloading.go:5: [role-defs-wired] internal/downloading.Def is not served in internal/node",
		"internal/fetching/fetching.go:6: [role-defs-wired] role downloading is declared twice",
		"internal/fetching/fetching.go:7: [role-defs-wired] internal/fetching.Named names its Role without a role constant",
	})
}

func TestFakesWired(t *testing.T) {
	assertFindings(t, check(t, "fakes-wired", fixtureRepo(t, "fakes-wired")), []string{
		"fakes/contenttypes/movie/moviemeta/moviemeta.go:5: [fakes-wired] fakes/contenttypes/movie/moviemeta.Fake is not listed in fakes/world",
		"fakes/contenttypes/movie/posters/posters.go:1: [fakes-wired] fakes/contenttypes/movie/posters declares no var Fake fake.Spec",
		"fakes/tmdb/doc.go:3: [fakes-wired] fakes/tmdb declares no var Fake fake.Spec",
		"fakes/torznab/torznab.go:7: [fakes-wired] fakes/torznab.Fake is not listed in fakes/world",
		"fakes/tracker/tracker_windows.go:5: [fakes-wired] fakes/tracker declares var Fake fake.Spec more than once",
	})
}

func TestConformanceSuites(t *testing.T) {
	assertFindings(t, check(t, "conformance-suites", fixtureRepo(t, "conformance-suites")), []string{
		"fakes/music/music.go:7: [conformance-suites] conformance/musicbrainz declares no func Run",
		"fakes/music/music.go:7: [conformance-suites] conformance/musicbrainz/online_test.go is missing or not built only under //go:build online",
		"fakes/music/music.go:7: [conformance-suites] no test in fakes/music calls conformance/musicbrainz.Run",
		"fakes/other/other.go:7: [conformance-suites] fake.OnlineSuite takes a string literal naming its suite under conformance/",
		"fakes/radarr/radarr.go:5: [conformance-suites] conformance/radarr declares no func Run",
		"fakes/radarr/radarr.go:5: [conformance-suites] conformance/radarr/online_test.go is missing or not built only under //go:build online",
		"fakes/radarr/radarr.go:5: [conformance-suites] no test in fakes/radarr calls conformance/radarr.Run",
	})
}

func TestDeadlineWaits(t *testing.T) {
	const (
		until    = " in the end-to-end tier; wait on a condition with worldtest.Until or on a record with events.Wait"
		context  = " in the end-to-end tier; take the context from worldtest.Context"
		unlinked = " without an issue link; a flaky test gets an issue first"
	)
	assertFindings(t, check(t, "deadline-waits", fixtureRepo(t, "deadline-waits")), []string{
		"cmd/fakeworld/fakeworld_test.go:9: [deadline-waits] time.Sleep" + until,
		"conformance/tmdb/online_test.go:8: [deadline-waits] t.Skip" + unlinked,
		"e2e/first_run_test.go:10: [deadline-waits] time.Sleep" + until,
		"e2e/first_run_test.go:11: [deadline-waits] time.After" + until,
		"e2e/first_run_test.go:12: [deadline-waits] context.Background" + context,
		"e2e/first_run_test.go:13: [deadline-waits] context.TODO" + context,
		"e2e/first_run_test.go:14: [deadline-waits] (*testing.T).Context" + context,
		"e2e/first_run_test.go:16: [deadline-waits] t.Skip" + unlinked,
		"e2e/first_run_test.go:17: [deadline-waits] t.Skipf" + unlinked,
		"e2e/first_run_test.go:20: [deadline-waits] t.Skip" + unlinked,
		"e2e/first_run_test.go:21: [deadline-waits] t.SkipNow" + unlinked,
		"e2e/first_run_test.go:22: [deadline-waits] (*testing.T).Context" + context,
		"e2e/first_run_test.go:26: [deadline-waits] (*testing.B).Context" + context,
		"e2e/first_run_test.go:31: [deadline-waits] tb.Skip" + unlinked,
		"e2e/helpers.go:10: [deadline-waits] time.NewTicker" + until,
		"e2e/helpers.go:13: [deadline-waits] (*testing.T).Context" + context,
		"e2e/helpers.go:14: [deadline-waits] time.Tick" + until,
		"fakes/events/log.go:5: [deadline-waits] time.Sleep" + until,
		"fakes/world/world.go:9: [deadline-waits] time.NewTimer" + until,
		"fakes/world/world.go:10: [deadline-waits] time.AfterFunc" + until,
		"fakes/world/world.go:12: [deadline-waits] context.Background" + context,
		"fakes/world/worldtest/launch.go:5: [deadline-waits] time.Sleep" + until,
	})
}

func TestWebLayout(t *testing.T) {
	const (
		route   = " signed-in pages live under web/src/routes/_app/ or web/src/routes/portal/; a public route is listed in web-layout"
		browser = " browser tests live under web/e2e/"
		comp    = " component tests live under web/src/; browser tests are web/e2e/*.spec.ts"
		apart   = " web/harness/ imports nothing from web/src/ or web/e2e/"
	)
	assertFindings(t, check(t, "web-layout", fixtureRepo(t, "web-layout")), []string{
		"web/e2e/x.test.ts:1: [web-layout]" + comp,
		"web/harness/bin/sign-in.ts:2: [web-layout]" + apart,
		"web/harness/manifest.ts:4: [web-layout]" + apart,
		"web/harness/routes/index.ts:1: [web-layout] web/harness/ is a library and holds no routes",
		"web/harness/sign-in.spec.ts:1: [web-layout]" + browser,
		"web/harness/sign-in.test.ts:1: [web-layout]" + comp,
		"web/src/app.tsx:3: [web-layout] the app never imports web/harness/",
		"web/src/routes/admin/users.tsx:1: [web-layout]" + route,
		"web/src/routes/movies.tsx:1: [web-layout]" + route,
		"web/src/routes/portals/home.tsx:1: [web-layout]" + route,
		"web/src/routes/setup.test.ts:1: [web-layout]" + route,
		"web/src/x.spec.ts:1: [web-layout]" + browser,
		"web/x.test.tsx:1: [web-layout]" + comp,
	})
}

func TestContentTypeConfined(t *testing.T) {
	const (
		facet   = "; ask its facet instead"
		surface = "internal/node/surface/surface.go:6: [content-type-confined] surface depends on internal/contenttype; the API document must not vary with registered Content Types"
	)
	assertFindings(t, check(t, "content-type-confined", fixtureRepo(t, "content-type-confined")), []string{
		"internal/contenttypes/book/book.go:7: [content-type-confined] contenttype.Define takes the Content Type id as a string literal",
		"internal/contenttypes/tv/tv.go:5: [content-type-confined] Content Type tv imports Content Type movie; Content Types never import each other",
		"internal/library/types.go:5: [content-type-confined] contenttype.Define outside a Content Type package",
		surface,
		"internal/search/plan.go:7: [content-type-confined] core imports fakes/contenttypes/music; reach a Content Type through its facet",
		"internal/search/plan.go:13: [content-type-confined] identifier music names Content Type music",
		"internal/works/api.go:3: [content-type-confined] identifier musicTracks names Content Type music",
		"internal/works/api.go:5: [content-type-confined] identifier HTTPMovieClient names Content Type movie",
		"internal/works/api.go:7: [content-type-confined] identifier tv_shows names Content Type tv",
		"internal/works/api.go:9: [content-type-confined] identifier musicTracks names Content Type music",
		"internal/works/tree.go:5: [content-type-confined] core names Content Type music" + facet,
		"internal/works/tree.go:7: [content-type-confined] core names Content Type tv" + facet,
		"internal/works/works_test.go:7: [content-type-confined] core names Content Type tv" + facet,
		"web/src/works/badge.tsx:4: [content-type-confined] core names Content Type music" + facet,
		"web/src/works/kinds.ts:4: [content-type-confined] core names Content Type movie" + facet,
		"web/src/works/kinds.ts:4: [content-type-confined] core names Content Type music" + facet,
		"web/src/works/kinds.ts:4: [content-type-confined] core names Content Type tv" + facet,
		"web/src/works/kinds.ts:6: [content-type-confined] core names Content Type movie" + facet,
		"web/src/works/kinds.ts:6: [content-type-confined] core names Content Type tv" + facet,
		"web/src/works/kinds.ts:7: [content-type-confined] core names Content Type tv" + facet,
	})
}

func TestMainsMatch(t *testing.T) {
	const extra = "cmd/slipstream-e2e/flags.go:1: [mains-match] a main holds only main.go"
	cases := []struct {
		name    string
		e2eMain string
		want    []string
	}{
		{
			name: "differs",
			want: []string{
				extra,
				"cmd/slipstream-e2e/main.go:11: [mains-match] differs from cmd/slipstream/main.go beyond registering test Content Types",
			},
		},
		{
			name: "registers music",
			e2eMain: "package main\n\nimport (\n\t\"os\"\n\n\tm \"example.com/app/fakes/contenttypes/music\"\n\t\"example.com/app/internal/node\"\n)\n\n" +
				"func main() {\n\tos.Exit(node.Main(os.Args, m.Def))\n}\n",
			want: []string{extra},
		},
		{
			name:    "registers nothing",
			e2eMain: "package main\n\nimport (\n\t\"os\"\n\n\t\"example.com/app/internal/node\"\n)\n\nfunc main() {\n\tos.Exit(node.Main(os.Args))\n}\n",
			want: []string{
				extra,
				"cmd/slipstream-e2e/main.go:1: [mains-match] registers no test Content Type",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := fixtureRepo(t, "mains-match")
			if tc.e2eMain != "" {
				if err := os.WriteFile(filepath.Join(repo.Root, "cmd", "slipstream-e2e", "main.go"), []byte(tc.e2eMain), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			assertFindings(t, check(t, "mains-match", repo), tc.want)
		})
	}
}

func TestMainsMatchWithoutE2EMain(t *testing.T) {
	repo := fixtureRepo(t, "mains-match")
	cmd := exec.CommandContext(t.Context(), "git", "rm", "-q", "-r", "-f", "cmd/slipstream-e2e")
	cmd.Dir = repo.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git rm: %v\n%s", err, out)
	}
	assertFindings(t, check(t, "mains-match", repo), []string{
		"cmd/slipstream-e2e/main.go:1: [mains-match] does not exist",
	})
}
