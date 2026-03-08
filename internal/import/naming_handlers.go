package importer

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/module"
)

const boolTrue = "true"

// ModuleNamingResponse represents the naming settings for a module.
type ModuleNamingResponse struct {
	ModuleType             string            `json:"moduleType"`
	RenameEnabled          bool              `json:"renameEnabled"`
	ColonReplacement       string            `json:"colonReplacement"`
	CustomColonReplacement string            `json:"customColonReplacement,omitempty"`
	Patterns               map[string]string `json:"patterns"`
	TokenContexts          []tokenContextDTO `json:"tokenContexts"`
	FormatOptions          []formatOptionDTO `json:"formatOptions"`
}

type tokenContextDTO struct {
	Name      string                `json:"name"`
	Label     string                `json:"label"`
	Variables []templateVariableDTO `json:"variables"`
	Variants  []string              `json:"variants,omitempty"`
	IsFolder  bool                  `json:"isFolder"`
}

type templateVariableDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
	DataKey     string `json:"dataKey"`
}

type formatOptionDTO struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	EnumValues   []string `json:"enumValues,omitempty"`
	DefaultValue string   `json:"defaultValue"`
}

// ModuleNamingUpdateRequest is the request body for updating module naming settings.
type ModuleNamingUpdateRequest struct {
	RenameEnabled          *bool              `json:"renameEnabled,omitempty"`
	ColonReplacement       *string            `json:"colonReplacement,omitempty"`
	CustomColonReplacement *string            `json:"customColonReplacement,omitempty"`
	Patterns               *map[string]string `json:"patterns,omitempty"`
}

// GetModuleNaming returns naming settings for a specific module.
// GET /api/v1/settings/:moduleId/naming
func (h *SettingsHandlers) GetModuleNaming(c echo.Context) error {
	ctx := c.Request().Context()
	moduleID := c.Param("moduleId")

	mod := h.registry.Get(module.Type(moduleID))
	if mod == nil {
		return echo.NewHTTPError(http.StatusNotFound, "module not found")
	}

	namingProvider, ok := mod.(module.NamingProvider)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "module does not support naming settings")
	}

	rows, err := h.queries.ListModuleNamingSettings(ctx, moduleID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	dbSettings := make(map[string]string)
	for _, row := range rows {
		dbSettings[row.SettingKey] = row.SettingValue
	}

	defaults := namingProvider.DefaultFileTemplates()
	patterns := make(map[string]string, len(defaults))
	for key, defaultVal := range defaults {
		if dbVal, ok := dbSettings[key]; ok {
			patterns[key] = dbVal
		} else {
			patterns[key] = defaultVal
		}
	}

	renameEnabled := true
	if v, ok := dbSettings["rename_enabled"]; ok {
		renameEnabled = v == boolTrue
	}

	colonReplacement := "smart"
	if v, ok := dbSettings["colon_replacement"]; ok {
		colonReplacement = v
	}

	customColonReplacement := ""
	if v, ok := dbSettings["custom_colon_replacement"]; ok {
		customColonReplacement = v
	}

	resp := ModuleNamingResponse{
		ModuleType:             moduleID,
		RenameEnabled:          renameEnabled,
		ColonReplacement:       colonReplacement,
		CustomColonReplacement: customColonReplacement,
		Patterns:               patterns,
		TokenContexts:          convertTokenContexts(namingProvider.TokenContexts()),
		FormatOptions:          convertFormatOptions(namingProvider.FormatOptions()),
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateModuleNaming updates naming settings for a specific module.
// PUT /api/v1/settings/:moduleId/naming
func (h *SettingsHandlers) UpdateModuleNaming(c echo.Context) error {
	ctx := c.Request().Context()
	moduleID := c.Param("moduleId")

	mod := h.registry.Get(module.Type(moduleID))
	if mod == nil {
		return echo.NewHTTPError(http.StatusNotFound, "module not found")
	}

	if _, ok := mod.(module.NamingProvider); !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "module does not support naming settings")
	}

	var req ModuleNamingUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.applyModuleNamingUpdates(ctx, moduleID, &req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if h.service != nil {
		if err := h.service.ReloadModuleRenamer(ctx, module.Type(moduleID)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to reload renamer: "+err.Error())
		}
	}

	return h.GetModuleNaming(c)
}

func (h *SettingsHandlers) applyModuleNamingUpdates(ctx context.Context, moduleID string, req *ModuleNamingUpdateRequest) error {
	upsert := func(key, value string) error {
		return h.queries.UpsertModuleNamingSetting(ctx, sqlc.UpsertModuleNamingSettingParams{
			ModuleType:   moduleID,
			SettingKey:   key,
			SettingValue: value,
		})
	}

	if err := h.upsertScalarNamingSettings(upsert, req); err != nil {
		return err
	}

	if req.Patterns != nil {
		for key, val := range *req.Patterns {
			if err := upsert(key, val); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *SettingsHandlers) upsertScalarNamingSettings(upsert func(string, string) error, req *ModuleNamingUpdateRequest) error {
	if req.RenameEnabled != nil {
		val := "false"
		if *req.RenameEnabled {
			val = boolTrue
		}
		if err := upsert("rename_enabled", val); err != nil {
			return err
		}
	}

	if req.ColonReplacement != nil {
		if err := upsert("colon_replacement", *req.ColonReplacement); err != nil {
			return err
		}
	}

	if req.CustomColonReplacement != nil {
		if err := upsert("custom_colon_replacement", *req.CustomColonReplacement); err != nil {
			return err
		}
	}

	return nil
}

// PreviewModuleNamingPattern previews a naming pattern with sample data for a module.
// POST /api/v1/settings/:moduleId/naming/preview
func (h *SettingsHandlers) PreviewModuleNamingPattern(c echo.Context) error {
	moduleID := c.Param("moduleId")

	mod := h.registry.Get(module.Type(moduleID))
	if mod == nil {
		return echo.NewHTTPError(http.StatusNotFound, "module not found")
	}

	var req PatternPreviewRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Pattern == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pattern is required")
	}

	resolver := renamer.NewResolver(defaultRenamerSettings())
	sampleCtx := renamer.GetSampleContext()

	resp := PatternPreviewResponse{
		Pattern: req.Pattern,
	}

	if err := resolver.ValidatePattern(req.Pattern); err != nil {
		resp.Valid = false
		resp.Error = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	preview, err := resolver.PreviewPattern(req.Pattern, sampleCtx)
	if err != nil {
		resp.Valid = false
		resp.Error = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	resp.Valid = true
	resp.Preview = preview

	breakdown := resolver.GetTokenBreakdown(req.Pattern, sampleCtx)
	resp.Tokens = make([]TokenBreakdown, len(breakdown))
	for i, b := range breakdown {
		resp.Tokens[i] = TokenBreakdown{
			Token:    b.Token,
			Name:     b.Name,
			Value:    b.Value,
			Empty:    b.Empty,
			Modified: b.Modified,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// ParseModuleFilename parses a filename for a specific module's naming context.
// POST /api/v1/settings/:moduleId/naming/parse
func (h *SettingsHandlers) ParseModuleFilename(c echo.Context) error {
	moduleID := c.Param("moduleId")

	mod := h.registry.Get(module.Type(moduleID))
	if mod == nil {
		return echo.NewHTTPError(http.StatusNotFound, "module not found")
	}

	var req ParseFilenameRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename is required")
	}

	resp := ParseFilenameResponse{
		Filename: req.Filename,
		Tokens:   []ParsedTokenDetail{},
	}

	parsed := parseFilename(req.Filename)
	if parsed == nil {
		return c.JSON(http.StatusOK, resp)
	}

	resp.ParsedInfo = parsedMediaInfoFromParsed(parsed)
	resp.Tokens = buildParsedTokens(parsed)

	return c.JSON(http.StatusOK, resp)
}

func convertTokenContexts(contexts []module.TokenContext) []tokenContextDTO {
	result := make([]tokenContextDTO, len(contexts))
	for i, ctx := range contexts {
		vars := make([]templateVariableDTO, len(ctx.Variables))
		for j, v := range ctx.Variables {
			vars[j] = templateVariableDTO{
				Name:        v.Name,
				Description: v.Description,
				Example:     v.Example,
				DataKey:     v.DataKey,
			}
		}
		variants := ctx.Variants
		if variants == nil {
			variants = []string{}
		}
		result[i] = tokenContextDTO{
			Name:      ctx.Name,
			Label:     ctx.Label,
			Variables: vars,
			Variants:  variants,
			IsFolder:  ctx.IsFolder,
		}
	}
	return result
}

func convertFormatOptions(options []module.FormatOption) []formatOptionDTO {
	result := make([]formatOptionDTO, len(options))
	for i, opt := range options {
		enumValues := opt.EnumValues
		if enumValues == nil {
			enumValues = []string{}
		}
		result[i] = formatOptionDTO{
			Key:          opt.Key,
			Label:        opt.Label,
			Description:  opt.Description,
			Type:         opt.Type,
			EnumValues:   enumValues,
			DefaultValue: opt.DefaultValue,
		}
	}
	return result
}
