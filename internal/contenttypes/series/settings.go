package series

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

const day = 24 * time.Hour

// The series settings, stored in series_settings. AddNameByAirDate is
// whether a new series is named by air date, AirDelay how long after its
// air date an episode counts as out, and YearSlack how many years a
// release's year may differ from the series'.
var (
	AddNameByAirDate = setting.Switch{Key: "series.add.nameByAirDate", Label: airDateSwitch.Label}
	AirDelay         = setting.Duration{Key: "series.airDelay", Label: "Delay after an air date", Max: 30 * day, Step: day}
	YearSlack        = setting.Count{Key: "series.yearSlack", Label: "Release year can differ by", Default: 1, Min: 0, Max: 10, Unit: setting.Noun{One: "year", Many: "years"}}
)

func (*Series) SettingFields(ctx context.Context, r storage.Reader[file]) ([]setting.Field, error) {
	row, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{AddNameByAirDate.Field(row.AddNameByAirDate), AirDelay.Field(row.AirDelaySeconds), YearSlack.Field(row.YearSlack)}, nil
}

func (*Series) ApplySettings(ctx context.Context, tx *storage.Tx[file], p setting.Patch) (bool, error) {
	q := db.New(tx)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	was := db.SetSettingsParams{AirDelaySeconds: row.AirDelaySeconds, YearSlack: row.YearSlack, AddNameByAirDate: row.AddNameByAirDate}
	next := was
	stored := map[string]func(json.RawMessage) error{
		AddNameByAirDate.Key: into(AddNameByAirDate.ParseJSON, &next.AddNameByAirDate),
		AirDelay.Key:         into(AirDelay.ParseJSON, &next.AirDelaySeconds),
		YearSlack.Key:        into(YearSlack.ParseJSON, &next.YearSlack),
	}
	for _, key := range slices.Sorted(maps.Keys(p)) {
		parse, ok := stored[key]
		if !ok {
			return false, &setting.UnknownKeyError{Key: key}
		}
		if err := parse(p[key]); err != nil {
			return false, err
		}
	}
	if reflect.DeepEqual(next, was) {
		return false, nil
	}
	return true, q.SetSettings(ctx, next)
}

// into parses a value with parse and keeps its override in to.
func into[T any](parse func(json.RawMessage) (setting.Stored[T], error), to **T) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := parse(raw)
		*to = v.Override()
		return err
	}
}
