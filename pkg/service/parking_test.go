package service

import (
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"github.com/jdquidet/oo-parking-lot/pkg/repository"
)

func TestParkingService_ParkAndUnpark(t *testing.T) {
	s, _ := setupTestService()
	car := domain.Vehicle{LicensePlate: "SML-C4R", Size: domain.SizeSmall}

	t.Run("Successfully park vehicle", func(t *testing.T) {
		session, slot, err := s.Park(car, 1, baseTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if slot.ID != 102 { // Parking slot 102 is the most optimal
			t.Errorf("expected slot 102, got %d", slot.ID)
		}
		if !session.IsActive {
			t.Errorf("expected session to be active")
		}
	})

	t.Run("Cannot park vehicle if already parked", func(t *testing.T) {
		_, _, err := s.Park(car, 1, baseTime.Add(5*time.Minute))
		if err != ErrVehicleAlreadyParked {
			t.Errorf("expected ErrVehicleAlreadyParked, got %v", err)
		}
	})

	t.Run("Successfully unpark vehicle and compute fee", func(t *testing.T) {
		session, fee, err := s.Unpark(car.LicensePlate, baseTime.Add(2*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fee != 40.0 { // Parked for 2 hours
			t.Errorf("expected flat rate 40.0, got %.2f", fee)
		}
		if session.IsActive {
			t.Errorf("expected session to be inactive after unpark")
		}
	})
}

func setupTestService() (ParkingService, repository.ParkingRepository) {
	repo := repository.NewMemoryRepository()

	_ = repo.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
	_ = repo.AddGate(&domain.Gate{ID: 2, Name: "Gate B"})

	_ = repo.AddSlot(&domain.ParkingSlot{
		ID:        101,
		Size:      domain.SlotSP,
		Distances: domain.DistanceMap{1: 2, 2: 5},
	})
	_ = repo.AddSlot(&domain.ParkingSlot{
		ID:        102,
		Size:      domain.SlotLP,
		Distances: domain.DistanceMap{1: 1, 2: 3},
	})

	return NewParkingService(repo), repo
}
