package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"github.com/jdquidet/oo-parking-lot/pkg/repository"
)

var (
	ErrVehicleAlreadyParked = errors.New("vehicle is already parked in the complex")
	ErrNoActiveSession      = errors.New("no active parking session found for vehicle")
)

type ParkingService interface {
	Park(vehicle domain.Vehicle, gateID int, entryTime time.Time) (*domain.ParkingSession, *domain.ParkingSlot, error)
	Unpark(licensePlate string, exitTime time.Time) (*domain.ParkingSession, float64, error)
}

type parkingService struct {
	repo repository.ParkingRepository
}

func NewParkingService(repo repository.ParkingRepository) ParkingService {
	return &parkingService{repo: repo}
}

// Park handles entry of a vehicle into the parking complex.
func (s *parkingService) Park(
	vehicle domain.Vehicle,
	gateID int,
	entryTime time.Time,
) (*domain.ParkingSession, *domain.ParkingSlot, error) {
	// Verify gate exists
	if _, err := s.repo.GetGate(gateID); err != nil {
		return nil, nil, err
	}
	// Prevent duplicate active sessions
	if existing, _ := s.repo.GetActiveSessionByVehicle(vehicle.LicensePlate); existing != nil {
		return nil, nil, ErrVehicleAlreadyParked
	}

	// Retrieve slots and find optimal slot
	slots, err := s.repo.GetSlots()
	if err != nil {
		return nil, nil, err
	}
	optimalSlot, err := domain.FindOptimalSlot(slots, vehicle, gateID)
	if err != nil {
		return nil, nil, err
	}

	// Occupy the slot, complete log, and save session
	optimalSlot.IsOccupied = true
	optimalSlot.CurrentVehicleID = vehicle.LicensePlate
	if err := s.repo.UpdateSlot(optimalSlot); err != nil {
		return nil, nil, err
	}
	sessionID := fmt.Sprintf("SESS-%s-%d", vehicle.LicensePlate, entryTime.UnixNano())
	session := &domain.ParkingSession{
		ID:          sessionID,
		VehicleID:   vehicle.LicensePlate,
		VehicleSize: vehicle.Size,
		SlotID:      optimalSlot.ID,
		SlotSize:    optimalSlot.Size,
		GateID:      gateID,
		EntryTime:   entryTime,
		IsActive:    true,
	}
	if err := s.repo.SaveSession(session); err != nil {
		return nil, nil, err
	}

	return session, optimalSlot, nil
}

func (s *parkingService) Unpark(
	licensePlate string,
	exitTime time.Time,
) (*domain.ParkingSession, float64, error) {
	activeSession, err := s.repo.GetActiveSessionByVehicle(licensePlate)
	if err != nil {
		return nil, 0.0, ErrNoActiveSession
	}
	if exitTime.Before(activeSession.EntryTime) {
		return nil, 0.0, errors.New("exit time can't be prior to entry time")
	}

	// Fetch last inactive session for continuous rate check
	var previousSession *domain.ParkingSession
	if last, err := s.repo.GetLastSessionByVehicle(licensePlate); err != nil {
		if !last.IsActive {
			previousSession = last
		}
	}

	fee, err := CalculateFee(activeSession.SlotSize, activeSession.EntryTime, exitTime, previousSession)
	if err != nil {
		return nil, 0.0, err
	}

	// Free the occupied slot, complete log, and save session
	slot, err := s.repo.GetSlot(activeSession.SlotID)
	if err == nil {
		slot.IsOccupied = false
		slot.CurrentVehicleID = ""
		_ = s.repo.UpdateSlot(slot)
	}
	activeSession.ExitTime = &exitTime
	activeSession.TotalFeeCharged = fee
	activeSession.IsActive = false
	if err := s.repo.SaveSession(activeSession); err != nil {
		return nil, 0.0, err
	}

	return activeSession, fee, nil
}
