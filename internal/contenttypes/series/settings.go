package series

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

const day = 24 * time.Hour

// AirDelay is the series setting, stored in series_settings: how long
// after its air date an episode counts as out.
var AirDelay = setting.Duration{Key: "series.airDelay", Label: "Delay after an air date", Max: 30 * day, Step: day}

func (*Series) SettingFields(ctx context.Context, r storage.Reader[file]) ([]setting.Field, error) {
	row, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{AirDelay.Field(row.AirDelaySeconds)}, nil
}

func (*Series) ApplySettings(ctx context.Context, tx *storage.Tx[file], p setting.Patch) (bool, error) {
	q := db.New(tx)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	next := row.AirDelaySeconds
	for _, key := range slices.Sorted(maps.Keys(p)) {
		if key != AirDelay.Key {
			return false, &setting.UnknownKeyError{Key: key}
		}
		s, err := AirDelay.ParseJSON(p[key])
		if err != nil {
			return false, err
		}
		next = s.Override()
	}
	if reflect.DeepEqual(next, row.AirDelaySeconds) {
		return false, nil
	}
	return true, q.SetAirDelay(ctx, next)
}
