package domain

import (
	"errors"
	"testing"
)

func TestFindOptimalSlot(t *testing.T) {
	setupTestSlots := func() []*ParkingSlot {
		return []*ParkingSlot{
			{ID: 1, Size: SlotSP, Distances: DistanceMap{1: 5, 2: 2}, IsOccupied: false},
			{ID: 2, Size: SlotMP, Distances: DistanceMap{1: 2, 2: 4}, IsOccupied: false},
			{ID: 3, Size: SlotLP, Distances: DistanceMap{1: 2, 2: 1}, IsOccupied: false},
			{ID: 4, Size: SlotSP, Distances: DistanceMap{1: 1, 2: 5}, IsOccupied: true}, // Occupied
		}
	}
	tests := []struct {
		name        string
		slots       []*ParkingSlot
		vehicle     Vehicle
		gateID      int
		expectedID  int
		expectedErr error
	}{
		{
			name:        "Large vehicle only fits LP Slot 3 at Gate 1",
			slots:       setupTestSlots(),
			vehicle:     Vehicle{LicensePlate: "LRG-000", Size: SizeLarge},
			gateID:      1,
			expectedID:  3,
			expectedErr: nil,
		},
		{
			name:        "Tiebreaker: small vehicle gets smaller MP Slot 2 at Gate 1",
			slots:       setupTestSlots(),
			vehicle:     Vehicle{LicensePlate: "TIE-001", Size: SizeSmall},
			gateID:      1,
			expectedID:  2,
			expectedErr: nil,
		},
		{
			name: "Exact distance and size tie selects lower slot ID with reversed input order",
			slots: []*ParkingSlot{
				{ID: 20, Size: SlotMP, Distances: DistanceMap{1: 2}},
				{ID: 10, Size: SlotMP, Distances: DistanceMap{1: 2}},
			},
			vehicle:     Vehicle{LicensePlate: "TIE-002", Size: SizeSmall},
			gateID:      1,
			expectedID:  10,
			expectedErr: nil,
		},
		{
			name: "Returns ErrNoAvailableSlot when no fitting slot is available",
			slots: func() []*ParkingSlot {
				s := setupTestSlots()
				s[2].IsOccupied = true // Occupy LP Slot 3
				return s
			}(),
			vehicle:     Vehicle{LicensePlate: "SLT-002", Size: SizeLarge},
			gateID:      1,
			expectedErr: ErrNoAvailableSlot,
		},
		{
			name:        "Returns ErrInvalidGate when gateID does not exist",
			slots:       setupTestSlots(),
			vehicle:     Vehicle{LicensePlate: "GAT-003", Size: SizeSmall},
			gateID:      99,
			expectedErr: ErrInvalidGate,
		},
		{
			name: "Skips slots with negative distance",
			slots: []*ParkingSlot{
				{ID: 1, Size: SlotSP, Distances: DistanceMap{1: -1}, IsOccupied: false}, // Invalid distance
				{ID: 2, Size: SlotSP, Distances: DistanceMap{1: 5}, IsOccupied: false},  // Valid
			},
			vehicle:     Vehicle{LicensePlate: "NEG-004", Size: SizeSmall},
			gateID:      1,
			expectedID:  2,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot, err := FindOptimalSlot(tt.slots, tt.vehicle, tt.gateID)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if slot.ID != tt.expectedID {
				t.Errorf("expected Slot ID %d, got %d", tt.expectedID, slot.ID)
			}
		})
	}
}
