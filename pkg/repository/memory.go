package repository

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

var (
	ErrSlotNotFound    = errors.New("parking slot not found")
	ErrSessionNotFound = errors.New("parking session not found")
)

// ParkingRepository defines operations for managing system state.
type ParkingRepository interface {
	SaveToFile(filename string) error
	LoadFromFile(filename string) error

	AddGate(gate *domain.Gate) error
	GetGates() ([]*domain.Gate, error)
	GetGate(id int) (*domain.Gate, error)
	RemoveGate(id int) error

	AddSlot(slot *domain.ParkingSlot) error
	GetSlots() ([]*domain.ParkingSlot, error)
	GetSlot(id int) (*domain.ParkingSlot, error)
	UpdateSlot(slot *domain.ParkingSlot) error

	SaveSession(session *domain.ParkingSession) error
	GetSessions() ([]*domain.ParkingSession, error)
	GetActiveSessionByVehicle(licensePlate string) (*domain.ParkingSession, error)
	GetLastSessionByVehicle(licensePlate string) (*domain.ParkingSession, error)
	GetSession(id string) (*domain.ParkingSession, error)
}

// MemoryRepository is an in-memory, thread-safe implementation of ParkingRepository
type MemoryRepository struct {
	mu       sync.RWMutex
	gates    map[int]*domain.Gate
	slots    map[int]*domain.ParkingSlot
	sessions map[string]*domain.ParkingSession
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		gates:    make(map[int]*domain.Gate),
		slots:    make(map[int]*domain.ParkingSlot),
		sessions: make(map[string]*domain.ParkingSession),
	}
}

func (r *MemoryRepository) AddGate(gate *domain.Gate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gates[gate.ID] = gate
	return nil
}

func (r *MemoryRepository) GetGates() ([]*domain.Gate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	gates := make([]*domain.Gate, 0, len(r.gates))
	for _, g := range r.gates {
		gates = append(gates, g)
	}
	return gates, nil
}

func (r *MemoryRepository) GetGate(id int) (*domain.Gate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gates[id]
	if !ok {
		return nil, domain.ErrGateNotFound
	}
	return g, nil
}

func (r *MemoryRepository) RemoveGate(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.gates[id]; !ok {
		return domain.ErrGateNotFound
	}
	delete(r.gates, id)
	for _, slot := range r.slots {
		delete(slot.Distances, id)
	}
	return nil
}

func (r *MemoryRepository) AddSlot(slot *domain.ParkingSlot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slots[slot.ID] = slot
	return nil
}

func (r *MemoryRepository) GetSlots() ([]*domain.ParkingSlot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	slots := make([]*domain.ParkingSlot, 0, len(r.slots))
	for _, s := range r.slots {
		slots = append(slots, s)
	}
	return slots, nil
}

func (r *MemoryRepository) GetSlot(id int) (*domain.ParkingSlot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.slots[id]
	if !ok {
		return nil, ErrSlotNotFound
	}
	return s, nil
}

func (r *MemoryRepository) UpdateSlot(slot *domain.ParkingSlot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slots[slot.ID] = slot
	return nil
}

func (r *MemoryRepository) SaveSession(session *domain.ParkingSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *MemoryRepository) GetSessions() ([]*domain.ParkingSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sessions := make([]*domain.ParkingSession, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (r *MemoryRepository) GetActiveSessionByVehicle(licensePlate string) (*domain.ParkingSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		if s.VehicleID == licensePlate && s.IsActive {
			return s, nil
		}
	}
	return nil, ErrSessionNotFound
}

func (r *MemoryRepository) GetLastSessionByVehicle(licensePlate string) (*domain.ParkingSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var last *domain.ParkingSession
	for _, s := range r.sessions {
		if s.VehicleID == licensePlate && !s.IsActive {
			if last == nil || s.EntryTime.After(last.EntryTime) {
				last = s
			}
		}
	}
	if last == nil {
		return nil, ErrSessionNotFound
	}
	return last, nil
}

type StateData struct {
	Gates    map[int]*domain.Gate              `json:"gates"`
	Slots    map[int]*domain.ParkingSlot       `json:"slots"`
	Sessions map[string]*domain.ParkingSession `json:"sessions"`
}

func (r *MemoryRepository) SaveToFile(filename string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data := StateData{
		Gates:    r.gates,
		Slots:    r.slots,
		Sessions: r.sessions,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, bytes, 0644)
}

func (r *MemoryRepository) LoadFromFile(filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to load
		}
		return err
	}

	var data StateData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if data.Gates != nil {
		r.gates = data.Gates
	}
	if data.Slots != nil {
		r.slots = data.Slots
	}
	if data.Sessions != nil {
		r.sessions = data.Sessions
	}

	// Reconstruct PreviousSession links
	// Group sessions by VehicleID
	vehicleSessions := make(map[string][]*domain.ParkingSession)
	for _, s := range r.sessions {
		vehicleSessions[s.VehicleID] = append(vehicleSessions[s.VehicleID], s)
	}

	// Sort and relink
	for _, sessions := range vehicleSessions {
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].EntryTime.Before(sessions[j].EntryTime)
		})
		for i := 1; i < len(sessions); i++ {
			sessions[i].PreviousSession = sessions[i-1]
		}
	}

	return nil
}

func (r *MemoryRepository) GetSession(id string) (*domain.ParkingSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}
