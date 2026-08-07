package domain

import (
	"fmt"
	"time"
)

// VehicleSize represents the physical classification of a vehicle.
type VehicleSize int

const (
	SizeSmall VehicleSize = iota
	SizeMedium
	SizeLarge
)

// String returns the string representation of the VehicleSize.
func (v VehicleSize) String() string {
	switch v {
	case SizeSmall:
		return "S"
	case SizeMedium:
		return "M"
	case SizeLarge:
		return "L"
	default:
		return "Unknown"
	}
}

// SlotSize represents the physical classification of a parking slot.
type SlotSize int

const (
	SlotSP SlotSize = iota // Small Parking
	SlotMP                 // Medium
	SlotLP                 // Large
)

// String returns the string representation of the SlotSize.
func (s SlotSize) String() string {
	switch s {
	case SlotSP:
		return "SP"
	case SlotMP:
		return "MP"
	case SlotLP:
		return "LP"
	default:
		return "Unknown"
	}
}

// CanFit checks whether a vehicle size can fit into a given parking slot size.
func (v VehicleSize) CanFit(s SlotSize) bool {
	if v < SizeSmall || v > SizeLarge {
		return false
	}
	return SlotSize(v) <= s
}

// Vehicle represents a vehicle entering or parked in the parking complex.
type Vehicle struct {
	LicensePlate string      `json:"license_plate"`
	Size         VehicleSize `json:"size"`
}

// Gate represents an access point (entry or exit) into the parking complex.
type Gate struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DistanceMap maps a Gate ID to its distance unit from a given location.
type DistanceMap map[int]int

// ParkingSlot represents an individual parking space and its current status.
type ParkingSlot struct {
	ID               int         `json:"id"`
	Size             SlotSize    `json:"size"`
	Distances        DistanceMap `json:"distances"`
	IsOccupied       bool        `json:"is_occupied"`
	CurrentVehicleID string      `json:"current_vehicle_id,omitempty"`
}

// DistanceFrom returns the distance unit from the parking slot to a given gate ID.
func (ps *ParkingSlot) DistanceFrom(gateID int) (int, error) {
	dist, ok := ps.Distances[gateID]
	if !ok {
		return 0, fmt.Errorf("no distance found for gate ID %d", gateID)
	}
	return dist, nil
}

// ParkingSession represents an active or historical parking record for a vehicle.
type ParkingSession struct {
	ID              string      `json:"id"`
	VehicleID       string      `json:"vehicle_id"`
	VehicleSize     VehicleSize `json:"vehicle_size"`
	SlotID          int         `json:"slot_id"`
	SlotSize        SlotSize    `json:"slot_size"`
	GateID          int         `json:"gate_id"`
	EntryTime       time.Time   `json:"entry_time"`
	ExitTime        *time.Time  `json:"exit_time,omitempty"`
	TotalFeeCharged float64     `json:"total_fee_charged"`
	IsActive        bool        `json:"is_active"`
}
