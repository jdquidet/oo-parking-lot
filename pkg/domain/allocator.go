package domain

import (
	"errors"
	"sort"
)

var ErrNoAvailableSlot = errors.New("no available parking slot for this vehicle")

// FindOptimalSlot searches for the closest available and compatible parking slot relative to a gate.
func FindOptimalSlot(slots []*ParkingSlot, vehicle Vehicle, gateID int) (*ParkingSlot, error) {
	var candidates []*ParkingSlot

	for _, slot := range slots {
		if slot.IsOccupied {
			continue
		}
		if !vehicle.Size.CanFit(slot.Size) {
			continue
		}
		if _, err := slot.DistanceFrom(gateID); err != nil {
			continue
		}
		candidates = append(candidates, slot)
	}

	if len(candidates) == 0 {
		return nil, ErrNoAvailableSlot
	}

	sort.Slice(candidates, func(i, j int) bool {
		distI, _ := candidates[i].DistanceFrom(gateID)
		distJ, _ := candidates[j].DistanceFrom(gateID)

		if distI != distJ {
			return distI < distJ
		}
		return candidates[i].Size < candidates[j].Size
	})

	return candidates[0], nil
}
