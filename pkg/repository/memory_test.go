package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

var baseTime = time.Date(2026, time.August, 11, 17, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

func TestGetGate(t *testing.T) {
	r := NewMemoryRepository()
	_ = r.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})

	tests := []struct {
		name        string
		gateID      int
		expected    *domain.Gate
		expectedErr error
	}{
		{
			name:        "Returns gate when found",
			gateID:      1,
			expected:    &domain.Gate{ID: 1, Name: "Gate A"},
			expectedErr: nil,
		},
		{
			name:        "Returns ErrGateNotFound when gate does not exist",
			gateID:      99,
			expectedErr: domain.ErrGateNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate, err := r.GetGate(tt.gateID)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gate.ID != tt.expected.ID || gate.Name != tt.expected.Name {
				t.Errorf("expected gate %+v, got %+v", tt.expected, gate)
			}
		})
	}
}

func TestGetGates(t *testing.T) {
	r := NewMemoryRepository()
	_ = r.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
	_ = r.AddGate(&domain.Gate{ID: 2, Name: "Gate B"})

	gates, err := r.GetGates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gates) != 2 {
		t.Fatalf("expected 2 gates, got %d", len(gates))
	}

	got := make(map[int]string)
	for _, g := range gates {
		got[g.ID] = g.Name
	}
	if got[1] != "Gate A" || got[2] != "Gate B" {
		t.Errorf("expected gates A and B, got %v", got)
	}
}

func TestGetSlot(t *testing.T) {
	r := NewMemoryRepository()
	_ = r.AddSlot(&domain.ParkingSlot{
		ID:        101,
		Size:      domain.SlotSP,
		Distances: domain.DistanceMap{1: 2},
	})

	tests := []struct {
		name        string
		slotID      int
		expectedErr error
	}{
		{
			name:        "Returns slot when found",
			slotID:      101,
			expectedErr: nil,
		},
		{
			name:        "Returns ErrSlotNotFound when slot does not exist",
			slotID:      999,
			expectedErr: ErrSlotNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot, err := r.GetSlot(tt.slotID)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if slot.ID != 101 {
				t.Errorf("expected slot ID 101, got %d", slot.ID)
			}
		})
	}
}

func TestGetSlots(t *testing.T) {
	r := NewMemoryRepository()
	_ = r.AddSlot(&domain.ParkingSlot{ID: 101, Size: domain.SlotSP})
	_ = r.AddSlot(&domain.ParkingSlot{ID: 102, Size: domain.SlotLP})

	slots, err := r.GetSlots()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}

	got := make(map[int]bool)
	for _, s := range slots {
		got[s.ID] = true
	}
	if !got[101] || !got[102] {
		t.Errorf("expected slots 101 and 102, got %v", got)
	}
}

func TestUpdateSlot(t *testing.T) {
	r := NewMemoryRepository()
	_ = r.AddSlot(&domain.ParkingSlot{
		ID:        101,
		Size:      domain.SlotSP,
		Distances: domain.DistanceMap{1: 2},
	})

	slot, _ := r.GetSlot(101)
	if slot.IsOccupied {
		t.Fatalf("expected slot to be unoccupied initially")
	}

	slot.IsOccupied = true
	slot.CurrentVehicleID = "ABC-123"
	_ = r.UpdateSlot(slot)

	updated, err := r.GetSlot(101)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.IsOccupied || updated.CurrentVehicleID != "ABC-123" {
		t.Errorf("expected slot to be occupied by ABC-123, got %+v", updated)
	}
}

func TestGetSession(t *testing.T) {
	r := NewMemoryRepository()
	session := newTestSession("ABC-123", baseTime, true)
	_ = r.SaveSession(session)

	tests := []struct {
		name        string
		sessionID   string
		expectedErr error
	}{
		{
			name:        "Returns session when found",
			sessionID:   session.ID,
			expectedErr: nil,
		},
		{
			name:        "Returns ErrSessionNotFound when session does not exist",
			sessionID:   "SESS-NONE",
			expectedErr: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetSession(tt.sessionID)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.VehicleID != "ABC-123" {
				t.Errorf("expected vehicle ID ABC-123, got %s", got.VehicleID)
			}
		})
	}
}

func TestGetActiveSessionByVehicle(t *testing.T) {
	r := NewMemoryRepository()
	active := &domain.ParkingSession{
		ID:        "SESS-ACTIVE",
		VehicleID: "ABC-123",
		SlotID:    101,
		EntryTime: baseTime,
		IsActive:  true,
	}
	inactive := &domain.ParkingSession{
		ID:        "SESS-INACTIVE",
		VehicleID: "ABC-123",
		SlotID:    101,
		EntryTime: baseTime.Add(-24 * time.Hour),
		IsActive:  false,
		ExitTime:  timePtr(baseTime.Add(-22 * time.Hour)),
	}
	_ = r.SaveSession(active)
	_ = r.SaveSession(inactive)

	tests := []struct {
		name        string
		vehicle     string
		expectedID  string
		expectedErr error
	}{
		{
			name:        "Returns active session when vehicle has active + inactive",
			vehicle:     "ABC-123",
			expectedID:  active.ID,
			expectedErr: nil,
		},
		{
			name:        "Returns ErrSessionNotFound when no active session exists",
			vehicle:     "XYZ-999",
			expectedErr: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetActiveSessionByVehicle(tt.vehicle)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.expectedID {
				t.Errorf("expected session ID %s, got %s", tt.expectedID, got.ID)
			}
		})
	}
}

func TestGetLastSessionByVehicle(t *testing.T) {
	r := NewMemoryRepository()
	s1 := &domain.ParkingSession{
		ID:        "SESS-1",
		VehicleID: "ABC-123",
		SlotID:    101,
		EntryTime: baseTime,
		IsActive:  false,
		ExitTime:  timePtr(baseTime.Add(2 * time.Hour)),
	}
	s2 := &domain.ParkingSession{
		ID:        "SESS-2",
		VehicleID: "ABC-123",
		SlotID:    101,
		EntryTime: baseTime.Add(24 * time.Hour),
		IsActive:  false,
		ExitTime:  timePtr(baseTime.Add(26 * time.Hour)),
	}
	active := &domain.ParkingSession{
		ID:        "SESS-3",
		VehicleID: "ABC-123",
		SlotID:    101,
		EntryTime: baseTime.Add(48 * time.Hour),
		IsActive:  true,
	}
	_ = r.SaveSession(s1)
	_ = r.SaveSession(s2)
	_ = r.SaveSession(active)

	tests := []struct {
		name        string
		vehicle     string
		expectedID  string
		expectedErr error
	}{
		{
			name:        "Returns most recent inactive session",
			vehicle:     "ABC-123",
			expectedID:  s2.ID,
			expectedErr: nil,
		},
		{
			name:        "Returns ErrSessionNotFound when no inactive sessions exist",
			vehicle:     "XYZ-999",
			expectedErr: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetLastSessionByVehicle(tt.vehicle)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.expectedID {
				t.Errorf("expected session ID %s, got %s", tt.expectedID, got.ID)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func newTestSession(vehicle string, entry time.Time, active bool) *domain.ParkingSession {
	return &domain.ParkingSession{
		ID:              "SESS-" + vehicle,
		VehicleID:       vehicle,
		SlotID:          101,
		EntryTime:       entry,
		IsActive:        active,
		ExitTime:        nil,
		TotalFeeCharged: 0,
	}
}
