package indexers

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/cardigann"
)

func TestCarryTakesEachSettingByNameIntoWhereTheNewVersionKeepsIt(t *testing.T) {
	from := cardigann.Summary{Fields: []cardigann.Field{
		{Name: "username", Type: cardigann.TextField},
		{Name: "pin", Type: cardigann.SelectField, Options: []cardigann.Option{{Value: "1", Label: "One"}}},
		{Name: "freeleech", Type: cardigann.CheckboxField},
		{Name: "gone", Type: cardigann.TextField},
	}}
	to := cardigann.Summary{Fields: []cardigann.Field{
		{Name: "username", Type: cardigann.TextField},
		{Name: "password", Type: cardigann.PasswordField},
		{Name: "pin", Type: cardigann.TextField},
		{Name: "freeleech", Type: cardigann.CheckboxField},
		{Name: "info", Type: cardigann.InfoField},
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
