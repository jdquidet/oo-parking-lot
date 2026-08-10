package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"github.com/jdquidet/oo-parking-lot/pkg/repository"
	"github.com/jdquidet/oo-parking-lot/pkg/service"
)

var cliTestTime = time.Date(2026, time.August, 11, 17, 0, 0, 0, time.UTC)

func TestHandleParkUsesGateSizePlateOrder(t *testing.T) {
	app, repo := newTestCLI("1\n0\nabc-123\n")
	addTestGateAndSlot(t, repo, false)

	output := captureStdout(t, func() {
		if !app.handlePark() {
			t.Fatal("expected successful parking to request a pause")
		}
	})

	gateIndex := strings.Index(output, "Enter Entry Gate ID")
	sizeIndex := strings.Index(output, "Select Vehicle Size")
	plateIndex := strings.Index(output, "Enter Vehicle License Plate")
	if gateIndex < 0 || sizeIndex < 0 || plateIndex < 0 || !(gateIndex < sizeIndex && sizeIndex < plateIndex) {
		t.Fatalf("expected gate, size, then plate prompts; output:\n%s", output)
	}
	if !strings.Contains(output, "License Plate: ABC-123") {
		t.Fatalf("expected valid plate on successful ticket; output:\n%s", output)
	}
	if _, err := repo.GetActiveSessionByVehicle("ABC-123"); err != nil {
		t.Fatalf("expected valid active session: %v", err)
	}
}

func TestHandleParkReportsUnavailableBeforePlatePrompt(t *testing.T) {
	app, repo := newTestCLI("1\n2\n")
	addTestGateAndSlot(t, repo, true)

	output := captureStdout(t, func() {
		if !app.handlePark() {
			t.Fatal("expected unavailable result to request a pause")
		}
	})

	if !strings.Contains(output, "Parking unavailable") {
		t.Fatalf("expected parking unavailability message; output:\n%s", output)
	}
	if strings.Contains(output, "Enter Vehicle License Plate") {
		t.Fatalf("license plate must not be requested when no slot is available; output:\n%s", output)
	}
}

func TestHandleAddGateCancellationDoesNotPersistPartialGate(t *testing.T) {
	app, repo := newTestCLI("North Gate\nq\n")
	if err := repo.AddGate(&domain.Gate{ID: 1, Name: "Gate A"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddGate(&domain.Gate{ID: 3, Name: "Gate C"}); err != nil {
		t.Fatal(err)
	}
	slot := &domain.ParkingSlot{ID: 101, Size: domain.SlotSP, Distances: domain.DistanceMap{1: 1, 3: 3}}
	if err := repo.AddSlot(slot); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if app.handleAddGate() {
			t.Fatal("expected q to cancel without requesting a pause")
		}
	})

	if _, err := repo.GetGate(4); err == nil {
		t.Fatal("cancelled gate setup must not persist the new gate")
	}
	storedSlot, err := repo.GetSlot(101)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := storedSlot.Distances[4]; exists {
		t.Fatal("cancelled gate setup must not persist a distance mapping")
	}
	if !strings.Contains(output, "Gate 4") {
		t.Fatalf("expected next ID to use max existing ID + 1; output:\n%s", output)
	}
}

func TestHandleAddGateRejectsZeroDistance(t *testing.T) {
	app, repo := newTestCLI("North Gate\n0\nq\n")
	if err := repo.AddSlot(&domain.ParkingSlot{ID: 101, Size: domain.SlotSP, Distances: domain.DistanceMap{}}); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if app.handleAddGate() {
			t.Fatal("expected q after invalid distance to cancel")
		}
	})

	if !strings.Contains(output, "distance must be a positive integer") {
		t.Fatalf("expected positive-distance validation; output:\n%s", output)
	}
	if _, err := repo.GetGate(1); err == nil {
		t.Fatal("invalid/cancelled gate setup must not persist a gate")
	}
}

func TestHandleRemoveGate(t *testing.T) {
	app, repo := newTestCLI("1\ny\n")
	addTestGateAndSlot(t, repo, false)

	captureStdout(t, func() {
		if !app.handleRemoveGate() {
			t.Fatal("expected confirmed removal to request a pause")
		}
	})

	if _, err := repo.GetGate(1); err == nil {
		t.Fatal("expected gate to be removed")
	}
	slot, err := repo.GetSlot(101)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := slot.Distances[1]; exists {
		t.Fatal("expected removed gate distance to be deleted from slot")
	}
}

func TestRunRetriesBlankMenuInputWithoutPause(t *testing.T) {
	app, _ := newTestCLI("\n8\n")

	output := captureStdout(t, app.run)

	if strings.Count(output, "Select an option (1-8)") != 2 {
		t.Fatalf("expected blank input to immediately redraw the menu; output:\n%s", output)
	}
	if strings.Contains(output, "Press Enter to return") {
		t.Fatalf("invalid menu input must not request another Enter; output:\n%s", output)
	}
	if !strings.Contains(output, "Exiting system. Goodbye!") {
		t.Fatalf("expected second input to exit; output:\n%s", output)
	}
}

func TestRunCancellationReturnsImmediatelyToMenu(t *testing.T) {
	app, _ := newTestCLI("4\nq\n8\n")

	output := captureStdout(t, app.run)

	if strings.Contains(output, "Press Enter to return") {
		t.Fatalf("cancelled workflow must return immediately to the menu; output:\n%s", output)
	}
	if !strings.Contains(output, "Exiting system. Goodbye!") {
		t.Fatalf("expected exit input to remain available after cancellation; output:\n%s", output)
	}
}

func TestHandleDisplaySessionLogsSupportsValidPlateFilter(t *testing.T) {
	app, repo := newTestCLI("abc-123\n")
	session := &domain.ParkingSession{
		ID:          "SESS-ABC-123",
		VehicleID:   "ABC-123",
		VehicleSize: domain.SizeSmall,
		SlotID:      101,
		SlotSize:    domain.SlotSP,
		GateID:      1,
		EntryTime:   cliTestTime,
		IsActive:    true,
	}
	if err := repo.SaveSession(session); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if !app.handleDisplaySessionLogs() {
			t.Fatal("expected displayed logs to request a pause")
		}
	})

	for _, expected := range []string{"SESS-ABC-123", "License Plate: ABC-123", "Status:        ACTIVE"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q in session log output:\n%s", expected, output)
		}
	}
}

func newTestCLI(input string) (*CLIApp, *repository.MemoryRepository) {
	repo := repository.NewMemoryRepository()
	return &CLIApp{
		repo:       repo,
		service:    service.NewParkingService(repo),
		scanner:    bufio.NewScanner(strings.NewReader(input)),
		systemTime: cliTestTime,
	}, repo
}

func addTestGateAndSlot(t *testing.T, repo *repository.MemoryRepository, occupied bool) {
	t.Helper()
	if err := repo.AddGate(&domain.Gate{ID: 1, Name: "Gate A"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddSlot(&domain.ParkingSlot{
		ID:               101,
		Size:             domain.SlotLP,
		Distances:        domain.DistanceMap{1: 1},
		IsOccupied:       occupied,
		CurrentVehicleID: map[bool]string{true: "OCC-001", false: ""}[occupied],
	}); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original

	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
