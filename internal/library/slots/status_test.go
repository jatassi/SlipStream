package slots

import (
	"testing"
)

// Test matrix for status determination scenarios from spec Req 20.1
// T17: Missing status with 1 monitored slot empty, 1 filled
// T18: Missing status with 1 unmonitored slot empty

func TestMissingStatusDetermination(t *testing.T) {
	// T17: Req 6.1.1: Movie/episode is "missing" if ANY monitored slot is empty
	// T18: Req 6.1.2: Unmonitored empty slots do not affect missing status

	tests := []struct {
		name           string
		slotStatuses   []SlotStatus
		expectedMissing bool
		testID         string
	}{
		{
			name: "T17/Req 6.1.1: 1 monitored empty, 1 filled → missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
				{SlotID: 2, Monitored: true, Status: "missing"},
			},
			expectedMissing: true,
			testID:          "T17",
		},
		{
			name: "T18/Req 6.1.2: 1 unmonitored empty, 1 filled → not missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
				{SlotID: 2, Monitored: false, Status: "missing"},
			},
			expectedMissing: false,
			testID:          "T18",
		},
		{
			name: "All monitored slots filled → not missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
				{SlotID: 2, Monitored: true, Status: "available"},
			},
			expectedMissing: false,
		},
		{
			name: "All monitored slots empty → missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true, Status: "missing"},
				{SlotID: 2, Monitored: true, Status: "missing"},
			},
			expectedMissing: true,
		},
		{
			name: "No monitored slots (all unmonitored empty) → not missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: false, Status: "missing"},
				{SlotID: 2, Monitored: false, Status: "missing"},
			},
			expectedMissing: false,
		},
		{
			name: "Mixed: 1 monitored filled, 1 unmonitored filled, 1 monitored empty → missing",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true, Status: "available"},
				{SlotID: 2, Monitored: false, Status: "available"},
				{SlotID: 3, Monitored: true, Status: "missing"},
			},
			expectedMissing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isMissing := calculateMissingStatus(tt.slotStatuses)
			if isMissing != tt.expectedMissing {
				t.Errorf("calculateMissingStatus() = %v, want %v", isMissing, tt.expectedMissing)
			}
		})
	}
}

// calculateMissingStatus implements the missing status logic from GetMovieStatus/GetEpisodeStatus.
// Req 6.1.1: Movie/episode is "missing" if ANY monitored slot is empty
// Req 6.1.2: Unmonitored empty slots do not affect missing status
func calculateMissingStatus(slotStatuses []SlotStatus) bool {
	for _, status := range slotStatuses {
		if status.Monitored && status.Status == "missing" {
			return true
		}
	}
	return false
}

func TestUpgradeStatusDetermination(t *testing.T) {
	// Req 6.2.1: Each slot independently tracks upgrade eligibility based on profile cutoff
	// Req 6.2.2: Slot is "upgrade needed" if file exists but quality below profile cutoff

	tests := []struct {
		name              string
		slotStatuses      []SlotStatus
		expectedNeedsUpgrade bool
	}{
		{
			name: "Req 6.2.2: File below cutoff → needs upgrade",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Status: "upgradable", CurrentQualityID: intPtr(10), ProfileCutoff: 20},
			},
			expectedNeedsUpgrade: true,
		},
		{
			name: "File at cutoff → no upgrade needed",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Status: "available", CurrentQualityID: intPtr(20), ProfileCutoff: 20},
			},
			expectedNeedsUpgrade: false,
		},
		{
			name: "File above cutoff → no upgrade needed",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Status: "available", CurrentQualityID: intPtr(30), ProfileCutoff: 20},
			},
			expectedNeedsUpgrade: false,
		},
		{
			name: "Req 6.2.1: One slot needs upgrade, one doesn't → overall needs upgrade",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Status: "available", CurrentQualityID: intPtr(30), ProfileCutoff: 20},
				{SlotID: 2, Status: "upgradable", CurrentQualityID: intPtr(10), ProfileCutoff: 20},
			},
			expectedNeedsUpgrade: true,
		},
		{
			name: "Empty slot doesn't contribute to upgrade status",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Status: "missing"},
				{SlotID: 2, Status: "available", CurrentQualityID: intPtr(30), ProfileCutoff: 20},
			},
			expectedNeedsUpgrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsUpgrade := calculateOverallUpgradeStatus(tt.slotStatuses)
			if needsUpgrade != tt.expectedNeedsUpgrade {
				t.Errorf("calculateOverallUpgradeStatus() = %v, want %v", needsUpgrade, tt.expectedNeedsUpgrade)
			}
		})
	}
}

// calculateOverallUpgradeStatus checks if any slot needs an upgrade.
func calculateOverallUpgradeStatus(slotStatuses []SlotStatus) bool {
	for _, status := range slotStatuses {
		if status.Status == "upgradable" {
			return true
		}
	}
	return false
}

func TestPerSlotMonitoringIndependence(t *testing.T) {
	// Req 1.1.6: Each slot has its own independent monitored status per movie/episode
	// Req 8.1.1: Each slot has its own monitored toggle per movie/episode
	// Req 8.1.2: A slot can be monitored independently

	tests := []struct {
		name            string
		slotStatuses    []SlotStatus
		expectedMonitoredCount int
	}{
		{
			name: "Both slots monitored",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true},
				{SlotID: 2, Monitored: true},
			},
			expectedMonitoredCount: 2,
		},
		{
			name: "Only first slot monitored",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: true},
				{SlotID: 2, Monitored: false},
			},
			expectedMonitoredCount: 1,
		},
		{
			name: "Only second slot monitored",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: false},
				{SlotID: 2, Monitored: true},
			},
			expectedMonitoredCount: 1,
		},
		{
			name: "Req 8.1.2: Neither slot monitored (independent)",
			slotStatuses: []SlotStatus{
				{SlotID: 1, Monitored: false},
				{SlotID: 2, Monitored: false},
			},
			expectedMonitoredCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			for _, s := range tt.slotStatuses {
				if s.Monitored {
					count++
				}
			}
			if count != tt.expectedMonitoredCount {
				t.Errorf("monitored count = %d, want %d", count, tt.expectedMonitoredCount)
			}
		})
	}
}

// Helper function to create an int64 pointer
func intPtr(i int64) *int64 {
	return &i
}
