package slots

import (
	"testing"
)

// Tests for computeAggregateStatus covering all priority combinations.
// Spec: docs/status-consolidation.md - "Cached Aggregate Computation"
// Priority order: downloading > failed > missing > upgradable > available > unreleased

// testService creates a minimal Service instance for testing the computeAggregateStatus method.
func testService() *Service {
	return &Service{}
}

func TestComputeAggregateStatus_AllSameStatus(t *testing.T) {
	s := testService()
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"All downloading", "downloading", "downloading"},
		{"All failed", "failed", "failed"},
		{"All missing", "missing", "missing"},
		{"All upgradable", "upgradable", "upgradable"},
		{"All available", "available", "available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots := []SlotStatus{
				{SlotID: 1, Monitored: true, Status: tt.status},
				{SlotID: 2, Monitored: true, Status: tt.status},
			}
			got := s.computeAggregateStatus(slots)
			if got != tt.expected {
				t.Errorf("computeAggregateStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestComputeAggregateStatus_PriorityOrder(t *testing.T) {
	// Spec: Priority order: downloading > failed > missing > upgradable > available > unreleased
	s := testService()

	tests := []struct {
		name     string
		statuses []string
		expected string
	}{
		{
			"downloading wins over everything",
			[]string{"downloading", "failed", "missing", "upgradable", "available"},
			"downloading",
		},
		{
			"failed wins over missing/upgradable/available",
			[]string{"failed", "missing", "upgradable", "available"},
			"failed",
		},
		{
			"missing wins over upgradable/available",
			[]string{"missing", "upgradable", "available"},
			"missing",
		},
		{
			"upgradable wins over available",
			[]string{"upgradable", "available"},
			"upgradable",
		},
		{
			"downloading + available = downloading",
			[]string{"downloading", "available"},
			"downloading",
		},
		{
			"failed + available = failed",
			[]string{"failed", "available"},
			"failed",
		},
		{
			"missing + available = missing",
			[]string{"missing", "available"},
			"missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots := make([]SlotStatus, len(tt.statuses))
			for i, status := range tt.statuses {
				slots[i] = SlotStatus{SlotID: int64(i + 1), Monitored: true, Status: status}
			}
			got := s.computeAggregateStatus(slots)
			if got != tt.expected {
				t.Errorf("computeAggregateStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestComputeAggregateStatus_UnmonitoredIgnored(t *testing.T) {
	// Spec: Unmonitored slots should not affect aggregate
	s := testService()

	tests := []struct {
		name     string
		slots    []SlotStatus
		expected string
	}{
		{
			"Unmonitored downloading ignored, monitored available",
			[]SlotStatus{
				{SlotID: 1, Monitored: false, Status: "downloading"},
				{SlotID: 2, Monitored: true, Status: "available"},
			},
			"available",
		},
		{
			"Unmonitored failed ignored, monitored missing",
			[]SlotStatus{
				{SlotID: 1, Monitored: false, Status: "failed"},
				{SlotID: 2, Monitored: true, Status: "missing"},
			},
			"missing",
		},
		{
			"Unmonitored missing ignored, monitored available",
			[]SlotStatus{
				{SlotID: 1, Monitored: false, Status: "missing"},
				{SlotID: 2, Monitored: true, Status: "available"},
			},
			"available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.computeAggregateStatus(tt.slots)
			if got != tt.expected {
				t.Errorf("computeAggregateStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestComputeAggregateStatus_AllUnmonitoredFallback(t *testing.T) {
	// Spec: "If no monitored slot assignments exist, fall back to unreleased"
	s := testService()

	slots := []SlotStatus{
		{SlotID: 1, Monitored: false, Status: "available"},
		{SlotID: 2, Monitored: false, Status: "missing"},
	}
	got := s.computeAggregateStatus(slots)
	if got != "unreleased" {
		t.Errorf("All unmonitored: computeAggregateStatus() = %q, want %q", got, "unreleased")
	}
}

func TestComputeAggregateStatus_EmptySlots(t *testing.T) {
	// No slots at all → unreleased
	s := testService()

	got := s.computeAggregateStatus([]SlotStatus{})
	if got != "unreleased" {
		t.Errorf("Empty slots: computeAggregateStatus() = %q, want %q", got, "unreleased")
	}
}

func TestComputeAggregateStatus_SingleSlot(t *testing.T) {
	s := testService()

	statuses := []string{"unreleased", "missing", "downloading", "failed", "upgradable", "available"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			slots := []SlotStatus{
				{SlotID: 1, Monitored: true, Status: status},
			}
			got := s.computeAggregateStatus(slots)
			expected := status
			// Special case: "unreleased" monitored slot still shows as unreleased (default fallback)
			// The function only checks for downloading/failed/missing/upgradable/available
			if status == "unreleased" {
				expected = "unreleased"
			}
			if got != expected {
				t.Errorf("Single slot %q: computeAggregateStatus() = %q, want %q", status, got, expected)
			}
		})
	}
}

func TestComputeAggregateStatus_MixedMonitoredUnmonitored(t *testing.T) {
	// Real-world scenario: 3 slots, one unmonitored
	s := testService()

	slots := []SlotStatus{
		{SlotID: 1, Monitored: true, Status: "available"},
		{SlotID: 2, Monitored: false, Status: "missing"},
		{SlotID: 3, Monitored: true, Status: "upgradable"},
	}
	got := s.computeAggregateStatus(slots)
	if got != "upgradable" {
		t.Errorf("Mixed monitored: computeAggregateStatus() = %q, want %q (upgradable > available)", got, "upgradable")
	}
}

func TestComputeAggregateStatus_DownloadingWinsOverFailed(t *testing.T) {
	// Spec confirms: downloading has highest priority after unreleased
	s := testService()

	slots := []SlotStatus{
		{SlotID: 1, Monitored: true, Status: "downloading"},
		{SlotID: 2, Monitored: true, Status: "failed"},
	}
	got := s.computeAggregateStatus(slots)
	if got != "downloading" {
		t.Errorf("downloading vs failed: computeAggregateStatus() = %q, want %q", got, "downloading")
	}
}

// Gap 10: Slot-level transition scenarios
func TestComputeAggregateStatus_SlotTransitions(t *testing.T) {
	s := testService()

	tests := []struct {
		name     string
		slots    []SlotStatus
		expected string
	}{
		{
			"One slot downloading, one available → downloading",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "downloading"},
				{SlotID: 2, Monitored: true, Status: "available"},
			},
			"downloading",
		},
		{
			"One slot failed, one upgradable → failed",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "failed"},
				{SlotID: 2, Monitored: true, Status: "upgradable"},
			},
			"failed",
		},
		{
			"Single slot missing → downloading → available (final state: available)",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
			},
			"available",
		},
		{
			"All slots available, one unmonitored missing → available",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
				{SlotID: 2, Monitored: true, Status: "available"},
				{SlotID: 3, Monitored: false, Status: "missing"},
			},
			"available",
		},
		{
			"Upgradable and missing → missing wins",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "upgradable"},
				{SlotID: 2, Monitored: true, Status: "missing"},
			},
			"missing",
		},
		{
			"Downloading, missing, available → downloading wins",
			[]SlotStatus{
				{SlotID: 1, Monitored: true, Status: "downloading"},
				{SlotID: 2, Monitored: true, Status: "missing"},
				{SlotID: 3, Monitored: true, Status: "available"},
			},
			"downloading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.computeAggregateStatus(tt.slots)
			if got != tt.expected {
				t.Errorf("computeAggregateStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}
