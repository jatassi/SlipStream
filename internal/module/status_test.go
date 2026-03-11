package module

import "testing"

func TestAggregateStatus_EmptySlice(t *testing.T) {
	got := AggregateStatus(nil)
	if got != StatusUnreleased {
		t.Errorf("AggregateStatus(nil) = %q, want %q", got, StatusUnreleased)
	}

	got = AggregateStatus([]string{})
	if got != StatusUnreleased {
		t.Errorf("AggregateStatus([]) = %q, want %q", got, StatusUnreleased)
	}
}

func TestAggregateStatus_SingleStatus(t *testing.T) {
	statuses := []string{
		StatusUnreleased, StatusAvailable, StatusUpgradable,
		StatusMissing, StatusFailed, StatusDownloading,
	}
	for _, s := range statuses {
		got := AggregateStatus([]string{s})
		if got != s {
			t.Errorf("AggregateStatus([%q]) = %q, want %q", s, got, s)
		}
	}
}

func TestAggregateStatus_HighestPriorityWins(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "downloading beats all",
			input:  []string{StatusAvailable, StatusMissing, StatusFailed, StatusDownloading, StatusUnreleased},
			expect: StatusDownloading,
		},
		{
			name:   "failed beats missing",
			input:  []string{StatusMissing, StatusFailed, StatusAvailable},
			expect: StatusFailed,
		},
		{
			name:   "missing beats upgradable",
			input:  []string{StatusUpgradable, StatusMissing},
			expect: StatusMissing,
		},
		{
			name:   "upgradable beats available",
			input:  []string{StatusAvailable, StatusUpgradable},
			expect: StatusUpgradable,
		},
		{
			name:   "available beats unreleased",
			input:  []string{StatusUnreleased, StatusAvailable},
			expect: StatusAvailable,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AggregateStatus(tc.input)
			if got != tc.expect {
				t.Errorf("AggregateStatus(%v) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestAggregateStatus_AllPriorityPairs(t *testing.T) {
	// Verify each adjacent pair in the priority chain.
	ordered := []string{
		StatusUnreleased, StatusAvailable, StatusUpgradable,
		StatusMissing, StatusFailed, StatusDownloading,
	}
	for i := 1; i < len(ordered); i++ {
		lower := ordered[i-1]
		higher := ordered[i]
		t.Run(higher+" > "+lower, func(t *testing.T) {
			got := AggregateStatus([]string{lower, higher})
			if got != higher {
				t.Errorf("AggregateStatus([%q, %q]) = %q, want %q", lower, higher, got, higher)
			}
			// Reverse order should give the same result.
			got = AggregateStatus([]string{higher, lower})
			if got != higher {
				t.Errorf("AggregateStatus([%q, %q]) = %q, want %q", higher, lower, got, higher)
			}
		})
	}
}

func TestAggregateStatus_UnknownStatusesIgnored(t *testing.T) {
	got := AggregateStatus([]string{"bogus", "invalid"})
	if got != StatusUnreleased {
		t.Errorf("AggregateStatus with only unknown statuses = %q, want %q", got, StatusUnreleased)
	}

	got = AggregateStatus([]string{"bogus", StatusAvailable, "invalid"})
	if got != StatusAvailable {
		t.Errorf("AggregateStatus with mixed unknown/known = %q, want %q", got, StatusAvailable)
	}
}
