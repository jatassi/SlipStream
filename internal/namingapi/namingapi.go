// Package namingapi is each Content Type's naming settings over the API:
// the templates and modes, their tokens and a preview of a draft.
package namingapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/works"
)

type Namer interface {
	// Naming is each Content Type's naming, false unless the Node is a
	// Controller.
	Naming() (*naming.Naming, bool)
}

func open(n Namer) *naming.Naming {
	nm, ok := n.Naming()
	if !ok {
		panic("namingapi: an operation ran on a Node that is no Controller")
	}
	return nm
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	templateRefused     = api.NewCode(http.StatusUnprocessableEntity, "naming_template_refused", "The naming settings don't pass the save check")
)

type problem struct {
	Key      string `json:"key" doc:"The setting key, such as <type>.naming.file.standard"`
	Sentence string `json:"sentence" doc:"What is wrong, then what passes"`
}

type typeNaming struct {
	ContentType string                        `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Settings    []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The rename switch, the folder templates outermost first, the file templates, the colon mode and its text, and the multi-episode style when the type's file templates number its Items"`
	Problems    []problem                     `json:"problems" nullable:"false" doc:"What the save check refuses in the stored settings once renaming is on; empty when every file of a Work shows as one item on Plex, Jellyfin and Emby"`
}

type namingList struct {
	ContentTypes []typeNaming `json:"contentTypes" nullable:"false" doc:"In id order"`
}

type namingListOutput struct{ Body namingList }

type typeNamingOutput struct{ Body typeNaming }

type contentTypeInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type setNamingInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

type token struct {
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases" nullable:"false" doc:"Other names that match the same token"`
	Kind     string   `json:"kind" enum:"title,year,number,date,text,id,list"`
	Scope    string   `json:"scope" enum:"work,level,item,file" doc:"The tier the value comes from"`
	Group    string   `json:"group" enum:"work,level,item,target,release,media" doc:"The token builder's heading"`
	Example  string   `json:"example" doc:"A value the token prints"`
	About    string   `json:"about" doc:"What the token prints, as a sentence"`
	Modifier *string  `json:"modifier" enum:"pad,cut,languages,formats,format" doc:"What it takes after a colon; null when it takes nothing"`
	Tags     []string `json:"tags" nullable:"false" doc:"The words of the tags that may wrap it, as in {imdb-{ImdbId}}"`
}

type fieldTokens struct {
	Key    string  `json:"key" doc:"The template setting's key"`
	Tokens []token `json:"tokens" nullable:"false" doc:"Every token the template may name, the Content Type's own first"`
}

type tokenList struct {
	Fields []fieldTokens `json:"fields" nullable:"false" doc:"Each template setting, in the settings' order"`
}

type tokenListOutput struct{ Body tokenList }

type namingPreviewInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        struct {
		Draft map[string]json.RawMessage `json:"draft" doc:"Values over the stored settings by key, as a save would send them; null reads a setting's default"`
	}
}

type samplePath struct {
	Label string `json:"label" doc:"The sample, and for a file template the file it names"`
	Path  string `json:"path" doc:"The path in the root, with \"/\" between folders"`
}

type fieldPreview struct {
	Key      string       `json:"key"`
	Samples  []samplePath `json:"samples" nullable:"false" doc:"The full paths a folder or file template names for the type's samples, with renaming on; empty for any other setting and for a template that doesn't parse"`
	Problems []string     `json:"problems" nullable:"false" doc:"What the save check refuses in the setting, each a sentence; empty when it passes"`
}

type preview struct {
	Fields []fieldPreview `json:"fields" nullable:"false" doc:"Every setting, in the settings' order"`
}

type previewOutput struct{ Body preview }

func Routes(r *api.Router, n Namer) {
	o := r.Owner("naming", api.Admin)
	h := handlers{n}

	api.Register(o, api.Admin, api.Op{
		ID:      "listNaming",
		Method:  http.MethodGet,
		Path:    "/naming",
		Summary: "List each Content Type's naming settings, with what the save check refuses once renaming is on",
		Status:  http.StatusOK,
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID:      "setNaming",
		Method:  http.MethodPatch,
		Path:    "/naming/{contentType}",
		Summary: "Change a Content Type's naming settings by key, all or none. Every setting must pass the save check as the change leaves them",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, templateRefused, settingapi.ErrUnknown},
	}, h.set)

	api.Register(o, api.Admin, api.Op{
		ID:      "listNamingTokens",
		Method:  http.MethodGet,
		Path:    "/naming/{contentType}/tokens",
		Summary: "List the tokens each of a Content Type's templates may name",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.tokens)

	api.Register(o, api.Admin, api.Op{
		ID:      "previewNaming",
		Method:  http.MethodPost,
		Path:    "/naming/{contentType}/preview",
		Summary: "Name a Content Type's samples under a draft of its naming settings, with what the save check refuses. Nothing is stored",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, settingapi.ErrUnknown},
	}, h.preview)
}

type handlers struct{ namer Namer }

func (h handlers) list(ctx context.Context, _ identity.Admin, _ *struct{}) (*namingListOutput, error) {
	nm := open(h.namer)
	out := namingList{ContentTypes: []typeNaming{}}
	for _, t := range nm.Types() {
		tn, err := typeNamingOf(ctx, nm, t)
		if err != nil {
			return nil, err
		}
		out.ContentTypes = append(out.ContentTypes, tn)
	}
	return &namingListOutput{Body: out}, nil
}

func (h handlers) set(ctx context.Context, _ identity.Admin, in *setNamingInput) (*typeNamingOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	if err := nm.Panel(t).Apply(ctx, in.Body.Values); err != nil {
		return nil, refused(err)
	}
	tn, err := typeNamingOf(ctx, nm, t)
	if err != nil {
		return nil, err
	}
	return &typeNamingOutput{Body: tn}, nil
}

func (h handlers) tokens(_ context.Context, _ identity.Admin, in *contentTypeInput) (*tokenListOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	fields, err := nm.Catalog(t)
	if err != nil {
		return nil, err
	}
	out := tokenList{Fields: make([]fieldTokens, len(fields))}
	for i, f := range fields {
		ft := fieldTokens{Key: f.Key, Tokens: make([]token, len(f.Tokens))}
		for j, e := range f.Tokens {
			ft.Tokens[j] = tokenOf(e)
		}
		out.Fields[i] = ft
	}
	return &tokenListOutput{Body: out}, nil
}

func (h handlers) preview(ctx context.Context, _ identity.Admin, in *namingPreviewInput) (*previewOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	p, err := nm.Preview(ctx, t, in.Body.Draft)
	if err != nil {
		return nil, settingapi.Refused(err)
	}
	out := preview{Fields: make([]fieldPreview, len(p.Fields))}
	for i, f := range p.Fields {
		fp := fieldPreview{Key: f.Key, Samples: []samplePath{}, Problems: []string{}}
		for _, s := range f.Samples {
			fp.Samples = append(fp.Samples, samplePath{Label: s.Label, Path: s.Path})
		}
		fp.Problems = append(fp.Problems, f.Problems...)
		out.Fields[i] = fp
	}
	return &previewOutput{Body: out}, nil
}

// typeOf is the Naming and the Content Type id, which must be one Naming
// names files for.
func (h handlers) typeOf(id string) (*naming.Naming, works.TypeID, error) {
	nm, t := open(h.namer), works.TypeID(id)
	if !slices.Contains(nm.Types(), t) {
		return nil, "", contentTypeNotFound
	}
	return nm, t, nil
}

func typeNamingOf(ctx context.Context, nm *naming.Naming, t works.TypeID) (typeNaming, error) {
	fields, err := nm.Panel(t).Fields(ctx)
	if err != nil {
		return typeNaming{}, err
	}
	problems, err := nm.Problems(ctx, t)
	if err != nil {
		return typeNaming{}, err
	}
	out := typeNaming{ContentType: string(t), Settings: settingapi.FieldsOf(fields), Problems: []problem{}}
	for _, p := range problems {
		out.Problems = append(out.Problems, problem{Key: p.Key, Sentence: p.Sentence})
	}
	return out, nil
}

// refused answers a *naming.RefusedError with one field error per
// problem, at the value's key.
func refused(err error) error {
	r, ok := errors.AsType[*naming.RefusedError](err)
	if !ok {
		return settingapi.Refused(err)
	}
	fields := make([]api.FieldError, len(r.Problems))
	for i, p := range r.Problems {
		fields[i] = api.FieldError{Location: "body.values." + p.Key, Message: p.Sentence}
	}
	return templateRefused.Fields(fields...)
}

var (
	kinds = map[naming.Kind]string{
		naming.KindTitle: "title", naming.KindYear: "year", naming.KindNumber: "number", naming.KindDate: "date",
		naming.KindText: "text", naming.KindID: "id", naming.KindList: "list",
	}
	scopes = map[naming.Scope]string{naming.WorkScope: "work", naming.LevelScope: "level", naming.ItemScope: "item", naming.FileScope: "file"}
	groups = map[naming.Group]string{
		naming.GroupWork: "work", naming.GroupLevel: "level", naming.GroupItem: "item",
		naming.GroupTarget: "target", naming.GroupRelease: "release", naming.GroupMedia: "media",
	}
	modifiers = map[naming.Modifier]string{
		naming.ModPad: "pad", naming.ModCut: "cut", naming.ModLanguages: "languages", naming.ModFormats: "formats", naming.ModFormat: "format",
	}
)

func tokenOf(e naming.Entry) token {
	out := token{
		Name: e.Name, Aliases: append([]string{}, e.Aliases...), Kind: kinds[e.Kind], Scope: scopes[e.Scope], Group: groups[e.Group],
		Example: e.Example, About: e.About, Tags: append([]string{}, e.Tags...),
	}
	if m, ok := modifiers[e.Modifier]; ok {
		out.Modifier = &m
	}
	return out
}
