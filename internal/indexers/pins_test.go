package indexers

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexerfields"
)

func TestCarryTakesEachSettingByNameIntoWhereTheNewVersionKeepsIt(t *testing.T) {
	from := cardigann.Summary{Fields: []indexerfields.Field{
		{Name: "username", Type: indexerfields.Text},
		{Name: "pin", Type: indexerfields.Select, Options: []indexerfields.Option{{Value: "1", Label: "One"}}},
		{Name: "freeleech", Type: indexerfields.Checkbox},
		{Name: "gone", Type: indexerfields.Text},
	}}
	to := cardigann.Summary{Fields: []indexerfields.Field{
		{Name: "username", Type: indexerfields.Text},
		{Name: "password", Type: indexerfields.Password},
		{Name: "pin", Type: indexerfields.Text},
		{Name: "freeleech", Type: indexerfields.Checkbox},
		{Name: "info", Type: indexerfields.Info},
	}}
	got := carry(
		map[string]string{"username": "ada", "pin": "1", "freeleech": "true"},
		map[string]string{"password": "hunter2", "gone": "x"},
		from, to,
	)
	want := carried{
		plain:  map[string]string{"username": "ada", "freeleech": "true"},
		sealed: map[string]string{"password": "hunter2", "pin": "1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("carry = %+v, want %+v", got, want)
	}
}
