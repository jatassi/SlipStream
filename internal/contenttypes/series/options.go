package series

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// orderChoice is a series' episode order, a fetch option. Store writes it
// from the record a fetch in that order made.
var orderChoice = setting.Choice[tvdb.Order]{Key: "order", Label: "Episode order", Default: tvdb.OrderAired, Options: []setting.Option[tvdb.Order]{
	{Value: tvdb.OrderAired, Label: orderWords[tvdb.OrderAired]},
	{Value: tvdb.OrderDVD, Label: orderWords[tvdb.OrderDVD]},
	{Value: tvdb.OrderAbsolute, Label: orderWords[tvdb.OrderAbsolute]},
}}

var orderWords = map[tvdb.Order]string{tvdb.OrderAired: "Aired", tvdb.OrderDVD: "DVD", tvdb.OrderAbsolute: "Absolute"}

// airDateSwitch names a series' episodes by air date, a non-fetch option
// StoreOptions writes. Absolute order numbers episodes, so the two
// exclude each other.
var airDateSwitch = setting.Switch{Key: "nameByAirDate", Label: "Name by air date"}

// options is FetchOptions' encoding, read by the series' Source. Order is
// empty for Aired. RatedAt is when OMDb last answered for each Aired
// season, in Unix microseconds, so a ratings-due refresh reads the
// seasons that have gone longest without a rating next.
type options struct {
	Order   tvdb.Order      `json:"order,omitempty"`
	RatedAt map[int64]int64 `json:"ratedAt,omitempty"`
}

// OptionFields is w's episode order, offering only the orders TVDB lists
// the series in, and whether it is named by air date.
func (*Series) OptionFields(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]setting.Field, error) {
	o, err := optionsOf(ctx, r, w)
	if err != nil {
		return nil, err
	}
	c := orderChoice
	c.Options = slices.DeleteFunc(slices.Clone(c.Options), func(opt setting.Option[tvdb.Order]) bool { return !slices.Contains(o.orders, opt.Value) })
	var order *tvdb.Order
	if o.order != orderChoice.Default {
		order = &o.order
	}
	var byAirDate *bool
	if o.byAirDate {
		byAirDate = &o.byAirDate
	}
	return []setting.Field{c.Field(order), airDateSwitch.Field(byAirDate)}, nil
}

// FetchOptions is w's episode order and when each of its seasons' ratings
// were last read, nil for an Aired series OMDb hasn't rated a season of.
func (*Series) FetchOptions(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]byte, error) {
	o, err := optionsOf(ctx, r, w)
	if err != nil {
		return nil, err
	}
	return encodeOptions(ctx, r, w, o.order)
}

// PatchOptions is the fetch options p's order makes of w's,
// works.ErrOrderUnavailable for an order TVDB doesn't list the series in.
func (*Series) PatchOptions(ctx context.Context, r storage.Reader[file], w works.WorkID, p setting.Patch) ([]byte, error) {
	o, err := patched(ctx, r, w, p)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(o.orders, o.order) {
		return nil, works.ErrOrderUnavailable
	}
	return encodeOptions(ctx, r, w, o.order)
}

// StoreOptions writes whether p names w by air date.
func (*Series) StoreOptions(ctx context.Context, tx *storage.Tx[file], w works.WorkID, p setting.Patch) (bool, error) {
	was, err := optionsOf(ctx, tx, w)
	if err != nil {
		return false, err
	}
	o, err := was.with(p)
	if err != nil || o.byAirDate == was.byAirDate {
		return false, err
	}
	return true, db.New(tx).SetNameByAirDate(ctx, db.SetNameByAirDateParams{NameByAirDate: o.byAirDate, WorkID: w})
}

// picked is a series' options: its order among the orders TVDB lists it
// in, and whether it is named by air date.
type picked struct {
	order     tvdb.Order
	orders    []tvdb.Order
	byAirDate bool
}

func optionsOf(ctx context.Context, r storage.Reader[file], w works.WorkID) (picked, error) {
	row, err := db.New(r).GetSeries(ctx, w)
	if err != nil {
		return picked{}, err
	}
	o := picked{order: tvdb.Order(row.EpisodeOrder), byAirDate: row.NameByAirDate}
	if err := json.Unmarshal(row.Orders, &o.orders); err != nil {
		return picked{}, fmt.Errorf("series %d: orders: %w", row.TvdbID, err)
	}
	return o, nil
}

func patched(ctx context.Context, r storage.Reader[file], w works.WorkID, p setting.Patch) (picked, error) {
	o, err := optionsOf(ctx, r, w)
	if err != nil {
		return picked{}, err
	}
	return o.with(p)
}

// with is o as p changes it. It refuses Absolute order with Name by air
// date on, on the key p changes.
func (o picked) with(p setting.Patch) (picked, error) {
	for key, raw := range p {
		switch key {
		case orderChoice.Key:
			v, err := orderChoice.ParseJSON(raw)
			if err != nil {
				return picked{}, err
			}
			o.order = orderChoice.Value(v.Override())
		case airDateSwitch.Key:
			v, err := airDateSwitch.ParseJSON(raw)
			if err != nil {
				return picked{}, err
			}
			o.byAirDate = airDateSwitch.Value(v.Override())
		default:
			return picked{}, &setting.UnknownKeyError{Key: key}
		}
	}
	if o.order != tvdb.OrderAbsolute || !o.byAirDate {
		return o, nil
	}
	if _, ok := p[orderChoice.Key]; ok {
		return picked{}, &setting.RangeError{Key: orderChoice.Key, Sentence: "Absolute order names episodes by number. Turn off Name by air date to use it."}
	}
	return picked{}, &setting.RangeError{Key: airDateSwitch.Key, Sentence: "Name by air date doesn't work with Absolute order. Change the episode order to Aired or DVD first."}
}

func encodeOptions(ctx context.Context, r storage.Reader[file], w works.WorkID, order tvdb.Order) ([]byte, error) {
	rated, err := db.New(r).ListSeasonRatingsRead(ctx, w)
	if err != nil {
		return nil, err
	}
	o := options{}
	if order != orderChoice.Default {
		o.Order = order
	}
	for _, s := range rated {
		if o.RatedAt == nil {
			o.RatedAt = map[int64]int64{}
		}
		o.RatedAt[s.AiredSeason] = s.ReadAt.Time().UnixMicro()
	}
	if o.Order == "" && o.RatedAt == nil {
		return nil, nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("encode the options of work %s: %w", w, err)
	}
	return b, nil
}
