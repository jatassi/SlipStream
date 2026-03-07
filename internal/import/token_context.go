package importer

import (
	"time"

	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/module"
)

// buildTokenContextFromData constructs a renamer.TokenContext from a MatchedEntity's
// TokenData map, a ParseResult, and MediaInfo. This is the module-aware counterpart
// to the legacy buildTokenContext method.
func buildTokenContextFromData(entity *module.MatchedEntity, parsed *module.ParseResult, mi *mediainfo.MediaInfo) *renamer.TokenContext {
	ctx := &renamer.TokenContext{}
	applyEntityTokenData(ctx, entity)
	applyParseResultToTokenCtx(ctx, parsed)
	applyMediaInfoToTokenCtx(ctx, mi)
	return ctx
}

func applyEntityTokenData(ctx *renamer.TokenContext, entity *module.MatchedEntity) {
	d := entity.TokenData
	applyMovieTokenData(ctx, d)
	applyTVTokenData(ctx, d)

	if len(entity.EntityIDs) > 1 {
		if v, ok := d["EpisodeNumbers"].([]int); ok {
			ctx.EpisodeNumbers = v
		}
	}
}

func applyMovieTokenData(ctx *renamer.TokenContext, d map[string]any) {
	if v, ok := d["MovieTitle"].(string); ok {
		ctx.MovieTitle = v
	}
	if v, ok := d["MovieYear"].(int); ok {
		ctx.MovieYear = v
	}
}

func applyTVTokenData(ctx *renamer.TokenContext, d map[string]any) {
	if v, ok := d["SeriesTitle"].(string); ok {
		ctx.SeriesTitle = v
	}
	if v, ok := d["SeriesYear"].(int); ok {
		ctx.SeriesYear = v
	}
	if v, ok := d["SeriesType"].(string); ok {
		ctx.SeriesType = v
	}
	if v, ok := d["SeasonNumber"].(int); ok {
		ctx.SeasonNumber = v
	}
	if v, ok := d["EpisodeNumber"].(int); ok {
		ctx.EpisodeNumber = v
	}
	if v, ok := d["EpisodeTitle"].(string); ok {
		ctx.EpisodeTitle = v
	}
	if v, ok := d["AirDate"].(time.Time); ok {
		ctx.AirDate = v
	}
	if v, ok := d["AbsoluteNumber"].(int); ok {
		ctx.AbsoluteNumber = v
	}
}

func applyParseResultToTokenCtx(ctx *renamer.TokenContext, parsed *module.ParseResult) {
	if parsed == nil {
		return
	}
	ctx.Quality = parsed.Quality
	ctx.Source = parsed.Source
	ctx.Codec = parsed.Codec
	ctx.Revision = parsed.Revision
	ctx.ReleaseGroup = parsed.ReleaseGroup
	ctx.EditionTags = parsed.Edition
}

func applyMediaInfoToTokenCtx(ctx *renamer.TokenContext, mi *mediainfo.MediaInfo) {
	if mi == nil {
		return
	}
	ctx.VideoCodec = mi.VideoCodec
	ctx.VideoBitDepth = mi.VideoBitDepth
	ctx.VideoDynamicRange = mi.DynamicRange
	ctx.AudioCodec = mi.AudioCodec
	ctx.AudioChannels = mi.AudioChannels
	if len(mi.AudioLanguages) > 0 {
		ctx.AudioLanguages = mi.AudioLanguages
	}
	if len(mi.SubtitleLanguages) > 0 {
		ctx.SubtitleLanguages = mi.SubtitleLanguages
	}
}
