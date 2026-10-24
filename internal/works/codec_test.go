package works_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

func everyKind() []works.FieldValue {
	return []works.FieldValue{
		works.TextField("origin", "Grandma"),
		works.NumberField("count", 2.5),
		works.DateField("packed", works.Date{Year: 2024, Month: 11}),
		works.DurationField("runtime", 90*time.Second),
		works.TextsField("tags", []string{"wool", "winter"}),
		works.PeopleField("makers", []works.Person{
			{Name: "Ada", Role: "Knitter", Department: "Craft", Photo: artwork.Ref{Source: "fixture", Shape: artwork.ShapePhoto, Path: "/ada.jpg"}},
			{Name: "Grace"},
		}),
		works.LinkField("page", works.Link{Label: "Shop", URL: "https://example.test/attic"}),
	}
}

func TestTheCodecRoundTripsEveryFieldKind(t *testing.T) {
	fields := everyKind()
	covered := map[works.FieldKind]bool{}
	for _, f := range fields {
		covered[f.Kind] = true
	}
	for k := works.FieldKind(1); k.String() != ""; k++ {
		if !covered[k] {
			t.Fatalf("no value of kind %s goes through the codec", k)
		}
	}

	candidates := []works.Candidate{{
		Key: "w1", Label: "The Attic", Original: "Le Grenier", Overview: "Boxes under the roof.",
		Poster: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/attic.jpg"}, Fields: fields,
	}}
	b, err := works.EncodeCandidates(candidates)
	if err != nil {
		t.Fatal(err)
	}
	gotCandidates, err := works.DecodeCandidates(b)
	if err != nil || !reflect.DeepEqual(gotCandidates, candidates) {
		t.Fatalf("candidates came back as %+v, %v; want %+v", gotCandidates, err, candidates)
	}

	fetched := works.Fetched{
		Label: "The Attic", SortLabel: "Attic, The",
		Children: []works.Entry{{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}}},
		Detail:   []byte{0, 1, 2, 0xff, '"'},
		Ratings: []works.Rating{
			{Source: "tmdb", Value: 7.5, Votes: votes(1200)},
			{Item: "c1", Source: "metacritic", Value: 70},
		},
		RatingsRead: true,
		Artwork:     []works.Art{{Kind: "photo", Ref: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/attic.jpg"}}},
	}
	b, err = works.EncodeFetched(fetched)
	if err != nil {
		t.Fatal(err)
	}
	gotFetched, err := works.DecodeFetched(b)
	if err != nil || !reflect.DeepEqual(gotFetched, fetched) {
		t.Fatalf("the record came back as %+v, %v; want %+v", gotFetched, err, fetched)
	}
	if _, err := works.DecodeFetched([]byte("{")); err == nil {
		t.Fatal("a truncated record decoded")
	}
}
