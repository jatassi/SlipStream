package indexersapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
)

var (
	catalogUnavailable        = api.NewCode(http.StatusServiceUnavailable, "catalog_unavailable", "SlipStream hasn't downloaded the Indexer catalog yet")
	catalogAddressUnreachable = api.NewCode(http.StatusUnprocessableEntity, "catalog_address_unreachable", "SlipStream can't read an Indexer catalog at that address")
	definitionUnloadable      = api.NewCode(http.StatusUnprocessableEntity, "definition_unloadable", "SlipStream can't run this definition yet")
	definitionIDTaken         = api.NewCode(http.StatusConflict, "definition_id_taken", "The Indexer catalog already has a definition with this id")
	definitionNotFound        = api.NewCode(http.StatusNotFound, "definition_not_found", "No definition has that id")
	definitionInUse           = api.NewCode(http.StatusConflict, "definition_in_use", "An Indexer runs this definition. Delete the Indexer first.")
)

func catalogAnswer(err error) error {
	var unrunnable *cardigann.Unrunnable
	var unreachable *indexers.CatalogUnreachableError
	switch {
	case errors.As(err, &unreachable):
		return catalogAddressUnreachable.Because("Check the address and that it serves schema 12.")
	case errors.As(err, &unrunnable):
		return definitionUnloadable.Because(capitalized(unrunnable.Reason) + ".")
	case errors.Is(err, indexers.ErrDefinitionIDTaken):
		return definitionIDTaken
	case errors.Is(err, indexers.ErrNoDefinition):
		return definitionNotFound
	case errors.Is(err, indexers.ErrDefinitionInUse):
		return definitionInUse
	}
	return err
}

// capitalized is a Go error string as the start of a sentence.
func capitalized(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:]
}

type catalogEntry struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description" doc:"The definition's own words"`
	Language    string    `json:"language" doc:"A language tag such as en-US"`
	Privacy     string    `json:"privacy" enum:"private,semi-private,public"`
	Categories  []int     `json:"categories" nullable:"false" doc:"The Newznab category ids it maps to"`
	Origin      string    `json:"origin" enum:"catalog,custom"`
	Reason      *string   `json:"reason" doc:"Why SlipStream can't run it; null when it can"`
	Replaced    bool      `json:"replacedByCustom" doc:"A custom definition has this catalog entry's id and runs in its place"`
	Version     time.Time `json:"version" doc:"When SlipStream downloaded this version"`
}

func entryOf(e indexers.Entry) catalogEntry {
	out := catalogEntry{
		ID: e.ID, Name: e.Name, Description: e.Description, Language: e.Language, Privacy: string(e.Privacy), Categories: []int{},
		Origin: string(e.Origin), Reason: text(e.Reason), Replaced: e.ReplacedBy, Version: e.Version,
	}
	for _, c := range e.Categories {
		out.Categories = append(out.Categories, int(c))
	}
	return out
}

type indexerCatalog struct {
	RefreshedAt   *time.Time     `json:"refreshedAt" doc:"Null until a refresh has downloaded the catalog"`
	NextRefreshAt *time.Time     `json:"nextRefreshAt" doc:"When the Indexer catalog refresh runs next; null while it runs or waits"`
	Address       string         `json:"address" doc:"The catalog's base address, to which SlipStream adds the schema version"`
	IsDefault     bool           `json:"addressIsDefault"`
	Notice        *string        `json:"notice" doc:"Why the catalog lists no site yet; null once it does"`
	Entries       []catalogEntry `json:"entries" nullable:"false" doc:"The catalog's definitions by name, then the custom definitions by id"`
}

func catalogOf(c indexers.CatalogView) indexerCatalog {
	out := indexerCatalog{
		RefreshedAt: when(c.Refreshed), NextRefreshAt: when(c.Next), Address: c.Address, IsDefault: c.IsDefault, Notice: text(c.Notice),
		Entries: []catalogEntry{},
	}
	for _, e := range c.Entries {
		out.Entries = append(out.Entries, entryOf(e))
	}
	return out
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type definitionField struct {
	Name    string   `json:"name" doc:"May hold hyphens or start with a digit"`
	Label   string   `json:"label"`
	Type    string   `json:"type" enum:"text,password,checkbox,select,info,info_cookie,info_flaresolverr,info_useragent,info_category_8000,cardigannCaptcha"`
	Default *string  `json:"default" doc:"Null when the field has none"`
	Options []option `json:"options" nullable:"false" doc:"A select's choices, by value"`
	Help    *string  `json:"help" doc:"An info field's text, written by the definition's authors"`
	Sealed  bool     `json:"sealed" doc:"Its saved value is never returned"`
}

type definitionForm struct {
	Entry   catalogEntry      `json:"entry"`
	Links   []string          `json:"links" nullable:"false" doc:"The site's addresses, the first being the default"`
	Fields  []definitionField `json:"fields" nullable:"false" doc:"In the definition's order, the captcha last"`
	Session bool              `json:"session" doc:"Its Indexers sign in and keep a session on their Node"`
	Captcha bool              `json:"captcha" doc:"Signing in asks for an image captcha"`
	Solver  bool              `json:"solver" doc:"The site sits behind a browser challenge that a Challenge solver can clear"`
	Cookie  bool              `json:"cookie" doc:"Its Indexers sign in with a browser cookie and its User-Agent, pasted as the cookie member, in place of a cookie field"`
}

func formOf(f indexers.Form) definitionForm {
	out := definitionForm{
		Entry: entryOf(f.Entry), Links: append([]string{}, f.Links...), Fields: []definitionField{},
		Session: f.Session, Captcha: f.Captcha, Solver: f.Solver, Cookie: f.Cookie,
	}
	for _, field := range f.Fields {
		d := definitionField{
			Name: field.Name, Label: field.Label, Type: string(field.Type), Default: text(field.Default), Options: []option{},
			Help: text(field.Help), Sealed: field.Sealed,
		}
		for _, o := range field.Options {
			d.Options = append(d.Options, option{Value: o.Value, Label: o.Label})
		}
		out.Fields = append(out.Fields, d)
	}
	return out
}

type catalogOutput struct{ Body indexerCatalog }

type definitionInput struct {
	ID string `path:"id" maxLength:"200"`
}

type formOutput struct{ Body definitionForm }

type catalogAddress struct {
	Address *string `json:"address" required:"true" nullable:"true" maxLength:"2048" doc:"The base address, to which SlipStream adds the schema version; null resets it to the default"`
}

type catalogAddressInput struct{ Body catalogAddress }

type customDefinition struct {
	YAML string `json:"yaml" minLength:"1" maxLength:"1048576" doc:"The definition's YAML file"`
}

type customDefinitionInput struct{ Body customDefinition }

type entryOutput struct{ Body catalogEntry }

func (h handlers) getCatalog(ctx context.Context, by identity.Admin, _ *struct{}) (*catalogOutput, error) {
	c, err := open(h.registry).Catalog(ctx, by)
	if err != nil {
		return nil, err
	}
	return &catalogOutput{Body: catalogOf(c)}, nil
}

func (h handlers) getDefinition(ctx context.Context, by identity.Admin, in *definitionInput) (*formOutput, error) {
	xs := open(h.registry)
	f, err := xs.Form(ctx, by, in.ID)
	if errors.Is(err, indexers.ErrCatalogUnavailable) {
		c, err := xs.Catalog(ctx, by)
		if err != nil {
			return nil, err
		}
		return nil, catalogUnavailable.Because(c.Notice)
	}
	if err != nil {
		return nil, catalogAnswer(err)
	}
	return &formOutput{Body: formOf(f)}, nil
}

func (h handlers) setAddress(ctx context.Context, by identity.Admin, in *catalogAddressInput) (*catalogOutput, error) {
	xs := open(h.registry)
	if err := xs.SetCatalogAddress(ctx, by, in.Body.Address); err != nil {
		return nil, catalogAnswer(err)
	}
	c, err := xs.Catalog(ctx, by)
	if err != nil {
		return nil, err
	}
	return &catalogOutput{Body: catalogOf(c)}, nil
}

func (h handlers) addCustom(ctx context.Context, by identity.Admin, in *customDefinitionInput) (*entryOutput, error) {
	xs := open(h.registry)
	e, err := xs.AddCustom(ctx, by, []byte(in.Body.YAML))
	if err != nil {
		return nil, catalogAnswer(err)
	}
	return &entryOutput{Body: entryOf(e)}, nil
}

func (h handlers) deleteCustom(ctx context.Context, by identity.Admin, in *definitionInput) (*struct{}, error) {
	xs := open(h.registry)
	return nil, catalogAnswer(xs.DeleteCustom(ctx, by, in.ID))
}
