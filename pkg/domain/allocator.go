package domain

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrNoAvailableSlot = errors.New("no available parking slot for this vehicle")
	ErrInvalidGate     = errors.New("invalid or nonexistent gate ID")
)

type slotCandidate struct {
	slot     *ParkingSlot
	distance int
}

// FindOptimalSlot searches for the closest available and compatible parking slot relative to a gate.
func FindOptimalSlot(slots []*ParkingSlot, vehicle Vehicle, gateID int) (*ParkingSlot, error) {
	candidates := make([]slotCandidate, 0, len(slots))
	validGateFound := false

	for _, slot := range slots {
		dist, err := slot.DistanceFrom(gateID)

		if err == nil {
			validGateFound = true
		} else {
			continue
		}

		if slot.IsOccupied {
			continue
		}
		if !vehicle.Size.CanFit(slot.Size) {
			continue
		}
		candidates = append(candidates, slotCandidate{
			slot:     slot,
			distance: dist,
		})
	}

	if !validGateFound {
		return nil, fmt.Errorf("%w: gate ID %d", ErrInvalidGate, gateID)
	}
	if len(candidates) == 0 {
		return nil, ErrNoAvailableSlot
	}

	// Slot candidate criteria: closest to gate, smallest compatible size, then lowest slot ID.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		if candidates[i].slot.Size != candidates[j].slot.Size {
			return candidates[i].slot.Size < candidates[j].slot.Size
		}
		return candidates[i].slot.ID < candidates[j].slot.ID
	})

	return candidates[0].slot, nil
}
