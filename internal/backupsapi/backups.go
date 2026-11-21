package backupsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
)

var (
	backupNotFound = api.NewCode(http.StatusNotFound, "backup_not_found", "No Backup has that id")
	newestVerified = api.NewCode(http.StatusConflict, "newest_verified", "The newest good Backup can't be deleted")
	notLocal       = api.NewCode(http.StatusConflict, "backup_not_local", "The Backup's file isn't on this machine")
	// needsSession refuses the API key at an Admin-session operation. It
	// isn't a 403, since the key passes the Admin level.
	needsSession = api.NewCode(http.StatusConflict, "passphrase_needs_session", "Only an Admin signed in to an Account can confirm the Backup passphrase")
)

type backupCheck struct {
	Result string     `json:"result" enum:"none,asked,ok,mismatch,unreadable,no_local_file" doc:"none: not checked yet; asked: waiting for its Node; no_local_file: this machine has no file of the Backup to check it against"`
	At     *time.Time `json:"at" doc:"When the result came in; null while none has"`
}

type backupCopy struct {
	DestinationID string      `json:"destinationId" pattern:"^[0-9a-f]{32}$"`
	State         string      `json:"state" enum:"waiting,copying,copied,failed"`
	CopiedAt      *time.Time  `json:"copiedAt" doc:"Null until it is copied"`
	Check         backupCheck `json:"check" doc:"Its last check"`
}

type backup struct {
	ID               string       `json:"id" pattern:"^[0-9a-f]{32}$"`
	Name             string       `json:"name" doc:"The Backup's file name"`
	TakenAt          time.Time    `json:"takenAt"`
	Trigger          string       `json:"trigger" enum:"scheduled,manual,update,before-restore,before-import,before-handover,superseded,restored" doc:"Why it was taken"`
	Version          string       `json:"version" doc:"The SlipStream version that took it"`
	Size             int64        `json:"size" doc:"Bytes"`
	State            string       `json:"state" enum:"taking,verified,failed,deleting" doc:"verified: it passed its check; failed: it counts for nothing and goes at the next Backup"`
	Local            bool         `json:"local" doc:"Its file is in this machine's Backups folder"`
	Copies           []backupCopy `json:"copies" nullable:"false" doc:"Its copy on each destination, in the order the destinations were added"`
	WhileReconciling bool         `json:"whileReconciling" doc:"Taken while SlipStream was Reconciling after a restore"`
}

type passphraseState struct {
	SavedAt *time.Time `json:"savedAt" doc:"When the Admin last showed they saved the current passphrase; null until then"`
}

type sheetState struct {
	DownloadedAt *time.Time `json:"downloadedAt" doc:"When the recovery sheet was last downloaded; null before"`
	Unlisted     []string   `json:"unlisted" nullable:"false" doc:"The destinations it doesn't list, every one before a download"`
}

type failure struct {
	Since time.Time `json:"since" doc:"When the first of the failed Backups since the newest good one was taken"`
	At    time.Time `json:"at" doc:"When the newest failed"`
}

type backupList struct {
	Backups    []backup              `json:"backups" nullable:"false" doc:"Newest first"`
	Used       int64                 `json:"used" doc:"Bytes the Backups in this machine's Backups folder take"`
	Failure    api.Nullable[failure] `json:"failure" doc:"Backups failing since the newest good one; null when the newest passed"`
	Passphrase passphraseState       `json:"passphrase"`
	Sheet      sheetState            `json:"sheet"`
}

type backupListOutput struct{ Body backupList }

type backupInput struct {
	ID string `path:"backupId" pattern:"^[0-9a-f]{32}$"`
}

type passphrase struct {
	Passphrase string `json:"passphrase" doc:"The ten words that open every new Backup"`
}

type passphraseOutput struct{ Body passphrase }

type replaceInput struct {
	Body struct {
		DeleteOlder bool `json:"deleteOlder" default:"true" doc:"Delete every older Backup, copies included, once the Backup of the new passphrase passes its check"`
	}
}

type checkInput struct {
	Body struct {
		Passphrase string `json:"passphrase" minLength:"1" maxLength:"1024"`
	}
}

type checked struct {
	Matches bool `json:"matches" doc:"It is the current passphrase, which now reads as saved"`
}

type checkedOutput struct{ Body checked }

type pullInput struct {
	ID   string `path:"backupId" pattern:"^[0-9a-f]{32}$"`
	Body struct {
		DestinationID string `json:"destinationId" pattern:"^[0-9a-f]{32}$"`
	}
}

type backupSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"How often SlipStream backs up, how long it keeps Backups, when Health warns, how long a copy waits and how often copies are checked"`
}

type backupSettingsOutput struct{ Body backupSettings }

type backupSettingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

func backupRoutes(o *api.Owner, h handlers) {
	api.Register(o, api.Admin, api.Op{
		ID: "listBackups", Method: http.MethodGet, Path: "/backups",
		Summary: "List the Backups, newest first, with their copies and the passphrase's state",
		Status:  http.StatusOK,
	}, h.listBackups)
	api.Register(o, api.Admin, api.Op{
		ID: "backUpNow", Method: http.MethodPost, Path: "/backups",
		Summary: "Start a Backup now. The Backup task's run shows how it went",
		Status:  http.StatusAccepted,
	}, h.backUpNow)
	api.Register(o, api.Admin, api.Op{
		ID: "deleteBackup", Method: http.MethodDelete, Path: "/backups/{backupId}",
		Summary: "Delete a Backup and its copies, and list the Backups left",
		Status:  http.StatusOK,
		Errors:  []*api.Code{backupNotFound, newestVerified},
	}, h.deleteBackup)
	api.Register(o, api.Admin, api.Op{
		ID: "pullBackup", Method: http.MethodPost, Path: "/backups/{backupId}/pull",
		Summary: "Copy a Backup back from a destination into this machine's Backups folder",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{backupNotFound, destinationNotFound},
	}, h.pullBackup)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "prepareBackupDownload", Method: http.MethodPost, Path: "/backups/{backupId}/file/ticket",
		Summary: "Confirm it's you before downloading a Backup's file",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{backupNotFound, notLocal},
	}, h.prepareDownload)
	api.RegisterDownload(o, api.ReconfirmedSession, api.Op{
		ID: "downloadBackup", Method: http.MethodGet, Path: "/backups/{backupId}/file",
		Summary: "Download a Backup's file as it was written, still encrypted",
		Status:  http.StatusOK,
		Errors:  []*api.Code{backupNotFound, notLocal},
	}, "application/octet-stream", h.downloadBackup)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "getBackupPassphrase", Method: http.MethodGet, Path: "/backups/passphrase",
		Summary: "Show the passphrase that opens every new Backup",
		Status:  http.StatusOK,
	}, h.getPassphrase)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "replaceBackupPassphrase", Method: http.MethodPost, Path: "/backups/passphrase",
		Summary: "Make a new Backup passphrase and start a Backup with it",
		Status:  http.StatusAccepted,
	}, h.replacePassphrase)
	api.Register(o, api.Admin, api.Op{
		ID: "saveBackupPassphrase", Method: http.MethodPost, Path: "/backups/passphrase/saved",
		Summary: "Record that you saved the current Backup passphrase",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{needsSession},
	}, h.savedPassphrase)
	api.Register(o, api.Admin, api.Op{
		ID: "checkBackupPassphrase", Method: http.MethodPost, Path: "/backups/passphrase/check",
		Summary: "Check a saved copy against the current Backup passphrase",
		Status:  http.StatusOK,
		Errors:  []*api.Code{needsSession},
	}, h.checkPassphrase)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "prepareRecoverySheet", Method: http.MethodPost, Path: "/backups/recovery-sheet/ticket",
		Summary: "Confirm it's you before downloading the recovery sheet",
		Status:  http.StatusNoContent,
	}, func(context.Context, identity.Admin, *struct{}) (*struct{}, error) { return nil, nil })
	api.RegisterDownload(o, api.ReconfirmedSession, api.Op{
		ID: "downloadRecoverySheet", Method: http.MethodGet, Path: "/backups/recovery-sheet",
		Summary: "Download the recovery sheet: the passphrase, each Backup folder and how to restore",
		Status:  http.StatusOK,
	}, "text/plain; charset=utf-8", h.downloadSheet)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "retireController", Method: http.MethodPost, Path: "/backups/retire",
		Summary: "Retire this Controller after its library was restored on another machine",
		Status:  http.StatusNoContent,
	}, h.retire)
	api.Register(o, api.Admin, api.Op{
		ID: "getBackupSettings", Method: http.MethodGet, Path: "/backup-settings",
		Summary: "Get the Backups settings",
		Status:  http.StatusOK,
	}, h.getSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "updateBackupSettings", Method: http.MethodPatch, Path: "/backup-settings",
		Summary: "Change Backups settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}

func (h handlers) listBackups(context.Context, identity.Admin, *struct{}) (*backupListOutput, error) {
	return &backupListOutput{Body: listOf(open(h.k).Ledger())}, nil
}

func listOf(l backups.Ledger) backupList {
	out := backupList{Backups: []backup{}, Sheet: sheetState{Unlisted: []string{}}}
	for _, e := range l.Backups {
		out.Backups = append(out.Backups, backupOf(l, e))
		if e.Local {
			out.Used += e.Size
		}
	}
	if f := l.Failure; f != nil {
		out.Failure = api.Some(failure{Since: f.Since, At: f.At})
	}
	if t := l.PassphraseConfirmedAt; !t.IsZero() {
		out.Passphrase.SavedAt = &t
	}
	if t := l.Sheet.DownloadedAt; !t.IsZero() {
		out.Sheet.DownloadedAt = &t
	}
	for _, d := range l.Unlisted() {
		out.Sheet.Unlisted = append(out.Sheet.Unlisted, d.ID.String())
	}
	return out
}

func backupOf(l backups.Ledger, e backups.Entry) backup {
	id, _ := e.ID.MarshalText()
	out := backup{
		ID: string(id), Name: e.Name.String(), TakenAt: e.TakenAt, Trigger: string(e.Trigger), Version: e.Version,
		Size: e.Size, State: string(e.State), Local: e.Local, Copies: []backupCopy{}, WhileReconciling: e.WhileReconciling,
	}
	for _, d := range l.Destinations {
		c, ok := e.Copy(d.ID)
		if !ok {
			continue
		}
		ch := checkOf(c.Check)
		bc := backupCopy{DestinationID: d.ID.String(), State: string(c.State), Check: backupCheck{Result: ch.Result, At: ch.At}}
		if !c.CopiedAt.IsZero() {
			bc.CopiedAt = &c.CopiedAt
		}
		out.Copies = append(out.Copies, bc)
	}
	return out
}

func (h handlers) backUpNow(ctx context.Context, by identity.Admin, _ *struct{}) (*struct{}, error) {
	req := backups.Request{Trigger: archive.Manual}
	if account, ok := by.Account(); ok {
		req.By = account.Profile().ID.UUID
	}
	_, err := open(h.k).Take(context.WithoutCancel(ctx), req)
	return nil, err
}

func parseBackup(s string) (archive.ID, error) {
	var id archive.ID
	if err := id.UnmarshalText([]byte(s)); err != nil {
		return id, backupNotFound
	}
	return id, nil
}

func (h handlers) deleteBackup(_ context.Context, _ identity.Admin, in *backupInput) (*backupListOutput, error) {
	id, err := parseBackup(in.ID)
	if err != nil {
		return nil, err
	}
	b := open(h.k)
	switch err := b.Delete(id); {
	case errors.Is(err, backups.ErrNoBackup):
		return nil, backupNotFound
	case errors.Is(err, backups.ErrNewestVerified):
		return nil, newestVerified.Because("This is your newest good Backup, so it can't be deleted. Take another Backup first.")
	case err != nil:
		return nil, err
	}
	return &backupListOutput{Body: listOf(b.Ledger())}, nil
}

func (h handlers) pullBackup(ctx context.Context, _ identity.Admin, in *pullInput) (*struct{}, error) {
	id, err := parseBackup(in.ID)
	if err != nil {
		return nil, err
	}
	from, err := backups.ParseDestinationID(in.Body.DestinationID)
	if err != nil {
		return nil, destinationNotFound
	}
	switch err := open(h.k).Pull(ctx, id, from); {
	case errors.Is(err, backups.ErrNoBackup):
		return nil, backupNotFound
	case errors.Is(err, backups.ErrNoDestination), errors.Is(err, backups.ErrNoCopy):
		return nil, destinationNotFound.Because("That destination has no copy of this Backup.")
	default:
		return nil, err
	}
}

// local is the Backup id names, which must have its file here.
func (h handlers) local(s string) (*backups.Backups, backups.Entry, error) {
	id, err := parseBackup(s)
	if err != nil {
		return nil, backups.Entry{}, err
	}
	b := open(h.k)
	l := b.Ledger()
	e, ok := l.Entry(id)
	switch {
	case !ok || e.State != backups.Verified:
		return nil, backups.Entry{}, backupNotFound
	case !e.Local:
		return nil, backups.Entry{}, notLocal.Because("Copy it back from a destination first.")
	}
	return b, e, nil
}

func (h handlers) prepareDownload(_ context.Context, _ identity.Admin, in *backupInput) (*struct{}, error) {
	_, _, err := h.local(in.ID)
	return nil, err
}

func (h handlers) downloadBackup(_ context.Context, _ identity.Admin, in *backupInput) (*api.Download, error) {
	b, e, err := h.local(in.ID)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(b.Path(e))
	if err != nil {
		return nil, fmt.Errorf("open the Backup: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	return &api.Download{Name: e.Name.String(), Size: info.Size(), Body: f}, nil
}

func (h handlers) getPassphrase(context.Context, identity.Admin, *struct{}) (*passphraseOutput, error) {
	p, err := open(h.k).Passphrase()
	if err != nil {
		return nil, err
	}
	return &passphraseOutput{Body: passphrase{Passphrase: p.Reveal()}}, nil
}

func (h handlers) replacePassphrase(ctx context.Context, by identity.Admin, in *replaceInput) (*struct{}, error) {
	account, _ := by.Account()
	_, err := open(h.k).Replace(context.WithoutCancel(ctx), account.Profile().ID.UUID, in.Body.DeleteOlder)
	return nil, err
}

func (h handlers) savedPassphrase(_ context.Context, by identity.Admin, _ *struct{}) (*struct{}, error) {
	if _, ok := by.Account(); !ok {
		return nil, needsSession
	}
	return nil, open(h.k).Saved()
}

func (h handlers) checkPassphrase(_ context.Context, by identity.Admin, in *checkInput) (*checkedOutput, error) {
	if _, ok := by.Account(); !ok {
		return nil, needsSession
	}
	matches, err := open(h.k).Check(in.Body.Passphrase)
	if err != nil {
		return nil, err
	}
	return &checkedOutput{Body: checked{Matches: matches}}, nil
}

func (h handlers) downloadSheet(context.Context, identity.Admin, *struct{}) (*api.Download, error) {
	sheet, err := open(h.k).RecoverySheet()
	if err != nil {
		return nil, err
	}
	return &api.Download{Name: backups.SheetFile, Size: int64(len(sheet)), Body: io.NopCloser(bytes.NewReader(sheet))}, nil
}

func (h handlers) retire(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
	return nil, h.k.Retire(ctx)
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*backupSettingsOutput, error) {
	return settingsOf(ctx, open(h.k).Settings())
}

func (h handlers) updateSettings(ctx context.Context, _ identity.Admin, in *backupSettingsInput) (*backupSettingsOutput, error) {
	p := open(h.k).Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func settingsOf(ctx context.Context, p setting.Panel) (*backupSettingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &backupSettingsOutput{Body: backupSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}
