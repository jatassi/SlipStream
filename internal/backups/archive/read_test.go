package archive_test

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"slices"
	"testing"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"
	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/backups/archive"
)

func TestReadOpensWhatWriteWrote(t *testing.T) {
	f := large()
	b, sums := f.write(t, params)
	o, rec, err := read(t, b)
	if err != nil {
		t.Fatal(err)
	}
	got := o.Manifest()
	if got.BackupID != f.m.BackupID || !got.TakenAt.Equal(f.m.TakenAt) || got.Trigger != f.m.Trigger || got.Generation != 1 || got.Format != archive.Format {
		t.Fatalf("manifest = %+v, want %+v", got, f.m)
	}
	if o.Sums() != sums {
		t.Fatalf("Read's sums differ from Write's")
	}
	if sums.Size != int64(len(b)) || sums.Ciphertext != blake3.Sum256(b) {
		t.Fatalf("sums name %d bytes, want the %d written", sums.Size, len(b))
	}
	for e := archive.LibraryEntry; e <= archive.IndexEntry; e++ {
		if !bytes.Equal(rec.got[e].Bytes(), f.body(e)) || !rec.closed[e] {
			t.Fatalf("%s reached the Sink as %d bytes, closed %v; want its %d", e.Name(), rec.got[e].Len(), rec.closed[e], len(f.body(e)))
		}
		if sums.Entries[e] != blake3.Sum256(f.body(e)) {
			t.Fatalf("the sum of %s isn't its plaintext's", e.Name())
		}
	}
	if rec.discarded {
		t.Fatal("an opened Backup was discarded")
	}
	checked, err := archive.Check(bytes.NewReader(b), pass(t), limits)
	if err != nil || checked.Sums() != sums {
		t.Fatalf("Check = %v; want the sums Write returned", err)
	}
}

func TestStandardReadersOpenIt(t *testing.T) {
	f := small()
	b, _ := f.write(t, params)
	id, err := age.NewScryptIdentity(words)
	if err != nil {
		t.Fatal(err)
	}
	ar, err := age.Decrypt(bytes.NewReader(b), id)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zstd.NewReader(ar)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	var names []string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
		if e := archive.Entry(len(names) - 1); e == archive.ManifestEntry {
			var m map[string]any
			if err := json.Unmarshal(body, &m); err != nil || m["format"] != float64(1) || m["trigger"] != "scheduled" {
				t.Fatalf("manifest.json = %s, %v", body, err)
			}
		} else if !bytes.Equal(body, f.body(e)) {
			t.Fatalf("%s reads back as %q", hdr.Name, body)
		}
	}
	want := []string{"manifest.json", "library.db", "library.key", "backup.key", "backups.json"}
	if !slices.Equal(names, want) {
		t.Fatalf("entries = %q, want %q", names, want)
	}
}

func TestAWrongPassphraseIsItsOwnError(t *testing.T) {
	b, _ := small().write(t, params)
	other, err := archive.ParsePassphrase("zoo zone zero zebra youth young you yellow year yard")
	if err != nil {
		t.Fatal(err)
	}
	rec := newRecorder()
	o, err := archive.Read(bytes.NewReader(b), other, limits, rec)
	if !errors.Is(err, archive.ErrWrongPassphrase) {
		t.Fatalf("err = %v, want ErrWrongPassphrase", err)
	}
	refused(t, o, rec)
}

func TestADamagedChunkIsDamaged(t *testing.T) {
	b, _ := large().write(t, params)
	b[len(b)-1] ^= 1
	o, rec, err := read(t, b)
	damaged(t, err, archive.DamageChunk)
	refused(t, o, rec)
}

func TestACutFileIsDamaged(t *testing.T) {
	b, _ := large().write(t, params)
	header := bytes.Index(b, []byte("\n--- ")) + 50
	for _, n := range []int{header, header + 64<<10, len(b) / 2, len(b) - 17, len(b) - 1} {
		o, rec, err := read(t, b[:n])
		damaged(t, err, archive.DamageCut, archive.DamageChunk)
		refused(t, o, rec)
	}
}

func TestAnExtraEntryIsDamaged(t *testing.T) {
	files := small().files(t)
	o, rec, err := read(t, seal(t, append(files, tarFile{name: "extra.txt", body: []byte("planted")})))
	damaged(t, err, archive.DamageExtra)
	refused(t, o, rec)
}

func TestEntriesOutOfOrderAreDamaged(t *testing.T) {
	for name, files := range map[string]func([]tarFile) []tarFile{
		"swapped keys": func(f []tarFile) []tarFile { f[2], f[3] = f[3], f[2]; return f },
		"missing key":  func(f []tarFile) []tarFile { return append(f[:2], f[3:]...) },
		"a symlink":    func(f []tarFile) []tarFile { f[1] = tarFile{name: "library.db", typ: tar.TypeSymlink}; return f },
		"a path":       func(f []tarFile) []tarFile { f[1].name = "../library.db"; return f },
	} {
		t.Run(name, func(t *testing.T) {
			o, rec, err := read(t, seal(t, files(small().files(t))))
			damaged(t, err, archive.DamageOrder)
			refused(t, o, rec)
		})
	}
	o, rec, err := read(t, seal(t, small().files(t)[:4]))
	damaged(t, err, archive.DamageCut)
	refused(t, o, rec)
}

func TestAnOversizeEntryIsDamaged(t *testing.T) {
	key := small()
	key.libraryKey = make([]byte, 1<<20+1)
	lib := small()
	lib.library = make([]byte, limits.Library+1)
	for name, f := range map[string]fixture{"library.key": key, "library.db": lib} {
		t.Run(name, func(t *testing.T) {
			o, rec, err := read(t, seal(t, f.files(t)))
			damaged(t, err, archive.DamageOversize)
			refused(t, o, rec)
			if name == "library.db" && rec.got[archive.LibraryEntry] != nil {
				t.Fatal("an oversize library.db reached the Sink")
			}
		})
	}
}

func TestANonAgeFileIsNotABackup(t *testing.T) {
	plain := func() []byte {
		var out bytes.Buffer
		zw, _ := zstd.NewWriter(&out)
		tw := tar.NewWriter(zw)
		_ = tw.WriteHeader(&tar.Header{Name: "manifest.json", Typeflag: tar.TypeReg, Size: 2})
		_, _ = tw.Write([]byte("{}"))
		_ = errors.Join(tw.Close(), zw.Close())
		return out.Bytes()
	}()
	keyed := func() []byte {
		id, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		w, err := age.Encrypt(&out, id.Recipient())
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("not a library"))
		_ = w.Close()
		return out.Bytes()
	}()
	notManifest := seal(t, []tarFile{{name: "manifest.json", body: []byte("hello")}})
	for name, b := range map[string][]byte{"empty": nil, "random": random(4096), "tar.zst": plain, "age to a key": keyed, "no manifest": notManifest} {
		t.Run(name, func(t *testing.T) {
			o, rec, err := read(t, b)
			if !errors.Is(err, archive.ErrNotBackup) {
				t.Fatalf("err = %v, want ErrNotBackup", err)
			}
			refused(t, o, rec)
		})
	}
}

func TestANewerFormatIsTooNew(t *testing.T) {
	files := small().files(t)
	files[0].body = []byte(`{"format":2,"version":"2.3.0","takenAt":"2026-11-14T04:00:12Z","somethingNew":true}`)
	o, rec, err := read(t, seal(t, files))
	tooNew, ok := errors.AsType[*archive.TooNewError](err)
	if !ok || tooNew.Format != 2 || tooNew.Version != "2.3.0" {
		t.Fatalf("err = %v, want *TooNewError for format 2 from 2.3.0", err)
	}
	refused(t, o, rec)
}

func TestNothingIsOpenedBeforeTheLastChunk(t *testing.T) {
	f := large()
	b, _ := f.write(t, params)
	held := 100
	pr, pw := io.Pipe()
	rec := newRecorder()
	libraryClosed := make(chan struct{})
	rec.onClose = func(e archive.Entry) {
		if e == archive.LibraryEntry {
			close(libraryClosed)
		}
	}
	type result struct {
		o   archive.Opened
		err error
	}
	done := make(chan result, 1)
	go func() {
		o, err := archive.Read(pr, pass(t), limits, rec)
		done <- result{o, err}
	}()
	go func() { _, _ = pw.Write(b[:len(b)-held]) }()
	<-libraryClosed
	select {
	case r := <-done:
		t.Fatalf("Read returned %v before the last chunk arrived", r.err)
	default:
	}
	if got := rec.size(archive.LibraryEntry); got != len(f.library) {
		t.Fatalf("the Sink holds %d bytes of library.db, want all %d", got, len(f.library))
	}
	if _, err := pw.Write(b[len(b)-held:]); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	r := <-done
	if r.err != nil || r.o.Manifest().BackupID != f.m.BackupID {
		t.Fatalf("Read = %v once the last chunk arrived, want the Backup opened", r.err)
	}
}

func TestTheDecoderRefusesAHugeWindow(t *testing.T) {
	b, _ := large().write(t, archive.Params{ScryptLog: params.ScryptLog, ZstdWindow: 64 << 20})
	o, rec, err := read(t, b)
	damaged(t, err, archive.DamageZstd)
	refused(t, o, rec)
}

func TestTheWorkFactorIsCapped(t *testing.T) {
	b, _ := small().write(t, params)
	stanza := regexp.MustCompile(`(?m)^(-> scrypt \S+) 10$`)
	if !stanza.Match(b) {
		t.Fatal("no scrypt stanza at work factor 10")
	}
	b = stanza.ReplaceAll(b, []byte("$1 23"))
	o, rec, err := read(t, b)
	if !errors.Is(err, archive.ErrNotBackup) {
		t.Fatalf("err = %v, want ErrNotBackup for work factor 23", err)
	}
	refused(t, o, rec)
}
