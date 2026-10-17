package api

import (
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Optional is a query parameter a request may leave out, where its zero
// value would mean something: Set reports whether the request named it.
// The document describes it as a T.
type Optional[T any] struct {
	Value T
	Set   bool
}

func (Optional[T]) Schema(r huma.Registry) *huma.Schema {
	return r.Schema(reflect.TypeFor[T](), true, "")
}

func (o *Optional[T]) Receiver() reflect.Value { return reflect.ValueOf(o).Elem().Field(0) }

func (o *Optional[T]) OnParamSet(isSet bool, _ any) { o.Set = isSet }
