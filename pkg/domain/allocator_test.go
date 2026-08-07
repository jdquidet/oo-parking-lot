package domain

import (
	"testing"
)

func TestFindOptimalSlot(t *testing.T) {
	slots := []*ParkingSlot{
		{ID: 1, Size: SlotSP, Distances: DistanceMap{1: 5, 2: 2}, IsOccupied: false},
		{ID: 2, Size: SlotMP, Distances: DistanceMap{1: 2, 2: 4}, IsOccupied: false},
		{ID: 3, Size: SlotLP, Distances: DistanceMap{1: 2, 2: 1}, IsOccupied: false},
		{ID: 4, Size: SlotSP, Distances: DistanceMap{1: 1, 2: 5}, IsOccupied: true},
	}

	t.Run("Small vehicle gets closest fitting Slot 2 at Gate 1", func(t *testing.T) {
		smallCar := Vehicle{LicensePlate: "ABC-123", Size: SizeSmall}

		slot, err := FindOptimalSlot(slots, smallCar, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if slot.ID != 2 {
			t.Errorf("expected Slot 2, got Slot %d", slot.ID)
		}
	})

	t.Run("Large vehicle only fits LP slot 3", func(t *testing.T) {
		largeCar := Vehicle{LicensePlate: "XYZ-789", Size: SizeLarge}

		slot, err := FindOptimalSlot(slots, largeCar, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if slot.ID != 3 {
			t.Errorf("expected Slot 3, got Slot %d", slot.ID)
		}
	})

	t.Run("Returns error when no fitting slot is available", func(t *testing.T) {
		largeCar := Vehicle{LicensePlate: "XYZ-789", Size: SizeLarge}

		slots[2].IsOccupied = true
		defer func() { slots[2].IsOccupied = false }()

		_, err := FindOptimalSlot(slots, largeCar, 1)
		if err != ErrNoAvailableSlot {
			t.Errorf("expected ErrNoAvailableSlot, got %v", err)
		}
	})
}
