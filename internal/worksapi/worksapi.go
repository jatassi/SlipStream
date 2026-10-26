package worksapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/works"
)

type Shelf interface {
	// Works is the Work tree, false unless the Node is a Controller.
	Works() (*works.Works, bool)
	// Minter signs the artwork URLs the answers carry, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
	// Targets answers each Item's availability, false unless the Node is a
	// Controller.
	Targets() (*targets.Targets, bool)
	// Library composes the Library's reads and actions, false unless the
	// Node is a Controller.
	Library() (*library.Library, bool)
}

func open(s Shelf) *works.Works {
	w, ok := s.Works()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return w
}

func minter(s Shelf) artwork.Minter {
	m, ok := s.Minter()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return m
}

func goals(s Shelf) *targets.Targets {
	t, ok := s.Targets()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return t
}

func composer(s Shelf) *library.Library {
	l, ok := s.Library()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return l
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	contentTypeDisabled = api.NewCode(http.StatusConflict, "content_type_disabled", "That Content Type is turned off")
	contentTypeInUse    = api.NewCode(http.StatusConflict, "content_type_in_use", "Remove its Works before turning this Content Type off")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	workGoneUpstream    = api.NewCode(http.StatusConflict, "work_gone_upstream", "The metadata source no longer has this Work")
	metadataNotFound    = api.NewCode(http.StatusNotFound, "metadata_not_found", "The metadata source has no record with that key")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't respond. Try again later.")
	metadataInvalid     = api.NewCode(http.StatusBadGateway, "metadata_invalid", "The metadata source sent a record SlipStream can't use")
	metadataRefused     = api.NewCode(http.StatusBadGateway, "metadata_refused", "The metadata source refused SlipStream's key")
	profileNotFound     = api.NewCode(http.StatusUnprocessableEntity, "profile_not_found", "No Profile has that id")
	profileWrongType    = api.NewCode(http.StatusUnprocessableEntity, "profile_wrong_type", "The Profile is for another Content Type")
	rootNotFound        = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	rootWrongType       = api.NewCode(http.StatusUnprocessableEntity, "root_wrong_type", "The root holds another Content Type's Works")
	notImplemented      = api.NewCode(http.StatusNotImplemented, "not_implemented", "This route isn't built yet")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{works.ErrNoType, contentTypeNotFound},
	{works.ErrTypeOff, contentTypeDisabled},
	{works.ErrTypeInUse, contentTypeInUse},
	{works.ErrNoWork, workNotFound},
	{works.ErrGoneUpstream, workGoneUpstream},
	{works.ErrNotFound, metadataNotFound},
	{works.ErrUnavailable, metadataUnavailable},
	{works.ErrInvalid, metadataInvalid},
	{works.ErrRefused, metadataRefused},
	{targets.ErrNoProfile, profileNotFound},
	{targets.ErrProfileWrongType, profileWrongType},
	{roots.ErrNotFound, rootNotFound},
	{roots.ErrWrongType, rootWrongType},
	{library.ErrNotBuilt, notImplemented},
}

func answer(err error) error {
	if source, ok := errors.AsType[*works.SourceError](err); ok {
		switch {
		case errors.Is(err, works.ErrRefused):
			return metadataRefused.Because(source.Source + " refused SlipStream's key")
		case errors.Is(err, works.ErrUnavailable):
			return metadataUnavailable.Because(source.Error())
		}
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

func urlOrNull(m artwork.Minter, r artwork.Ref) *string {
	if r.IsZero() {
		return nil
	}
	u := m.URL(r)
	return &u
}

func idOrNull(id storage.UUID) *string {
	if id.IsZero() {
		return nil
	}
	s := id.String()
	return &s
}

func timeOrNull(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t.Time()
	return &v
}

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

func workID(s string) (works.WorkID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return works.WorkID{}, workNotFound
	}
	return works.WorkID{UUID: u}, nil
}

func workIDsOf(ss []string) ([]works.WorkID, error) {
	out := make([]works.WorkID, len(ss))
	for i, s := range ss {
		id, err := workID(s)
		if err != nil {
			return nil, err
		}
		out[i] = id
	}
	return out, nil
}

func Routes(r *api.Router, shelf Shelf) {
	o := r.Owner("works", api.Account)
	h := handlers{shelf}

	api.Register(o, api.Account, api.Op{
		ID:      "listContentTypes",
		Method:  http.MethodGet,
		Path:    "/content-types",
		Summary: "List the Content Types this install has, in id order, off ones included",
		Status:  http.StatusOK,
	}, h.listContentTypes)

	api.Register(o, api.Admin, api.Op{
		ID:      "setContentTypeEnabled",
		Method:  http.MethodPut,
		Path:    "/content-types/{contentType}/enabled",
		Summary: "Turn a Content Type on or off. It can be off only while it holds no Works",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeInUse},
	}, h.setContentTypeEnabled)

	api.Register(o, api.Admin, api.Op{
		ID:      "getContentTypeSettings",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/settings",
		Summary: "Get a Content Type's own settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.getContentTypeSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateContentTypeSettings",
		Method:  http.MethodPatch,
		Path:    "/content-types/{contentType}/settings",
		Summary: "Change a Content Type's own settings by key, all or none. Every Work of the type may then read differently",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateContentTypeSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "searchCatalog",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/catalog",
		Summary: "Search a Content Type's metadata source",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataUnavailable, metadataRefused},
	}, h.searchCatalog)

	api.Register(o, api.Admin, api.Op{
		ID:      "previewCatalogRecord",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/catalog/{key}",
		Summary: "Read a record of a Content Type's metadata source as the Work adding it would make, without adding it or reading ratings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataNotFound, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.previewCatalogRecord)

	api.Register(o, api.Admin, api.Op{
		ID:      "addWork",
		Method:  http.MethodPost,
		Path:    "/works",
		Summary: "Add a Work by its key with its whole tree, or get the Work already added",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, profileNotFound, profileWrongType, rootNotFound, rootWrongType, metadataNotFound, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.addWork)

	api.Register(o, api.Admin, api.Op{
		ID:      "listWorks",
		Method:  http.MethodGet,
		Path:    "/works",
		Summary: "List the Works, or one Content Type's",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.listWorks)

	api.Register(o, api.Admin, api.Op{
		ID:      "listLibrary",
		Method:  http.MethodGet,
		Path:    "/library",
		Summary: "List every Work as a Library row, or one Content Type's, or the named ones",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.listLibrary)

	api.Register(o, api.Admin, api.Op{
		ID:      "searchWorks",
		Method:  http.MethodGet,
		Path:    "/works/search",
		Summary: "Search the library's Works by every title they go by, best first",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.searchWorks)

	api.Register(o, api.Admin, api.Op{
		ID:      "getWork",
		Method:  http.MethodGet,
		Path:    "/works/{workId}",
		Summary: "Get a Work's tree with each node's fields, its links, Profile and root",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.getWork)

	api.Register(o, api.Admin, api.Op{
		ID:      "setWorksMonitored",
		Method:  http.MethodPut,
		Path:    "/works/monitored",
		Summary: "Monitor every Item of each Work, or none, with an outcome per Work",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notImplemented},
	}, h.setWorksMonitored)

	api.Register(o, api.Admin, api.Op{
		ID:      "setWorksProfile",
		Method:  http.MethodPut,
		Path:    "/works/profile",
		Summary: "Give each Work a Profile through one Impact, with an outcome per Work",
		Status:  http.StatusOK,
		Errors:  []*api.Code{profileNotFound, targetsapi.ImpactChanged, notImplemented},
	}, h.setWorksProfile)

	api.Register(o, api.Admin, api.Op{
		ID:      "setWorksRoot",
		Method:  http.MethodPut,
		Path:    "/works/root",
		Summary: "Move each Work without files to a root, with an outcome per Work",
		Status:  http.StatusOK,
		Errors:  []*api.Code{rootNotFound, notImplemented},
	}, h.setWorksRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "refreshWorks",
		Method:  http.MethodPost,
		Path:    "/works/refresh",
		Summary: "Refresh each Work from its metadata source, keeping every id, with an outcome per Work",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notImplemented},
	}, h.refreshWorks)

	api.Register(o, api.Admin, api.Op{
		ID:      "deleteWorks",
		Method:  http.MethodPost,
		Path:    "/works/delete",
		Summary: "Remove each Work from the library, deleting its files or leaving them, with an outcome per Work",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notImplemented},
	}, h.deleteWorks)

	api.Register(o, api.Admin, api.Op{
		ID:      "refreshWork",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/refresh",
		Summary: "Refresh a Work from its metadata source, keeping every id",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, workGoneUpstream, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.refreshWork)
}

type handlers struct{ shelf Shelf }
