package repository

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

var baseTime = time.Date(2026, time.August, 11, 17, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

func TestSaveAndLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")

	// 1. Create a repository and populate it with data
	r1 := NewMemoryRepository()
	_ = r1.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
	_ = r1.AddSlot(&domain.ParkingSlot{
		ID:               101,
		Size:             domain.SlotSP,
		Distances:        domain.DistanceMap{1: 2},
		IsOccupied:       true,
		CurrentVehicleID: "ABC-123",
	})

	// Add chained sessions to test PreviousSession relinking
	sess1 := &domain.ParkingSession{
		ID:        "SESS-1",
		VehicleID: "ABC-123",
		EntryTime: baseTime,
		ExitTime:  timePtr(baseTime.Add(1 * time.Hour)),
		IsActive:  false,
	}
	sess2 := &domain.ParkingSession{
		ID:              "SESS-2",
		VehicleID:       "ABC-123",
		EntryTime:       baseTime.Add(2 * time.Hour),
		IsActive:        true,
		PreviousSession: sess1,
	}
	_ = r1.SaveSession(sess1)
	_ = r1.SaveSession(sess2)

	// 2. Save state to file
	err := r1.SaveToFile(stateFile)
	if err != nil {
		t.Fatalf("failed to save to file: %v", err)
	}

	// 3. Create a new repository and load state from the file
	r2 := NewMemoryRepository()
	err = r2.LoadFromFile(stateFile)
	if err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	// 4. Verify data was loaded correctly
	gates, _ := r2.GetGates()
	if len(gates) != 1 || gates[0].Name != "Gate A" {
		t.Errorf("expected 1 gate 'Gate A', got %v", gates)
	}

	slot, _ := r2.GetSlot(101)
	if slot == nil || !slot.IsOccupied || slot.CurrentVehicleID != "ABC-123" {
		t.Errorf("expected slot 101 to be occupied by 'ABC-123', got %+v", slot)
	}

	// Check if sessions were loaded and PreviousSession was correctly relinked
	loadedSess2, err := r2.GetSession("SESS-2")
	if err != nil {
		t.Fatalf("failed to get SESS-2: %v", err)
	}

	if loadedSess2.PreviousSession == nil {
		t.Fatal("expected SESS-2 to have a PreviousSession linked")
	}

	if loadedSess2.PreviousSession.ID != "SESS-1" {
		t.Errorf("expected PreviousSession ID to be 'SESS-1', got '%s'", loadedSess2.PreviousSession.ID)
	}
}

func TestLoadFromFile_NotExists(t *testing.T) {
	r := NewMemoryRepository()

	// Should return nil (no error) when file does not exist
	err := r.LoadFromFile("non_existent_state.json")
	if err != nil {
		t.Fatalf("expected nil when file doesn't exist, got error: %v", err)
	}
}

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

func TestRemoveGate(t *testing.T) {
	t.Run("removes gate and its distance mapping from every slot", func(t *testing.T) {
		r := NewMemoryRepository()
		_ = r.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
		_ = r.AddGate(&domain.Gate{ID: 2, Name: "Gate B"})
		_ = r.AddSlot(&domain.ParkingSlot{ID: 101, Distances: domain.DistanceMap{1: 2, 2: 5}})
		_ = r.AddSlot(&domain.ParkingSlot{ID: 102, Distances: domain.DistanceMap{1: 4, 2: 3}})

		if err := r.RemoveGate(1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := r.GetGate(1); !errors.Is(err, domain.ErrGateNotFound) {
			t.Fatalf("expected removed gate to return ErrGateNotFound, got %v", err)
		}
		for _, slotID := range []int{101, 102} {
			slot, err := r.GetSlot(slotID)
			if err != nil {
				t.Fatalf("unexpected error getting slot %d: %v", slotID, err)
			}
			if _, ok := slot.Distances[1]; ok {
				t.Errorf("expected gate 1 distance to be removed from slot %d", slotID)
			}
			if _, ok := slot.Distances[2]; !ok {
				t.Errorf("expected gate 2 distance to remain on slot %d", slotID)
			}
		}
	})

	t.Run("returns ErrGateNotFound without changing distances when gate is absent", func(t *testing.T) {
		r := NewMemoryRepository()
		_ = r.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
		_ = r.AddSlot(&domain.ParkingSlot{ID: 101, Distances: domain.DistanceMap{1: 2, 99: 8}})

		if err := r.RemoveGate(99); !errors.Is(err, domain.ErrGateNotFound) {
			t.Fatalf("expected ErrGateNotFound, got %v", err)
		}
		slot, err := r.GetSlot(101)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if slot.Distances[99] != 8 {
			t.Errorf("expected absent gate removal to leave distances unchanged")
		}
	})
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

func TestGetSessions(t *testing.T) {
	r := NewMemoryRepository()
	first := newTestSession("ABC-123", baseTime, true)
	second := newTestSession("XYZ-999", baseTime.Add(time.Hour), false)
	_ = r.SaveSession(first)
	_ = r.SaveSession(second)

	sessions, err := r.GetSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	got := make(map[string]bool)
	for _, session := range sessions {
		got[session.ID] = true
	}
	if !got[first.ID] || !got[second.ID] {
		t.Errorf("expected sessions %q and %q, got %v", first.ID, second.ID, got)
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
