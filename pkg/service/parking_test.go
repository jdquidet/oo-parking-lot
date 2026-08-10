package service

import (
	"errors"
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
		if !errors.Is(err, ErrVehicleAlreadyParked) {
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

func TestParkingService_FindAvailableSlot(t *testing.T) {
	s, repo := setupTestService()

	slot, err := s.FindAvailableSlot(domain.SizeSmall, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slot.ID != 102 {
		t.Errorf("expected slot 102, got %d", slot.ID)
	}
	if slot.IsOccupied {
		t.Errorf("expected availability lookup not to occupy the slot")
	}

	if _, err := s.FindAvailableSlot(domain.SizeSmall, 99); !errors.Is(err, domain.ErrGateNotFound) {
		t.Errorf("expected ErrGateNotFound for invalid gate, got %v", err)
	}

	largeSlot, _ := repo.GetSlot(102)
	largeSlot.IsOccupied = true
	_ = repo.UpdateSlot(largeSlot)
	if _, err := s.FindAvailableSlot(domain.SizeLarge, 1); !errors.Is(err, domain.ErrNoAvailableSlot) {
		t.Errorf("expected ErrNoAvailableSlot, got %v", err)
	}
}

func TestParkingService_RejectsInvalidLicensePlates(t *testing.T) {
	s, repo := setupTestService()

	if _, _, err := s.Park(domain.Vehicle{LicensePlate: "ABC--123", Size: domain.SizeSmall}, 1, baseTime); !errors.Is(err, domain.ErrInvalidLicensePlate) {
		t.Errorf("expected Park to return ErrInvalidLicensePlate, got %v", err)
	}
	if _, _, err := s.Unpark("   ", baseTime); !errors.Is(err, domain.ErrInvalidLicensePlate) {
		t.Errorf("expected Unpark to return ErrInvalidLicensePlate, got %v", err)
	}

	sessions, err := repo.GetSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected invalid plate not to create a session, got %d", len(sessions))
	}
}

func TestParkingService_ValidatesLicensePlateForDuplicateAndLookup(t *testing.T) {
	s, _ := setupTestService()
	vehicle := domain.Vehicle{LicensePlate: "  abc-123  ", Size: domain.SizeSmall}

	session, slot, err := s.Park(vehicle, 1, baseTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.VehicleID != "ABC-123" {
		t.Errorf("expected valid session vehicle ID, got %q", session.VehicleID)
	}
	if slot.CurrentVehicleID != "ABC-123" {
		t.Errorf("expected valid slot vehicle ID, got %q", slot.CurrentVehicleID)
	}

	if _, _, err := s.Park(domain.Vehicle{LicensePlate: "AbC-123", Size: domain.SizeSmall}, 1, baseTime.Add(time.Minute)); !errors.Is(err, ErrVehicleAlreadyParked) {
		t.Errorf("expected valid duplicate to return ErrVehicleAlreadyParked, got %v", err)
	}

	unparked, _, err := s.Unpark(" abc-123 ", baseTime.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unparked.VehicleID != "ABC-123" || unparked.IsActive {
		t.Errorf("expected valid lookup to unpark ABC-123, got %+v", unparked)
	}
}

func TestParkingService_ContinuedSessions(t *testing.T) {
	car := domain.Vehicle{LicensePlate: "CHN-SES", Size: domain.SizeSmall}

	t.Run("2-leg chain: second session within 1h gap", func(t *testing.T) {
		s, _ := setupTestService()

		// Leg 1: park at t=0, unpark at t=2h → flat rate 40
		_, _, err := s.Park(car, 1, baseTime)
		if err != nil {
			t.Fatalf("unexpected error on first park: %v", err)
		}
		session1, fee1, err := s.Unpark(car.LicensePlate, baseTime.Add(2*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error on first unpark: %v", err)
		}
		if fee1 != 40.0 {
			t.Errorf("expected leg 1 fee 40.0, got %.2f", fee1)
		}
		if session1.IsActive {
			t.Errorf("expected session 1 to be inactive")
		}

		// Leg 2: re-park at t=2h15m (gap < 1h), unpark at t=4h15m
		// Flat rate covers t=0 to t=3h; billable from t=3h to t=4h15m = 2h at LP rate 100 = 200
		session2, _, err := s.Park(car, 1, baseTime.Add(2*time.Hour+15*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on second park: %v", err)
		}
		if session2.PreviousSession == nil {
			t.Errorf("expected session 2 to have PreviousSession")
		}
		_, fee2, err := s.Unpark(car.LicensePlate, baseTime.Add(4*time.Hour+15*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on second unpark: %v", err)
		}
		if fee2 != 200.0 {
			t.Errorf("expected leg 2 fee 200.0, got %.2f", fee2)
		}
	})

	t.Run("3-leg chain: three sessions with gaps < 1h", func(t *testing.T) {
		s, _ := setupTestService()

		// Leg 1: park at t=0, unpark at t=2h → flat rate 40
		_, _, err := s.Park(car, 1, baseTime)
		if err != nil {
			t.Fatalf("unexpected error on first park: %v", err)
		}
		session1, fee1, err := s.Unpark(car.LicensePlate, baseTime.Add(2*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error on first unpark: %v", err)
		}
		if fee1 != 40.0 {
			t.Errorf("expected leg 1 fee 40.0, got %.2f", fee1)
		}
		if session1.IsActive {
			t.Errorf("expected session 1 to be inactive")
		}

		// Leg 2: re-park at t=2h15m, unpark at t=4h15m → 200 (2h billable at LP rate)
		session2, _, err := s.Park(car, 1, baseTime.Add(2*time.Hour+15*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on second park: %v", err)
		}
		if session2.PreviousSession == nil {
			t.Errorf("expected session 2 to have PreviousSession")
		}
		_, fee2, err := s.Unpark(car.LicensePlate, baseTime.Add(4*time.Hour+15*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on second unpark: %v", err)
		}
		if fee2 != 200.0 {
			t.Errorf("expected leg 2 fee 200.0, got %.2f", fee2)
		}

		// Leg 3: re-park at t=4h30m, unpark at t=6h30m
		// Root entry = t=0; flat rate covers t=0 to t=3h; billable from t=4h15m to t=6h30m = 2h15m → 3h at LP rate 100 = 300
		session3, _, err := s.Park(car, 1, baseTime.Add(4*time.Hour+30*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on third park: %v", err)
		}
		if session3.PreviousSession == nil {
			t.Errorf("expected session 3 to have PreviousSession")
		}
		_, fee3, err := s.Unpark(car.LicensePlate, baseTime.Add(6*time.Hour+30*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error on third unpark: %v", err)
		}
		if fee3 != 300.0 {
			t.Errorf("expected leg 3 fee 300.0, got %.2f", fee3)
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
