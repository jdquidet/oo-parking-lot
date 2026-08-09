package repository

import (
	"errors"
	"sync"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

var (
	ErrSlotNotFound    = errors.New("parking slot not found")
	ErrGateNotFound    = errors.New("gate not found")
	ErrSessionNotFound = errors.New("parking session not found")
)

// ParkingRepository defines operations for managing system state.
type ParkingRepository interface {
	AddGate(gate *domain.Gate) error
	GetGates() ([]*domain.Gate, error)
	GetGate(id int) (*domain.Gate, error)

	AddSlot(slot *domain.ParkingSlot) error
	GetSlots() ([]*domain.ParkingSlot, error)
	GetSlot(id int) (*domain.ParkingSlot, error)
	UpdateSlot(slot *domain.ParkingSlot) error

	SaveSession(session *domain.ParkingSession) error
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
		return nil, ErrGateNotFound
	}
	return g, nil
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
		if s.VehicleID == licensePlate {
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

func (r *MemoryRepository) GetSession(id string) (*domain.ParkingSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}
