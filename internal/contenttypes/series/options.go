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

// orderChoice is a series' one option, its episode order. Store writes it
// from the record a fetch in that order made.
var orderChoice = setting.Choice[tvdb.Order]{Key: "order", Label: "Episode order", Default: tvdb.OrderAired, Options: []setting.Option[tvdb.Order]{
	{Value: tvdb.OrderAired, Label: orderWords[tvdb.OrderAired]},
	{Value: tvdb.OrderDVD, Label: orderWords[tvdb.OrderDVD]},
	{Value: tvdb.OrderAbsolute, Label: orderWords[tvdb.OrderAbsolute]},
}}

var orderWords = map[tvdb.Order]string{tvdb.OrderAired: "Aired", tvdb.OrderDVD: "DVD", tvdb.OrderAbsolute: "Absolute"}

// options is FetchOptions' encoding, read by the series' Source.
type options struct {
	Order tvdb.Order `json:"order"`
}

// OptionFields is w's episode order, offering only the orders TVDB lists
// the series in.
func (*Series) OptionFields(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]setting.Field, error) {
	order, orders, err := ordersOf(ctx, r, w)
	if err != nil {
		return nil, err
	}
	c := orderChoice
	c.Options = slices.DeleteFunc(slices.Clone(c.Options), func(o setting.Option[tvdb.Order]) bool { return !slices.Contains(orders, o.Value) })
	var override *tvdb.Order
	if order != orderChoice.Default {
		override = &order
	}
	return []setting.Field{c.Field(override)}, nil
}

// FetchOptions is w's episode order, nil for Aired.
func (*Series) FetchOptions(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]byte, error) {
	order, _, err := ordersOf(ctx, r, w)
	if err != nil {
		return nil, err
	}
	return encodeOptions(order)
}

// PatchOptions is the fetch options p's order makes of w's,
// works.ErrOrderUnavailable for an order TVDB doesn't list the series in.
func (*Series) PatchOptions(ctx context.Context, r storage.Reader[file], w works.WorkID, p setting.Patch) ([]byte, error) {
	order, orders, err := ordersOf(ctx, r, w)
	if err != nil {
		return nil, err
	}
	for key, raw := range p {
		if key != orderChoice.Key {
			return nil, &setting.UnknownKeyError{Key: key}
		}
		v, err := orderChoice.ParseJSON(raw)
		if err != nil {
			return nil, err
		}
		order = orderChoice.Value(v.Override())
	}
	if !slices.Contains(orders, order) {
		return nil, works.ErrOrderUnavailable
	}
	return encodeOptions(order)
}

func ordersOf(ctx context.Context, r storage.Reader[file], w works.WorkID) (order tvdb.Order, orders []tvdb.Order, err error) {
	row, err := db.New(r).GetSeries(ctx, w)
	if err != nil {
		return "", nil, err
	}
	if err := json.Unmarshal(row.Orders, &orders); err != nil {
		return "", nil, fmt.Errorf("series %d: orders: %w", row.TvdbID, err)
	}
	return tvdb.Order(row.EpisodeOrder), orders, nil
}

func encodeOptions(o tvdb.Order) ([]byte, error) {
	if o == orderChoice.Default {
		return nil, nil
	}
	b, err := json.Marshal(options{Order: o})
	if err != nil {
		return nil, fmt.Errorf("encode the %s order: %w", o, err)
	}
	return b, nil
}
