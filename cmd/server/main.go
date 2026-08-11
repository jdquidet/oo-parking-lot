package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"github.com/jdquidet/oo-parking-lot/pkg/repository"
	"github.com/jdquidet/oo-parking-lot/pkg/service"
)

type ServerApp struct {
	repo       repository.ParkingRepository
	service    service.ParkingService
	mu         sync.RWMutex
	systemTime time.Time
}

func main() {
	repo := repository.NewMemoryRepository()

	// Attempt to load previous state
	_ = repo.LoadFromFile("state.json")

	svc := service.NewParkingService(repo)
	app := &ServerApp{
		repo:       repo,
		service:    svc,
		systemTime: time.Now().Truncate(time.Minute),
	}

	// If no gates were loaded, initialize default map
	gates, _ := repo.GetGates()
	if len(gates) == 0 {
		app.seedDefaultMap()
		_ = app.repo.SaveToFile("state.json")
	}

	mux := http.NewServeMux()

	// Endpoints (using Go 1.22+ routing syntax)
	mux.HandleFunc("GET /api/state", app.handleGetState)
	mux.HandleFunc("POST /api/park", app.handlePark)
	mux.HandleFunc("POST /api/unpark", app.handleUnpark)
	mux.HandleFunc("GET /api/sessions", app.handleGetSessions)
	mux.HandleFunc("POST /api/gates", app.handleAddGate)
	mux.HandleFunc("DELETE /api/gates/{id}", app.handleRemoveGate)
	mux.HandleFunc("POST /api/time/advance", app.handleAdvanceTime)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (app *ServerApp) seedDefaultMap() {
	_ = app.repo.AddGate(&domain.Gate{ID: 1, Name: "Gate A"})
	_ = app.repo.AddGate(&domain.Gate{ID: 2, Name: "Gate B"})
	_ = app.repo.AddGate(&domain.Gate{ID: 3, Name: "Gate C"})

	slots := []*domain.ParkingSlot{
		{ID: 101, Size: domain.SlotSP, Distances: domain.DistanceMap{1: 1, 2: 4, 3: 4}},
		{ID: 102, Size: domain.SlotMP, Distances: domain.DistanceMap{1: 2, 2: 3, 3: 3}},
		{ID: 103, Size: domain.SlotLP, Distances: domain.DistanceMap{1: 3, 2: 2, 3: 4}},
		{ID: 201, Size: domain.SlotMP, Distances: domain.DistanceMap{1: 2, 2: 3, 3: 3}},
		{ID: 202, Size: domain.SlotLP, Distances: domain.DistanceMap{1: 3, 2: 2, 3: 2}},
		{ID: 203, Size: domain.SlotSP, Distances: domain.DistanceMap{1: 4, 2: 1, 3: 3}},
		{ID: 301, Size: domain.SlotLP, Distances: domain.DistanceMap{1: 3, 2: 4, 3: 2}},
		{ID: 302, Size: domain.SlotSP, Distances: domain.DistanceMap{1: 4, 2: 3, 3: 1}},
		{ID: 303, Size: domain.SlotMP, Distances: domain.DistanceMap{1: 5, 2: 2, 3: 2}},
	}

	for _, s := range slots {
		_ = app.repo.AddSlot(s)
	}
}

// Helpers
func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, msg string) {
	sendJSON(w, status, map[string]string{"error": msg})
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Handlers ---

func (app *ServerApp) handleGetState(w http.ResponseWriter, r *http.Request) {
	gates, _ := app.repo.GetGates()
	slots, _ := app.repo.GetSlots()

	app.mu.RLock()
	currentTime := app.systemTime
	app.mu.RUnlock()

	sendJSON(w, http.StatusOK, map[string]any{
		"gates":      gates,
		"slots":      slots,
		"systemTime": currentTime.Format(time.RFC3339),
	})
}

func (app *ServerApp) handlePark(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GateID       int    `json:"gate_id"`
		LicensePlate string `json:"license_plate"`
		VehicleSize  int    `json:"vehicle_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app.mu.RLock()
	currTime := app.systemTime
	app.mu.RUnlock()

	vehicle := domain.Vehicle{
		LicensePlate: req.LicensePlate,
		Size:         domain.VehicleSize(req.VehicleSize),
	}

	session, slot, err := app.service.Park(vehicle, req.GateID, currTime)
	if err != nil {
		sendError(w, http.StatusConflict, err.Error())
		return
	}

	_ = app.repo.SaveToFile("state.json")

	sendJSON(w, http.StatusOK, map[string]any{
		"session": session,
		"slot":    slot,
	})
}

func (app *ServerApp) handleUnpark(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LicensePlate string `json:"license_plate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app.mu.RLock()
	currTime := app.systemTime
	app.mu.RUnlock()

	session, fee, err := app.service.Unpark(req.LicensePlate, currTime)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = app.repo.SaveToFile("state.json")

	sendJSON(w, http.StatusOK, map[string]any{
		"session": session,
		"fee":     fee,
	})
}

func (app *ServerApp) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := app.repo.GetSessions()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "failed to get sessions")
		return
	}

	plate := r.URL.Query().Get("plate")
	if plate != "" {
		filtered := make([]*domain.ParkingSession, 0)
		for _, s := range sessions {
			if s.VehicleID == plate {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	sendJSON(w, http.StatusOK, sessions)
}

func (app *ServerApp) handleAddGate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Distances []struct {
			SlotID   int `json:"slot_id"`
			Distance int `json:"distance"`
		} `json:"distances"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	gates, _ := app.repo.GetGates()
	newGateID := 1
	for _, g := range gates {
		if g.ID >= newGateID {
			newGateID = g.ID + 1
		}
	}

	newGate := &domain.Gate{ID: newGateID, Name: req.Name}
	if err := app.repo.AddGate(newGate); err != nil {
		sendError(w, http.StatusInternalServerError, "failed to add gate")
		return
	}

	// Update slots
	for _, d := range req.Distances {
		slot, err := app.repo.GetSlot(d.SlotID)
		if err == nil {
			if slot.Distances == nil {
				slot.Distances = make(domain.DistanceMap)
			}
			slot.Distances[newGateID] = d.Distance
			_ = app.repo.UpdateSlot(slot)
		}
	}

	_ = app.repo.SaveToFile("state.json")
	sendJSON(w, http.StatusOK, newGate)
}

func (app *ServerApp) handleRemoveGate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	gateID, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid gate ID")
		return
	}

	if err := app.repo.RemoveGate(gateID); err != nil {
		sendError(w, http.StatusNotFound, err.Error())
		return
	}

	_ = app.repo.SaveToFile("state.json")
	sendJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (app *ServerApp) handleAdvanceTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Minutes int `json:"minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Minutes <= 0 {
		sendError(w, http.StatusBadRequest, "minutes must be positive")
		return
	}

	app.mu.Lock()
	app.systemTime = app.systemTime.Add(time.Duration(req.Minutes) * time.Minute)
	newTime := app.systemTime
	app.mu.Unlock()

	_ = app.repo.SaveToFile("state.json") // Save state to persist timing implicitly if needed

	sendJSON(w, http.StatusOK, map[string]any{
		"systemTime": newTime.Format(time.RFC3339),
	})
}
