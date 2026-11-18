// Package subtitlesapi is the Subtitles operations: the section's
// settings and providers, a Storage Node's audio scans, a Work's
// subtitles, and the manual actions on one file's want.
package subtitlesapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targets"
)

type Owner interface {
	// Subtitles is the Controller's subtitles, false unless the Node is a
	// Controller.
	Subtitles() (*subtitles.Subtitles, bool)
	// Targets is the Targets and Profiles, false unless the Node is a
	// Controller.
	Targets() (*targets.Targets, bool)
}

func open(o Owner) *subtitles.Subtitles {
	s, ok := o.Subtitles()
	if !ok {
		panic("subtitlesapi: an operation ran on a Node that is no Controller")
	}
	return s
}

func goals(o Owner) *targets.Targets {
	t, ok := o.Targets()
	if !ok {
		panic("subtitlesapi: an operation ran on a Node that is no Controller")
	}
	return t
}

var (
	gapTooShort      = api.NewCode(http.StatusUnprocessableEntity, "gap_too_short", "Subtitle search runs less often than a subtitle that keeps coming up empty is searched for")
	providerNotFound = api.NewCode(http.StatusNotFound, "subtitle_provider_not_found", "No Subtitle Provider has that id")
	orderMismatch    = api.NewCode(http.StatusUnprocessableEntity, "provider_order_mismatch", "The order doesn't list every Subtitle Provider once")
	keyRejected      = api.NewCode(http.StatusUnprocessableEntity, "subtitle_key_rejected", "The Subtitle Provider rejected the key, so SlipStream didn't save it")
	keyless          = api.NewCode(http.StatusUnprocessableEntity, "subtitle_provider_keyless", "The Subtitle Provider takes no key")
	keyUnchecked     = api.NewCode(http.StatusBadGateway, "subtitle_key_unchecked", "SlipStream couldn't check the key")
	noHolder         = api.NewCode(http.StatusConflict, "no_subtitles_holder", "No Node has the Subtitles Role")
	notStorage       = api.NewCode(http.StatusNotFound, "not_a_storage_node", "No Node with the Storage Role has that id")
	badWant          = api.NewCode(http.StatusBadRequest, "bad_want", "The path doesn't name a subtitle language and kind")
	fileNotFound     = api.NewCode(http.StatusNotFound, "file_not_found", "No library file has that id")
	trackNotFound    = api.NewCode(http.StatusNotFound, "track_not_found", "The file has no text subtitle track at that position")
	notBuilt         = api.NewCode(http.StatusNotImplemented, "subtitle_action_not_built", "This build can't search for subtitles yet")
	searchExpired    = api.NewCode(http.StatusGone, "subtitle_search_expired", "The subtitle search expired")
	attemptExpired   = api.NewCode(http.StatusGone, "subtitle_attempt_expired", "The subtitle download or upload expired")
	uploadExpired    = api.NewCode(http.StatusGone, "subtitle_upload_expired", "The upload expired")
	tooLarge         = api.NewCode(http.StatusUnprocessableEntity, "subtitle_too_large", "The subtitle file is too large")
	image            = api.NewCode(http.StatusUnprocessableEntity, "subtitle_image", "The subtitle file holds image subtitles")
)

const notBuiltWords = "This build of SlipStream can't search for, download or upload subtitles yet."

func answer(err error) error {
	if err == nil {
		return nil
	}
	short, isShort := errors.AsType[*subtitles.GapTooShortError](err)
	refused, isRefused := errors.AsType[*subtitles.KeyRefusedError](err)
	unchecked, isUnchecked := errors.AsType[*subtitles.UnreachableError](err)
	large, isLarge := errors.AsType[*subtitles.TooLargeError](err)
	_, isNotStorage := errors.AsType[*subtitles.NotStorageError](err)
	_, isMismatch := errors.AsType[*subtitles.OrderMismatchError](err)
	switch {
	case isShort:
		return gapTooShort.On("values."+short.Key, short.Words())
	case isRefused:
		return keyRejected.On("key", refused.Provider+" rejected this key. It responded \""+refused.Words+"\". Check that you copied the whole key.")
	case isUnchecked:
		return keyUnchecked.Logged(unchecked, unchecked.Provider+" didn't respond, so SlipStream couldn't check the key. Try again later.")
	case isLarge:
		return tooLarge.On("bytes", large.Words())
	case isNotStorage:
		return notStorage
	case isMismatch:
		return orderMismatch.On("ids", "List every Subtitle Provider once.")
	case errors.Is(err, subtitles.ErrNoProvider):
		return providerNotFound
	case errors.Is(err, subtitles.ErrKeyless):
		return keyless
	case errors.Is(err, subtitles.ErrNoHolder):
		return noHolder.Because("No Node has the Subtitles Role, so SlipStream can't check the key. Assign it on the Nodes page.")
	case errors.Is(err, files.ErrNoFile):
		return fileNotFound
	case errors.Is(err, subtitles.ErrNoTrack):
		return trackNotFound
	case errors.Is(err, subtitles.ErrNotBuilt):
		return notBuilt.Because(notBuiltWords)
	case errors.Is(err, subtitles.ErrSearchExpired):
		return searchExpired.Because("These results have expired. Search again.")
	case errors.Is(err, subtitles.ErrAttemptExpired):
		return attemptExpired.Because("This download has expired. Search again.")
	case errors.Is(err, subtitles.ErrUploadExpired):
		return uploadExpired.Because("This upload has expired. Upload the file again.")
	case errors.Is(err, subtitles.ErrImage):
		return image.Because("SlipStream can't use image subtitles.")
	}
	return settingapi.Refused(err)
}

func fileID(s string) (files.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return files.ID{}, fileNotFound
	}
	return files.ID{UUID: u}, nil
}

func wantKey(file, want string) (subtitles.WantKey, error) {
	f, err := fileID(file)
	if err != nil {
		return subtitles.WantKey{}, err
	}
	w, err := subtitles.ParseWant(want)
	if err != nil {
		return subtitles.WantKey{}, badWant.Because(want + " isn't a subtitle language. Use a code such as en, pt-BR or en-forced.")
	}
	return subtitles.WantKey{File: f, Want: w}, nil
}

func uuidOf(s string, expired *api.Code, words string) (storage.UUID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return storage.UUID{}, expired.Because(words)
	}
	return u, nil
}

type handlers struct{ o Owner }
