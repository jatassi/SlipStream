package transfers

import (
	"slices"
	"testing"
)

func TestPlan(t *testing.T) {
	for _, tc := range []struct {
		name     string
		chunks   int
		verified []int
		want     []int
	}{
		{"an empty file", 0, nil, nil},
		{"a fresh file", 5, nil, []int{0, 1, 2, 3, 4}},
		{"broken at chunk 3 of 5", 5, []int{0, 1}, []int{2, 3, 4}},
		{"one chunk missing in the middle", 5, []int{0, 1, 3, 4}, []int{2}},
		{"every chunk verified", 5, []int{0, 1, 2, 3, 4}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verified := map[int]bool{}
			for _, i := range tc.verified {
				verified[i] = true
			}
			if got := plan(tc.chunks, verified); !slices.Equal(got, tc.want) {
				t.Errorf("plan(%d, %v) = %v, want %v", tc.chunks, tc.verified, got, tc.want)
			}
		})
	}
}
