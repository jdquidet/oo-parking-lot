package domain

import (
	"errors"
	"testing"
)

func TestDistanceFrom(t *testing.T) {
	slot := ParkingSlot{
		ID: 1,
		Distances: DistanceMap{
			1: 10,
			2: -5,
		},
	}

	tests := []struct {
		name        string
		gateID      int
		expected    int
		expectedErr error
	}{
		{
			name:        "Valid distance",
			gateID:      1,
			expected:    10,
			expectedErr: nil,
		},
		{
			name:        "Missing Gate ID",
			gateID:      99,
			expectedErr: ErrGateNotFound,
		},
		{
			name:        "Returns ErrInvalidDistance when distance is negative",
			gateID:      2,
			expectedErr: ErrInvalidDistance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist, err := slot.DistanceFrom(tt.gateID)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dist != tt.expected {
				t.Errorf("expected distance %d, got %d", tt.expected, dist)
			}
		})
	}
}
