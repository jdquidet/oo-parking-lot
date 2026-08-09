package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"github.com/jdquidet/oo-parking-lot/pkg/repository"
	"github.com/jdquidet/oo-parking-lot/pkg/service"
)

type CLIApp struct {
	repo       repository.ParkingRepository
	service    service.ParkingService
	scanner    *bufio.Scanner
	systemTime time.Time
}

func main() {
	repo := repository.NewMemoryRepository()
	svc := service.NewParkingService(repo)
	app := &CLIApp{
		repo:       repo,
		service:    svc,
		scanner:    bufio.NewScanner(os.Stdin),
		systemTime: time.Now().Truncate(time.Minute),
	}

	app.seedDefaultMap()
	app.run()
}

func (app *CLIApp) seedDefaultMap() {
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

func (app *CLIApp) run() {

	for {
		clearScreen()

		fmt.Println("===============================================")
		fmt.Println("   AYALA CORP - OBJECT-ORIENTED MALL PARKING")
		fmt.Println("===============================================")
		fmt.Printf("   [ VIRTUAL CLOCK: %s ]\n", app.systemTime.Format("2006-01-02 15:04 MST"))
		fmt.Println("-----------------------------------------------")
		fmt.Println("--- MAIN MENU ---")
		fmt.Println("1. Park a Vehicle")
		fmt.Println("2. Unpark a Vehicle")
		fmt.Println("3. Display Parking Lot Occupancy")
		fmt.Println("4. Add New Gate")
		fmt.Println("5. Advance System Time")
		fmt.Println("6. Exit")
		fmt.Print("Select an option (1-6): ")

		if !app.scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(app.scanner.Text())

		switch choice {
		case "1":
			app.handlePark()
		case "2":
			app.handleUnpark()
		case "3":
			app.handleDisplayMap()
		case "4":
			app.handleAddGate()
		case "5":
			app.handleAdvanceTime()
		case "6":
			fmt.Println("Exiting system. Goodbye!")
			return
		default:
			fmt.Println("Invalid selection. Please choose 1-6.")
		}

		app.waitForEnter()
	}
}

func (app *CLIApp) handlePark() {
	fmt.Println("\n--- PARK VEHICLE ---")

	plateVal, ok := app.readParsed(
		"Enter Vehicle License Plate (or press Enter for random): ",
		func(s string) (interface{}, error) {
			if s == "" {
				return generateRandomPlate(), nil
			}
			return strings.ToUpper(s), nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}
	plate := plateVal.(string)

	sizeVal, ok := app.readParsed(
		"Select Vehicle Size (0=Small [S], 1=Medium [M], 2=Large [L]): ",
		func(s string) (interface{}, error) {
			v, err := strconv.Atoi(s)
			if err != nil || v < 0 || v > 2 {
				return nil, fmt.Errorf("enter 0, 1, or 2")
			}
			return v, nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}
	vSize := domain.VehicleSize(sizeVal.(int))

	app.printGates()
	gateVal, ok := app.readParsed(
		"Enter Entry Gate ID: ",
		func(s string) (interface{}, error) {
			v, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("must be a number")
			}
			if _, err := app.repo.GetGate(v); err != nil {
				return nil, fmt.Errorf("gate %d not found", v)
			}
			return v, nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}
	gateID := gateVal.(int)

	vehicle := domain.Vehicle{LicensePlate: plate, Size: vSize}
	session, slot, err := app.service.Park(vehicle, gateID, app.systemTime)
	if err != nil {
		fmt.Printf("\nPark Error: %v\n", err)
		return
	}

	dist, _ := slot.DistanceFrom(gateID)
	fmt.Println("\nPARKING SUCCESSFUL!")
	fmt.Printf("  Ticket ID:     %s\n", session.ID)
	fmt.Printf("  Assigned Slot: #%d (%s)\n", slot.ID, slot.Size.String())
	fmt.Printf("  Distance:      %d units from Gate %d\n", dist, gateID)
	fmt.Printf("  Entry Time:    %s\n", session.EntryTime.Format("2006-01-02 15:04:05 MST"))
}

func (app *CLIApp) handleUnpark() {
	fmt.Println("\n--- UNPARK VEHICLE ---")

	if app.printParkedVehicles() == 0 {
		return
	}

	plateVal, ok := app.readParsed(
		"Enter Vehicle License Plate (q to cancel): ",
		func(s string) (interface{}, error) {
			p := strings.ToUpper(s)
			_, err := app.repo.GetActiveSessionByVehicle(p)
			if err != nil {
				return nil, fmt.Errorf("no parked vehicle found with plate %s", p)
			}
			return p, nil
		},
		"",
	)
	if !ok {
		return
	}
	plate := plateVal.(string)

	session, fee, err := app.service.Unpark(plate, app.systemTime)
	if err != nil {
		fmt.Printf("\nUnpark Error: %v\n", err)
		return
	}

	fmt.Println("\nUNPARK RECEIPT")
	fmt.Println("=====================================")
	fmt.Printf("Vehicle Plate:    %s (%s)\n", session.VehicleID, session.VehicleSize.String())
	fmt.Printf("Slot ID Released: #%d (%s)\n", session.SlotID, session.SlotSize.String())
	fmt.Printf("Entry Time:       %s\n", session.EntryTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Exit Time:        %s\n", session.ExitTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Fee Due:    PHP %.2f\n", fee)
	fmt.Println("=====================================")
}

func (app *CLIApp) handleDisplayMap() {
	fmt.Println("\n--- PARKING LOT OCCUPANCY ---")
	gates, _ := app.repo.GetGates()
	slots, _ := app.repo.GetSlots()

	sort.Slice(gates, func(i, j int) bool {
		return gates[i].ID < gates[j].ID
	})

	sort.Slice(slots, func(i, j int) bool {
		return slots[i].ID < slots[j].ID
	})

	// Initial minimum widths based on the column headers
	wSlot, wSize, wStatus, wVeh, wDist := 4, 4, 6, 7, 9

	// Struct to cache parsed row data to not calculate it twice
	type rowData struct {
		idStr   string
		sizeStr string
		status  string
		veh     string
		dist    string
		isOcc   bool
	}

	var rows []rowData

	// Compute actual max widths and cache row data
	for _, s := range slots {
		idStr := fmt.Sprintf("#%d", s.ID)
		sizeStr := s.Size.String()

		status := "FREE"
		veh := "-"
		if s.IsOccupied {
			status = "OCCUPIED"
			veh = s.CurrentVehicleID
		}

		distParts := make([]string, 0, len(gates))
		for _, g := range gates {
			if d, err := s.DistanceFrom(g.ID); err == nil {
				distParts = append(distParts, fmt.Sprintf("%d:%d", g.ID, d))
			}
		}
		dist := strings.Join(distParts, "  ")

		// Update max widths if the current cell is longer than the header
		if len(idStr) > wSlot {
			wSlot = len(idStr)
		}
		if len(sizeStr) > wSize {
			wSize = len(sizeStr)
		}
		if len(status) > wStatus {
			wStatus = len(status)
		}
		if len(veh) > wVeh {
			wVeh = len(veh)
		}
		if len(dist) > wDist {
			wDist = len(dist)
		}

		rows = append(rows, rowData{idStr, sizeStr, status, veh, dist, s.IsOccupied})
	}

	// Helper function to draw horizontal borders dynamically
	drawBorder := func(left, mid, right string) {
		fmt.Printf("%s%s%s%s%s%s%s%s%s%s%s\n",
			left, strings.Repeat("─", wSlot+2),
			mid, strings.Repeat("─", wSize+2),
			mid, strings.Repeat("─", wStatus+2),
			mid, strings.Repeat("─", wVeh+2),
			mid, strings.Repeat("─", wDist+2), right)
	}

	// Print the Table
	drawBorder("┌", "┬", "┐")

	// Print Headers using Go's dynamic width format specifier: %-*s
	fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
		wSlot, "SLOT", wSize, "SIZE", wStatus, "STATUS", wVeh, "VEHICLE", wDist, "DISTANCES")

	drawBorder("├", "┼", "┤")

	prevRow := -1
	for i, s := range slots {
		row := s.ID / 100
		if prevRow != -1 && row != prevRow {
			drawBorder("├", "┼", "┤")
		}
		prevRow = row

		r := rows[i]

		// Apply color independently after raw length is accounted for
		statusColor := "\033[32m" // Green
		if r.isOcc {
			statusColor = "\033[31m" // Red
		}
		paddedStatus := fmt.Sprintf("%-*s", wStatus, r.status)
		coloredStatus := fmt.Sprintf("%s%s\033[0m", statusColor, paddedStatus)

		fmt.Printf("│ %-*s │ %-*s │ %s │ %-*s │ %-*s │\n",
			wSlot, r.idStr,
			wSize, r.sizeStr,
			coloredStatus,
			wVeh, r.veh,
			wDist, r.dist)
	}

	drawBorder("└", "┴", "┘")

	fmt.Println("\nGates:")
	for _, g := range gates {
		fmt.Printf("  ID %d: %s\n", g.ID, g.Name)
	}
}

func (app *CLIApp) handleAddGate() {
	fmt.Println("\n--- ADD NEW GATE ---")
	gates, _ := app.repo.GetGates()
	newGateID := len(gates) + 1

	nameVal, ok := app.readParsed(
		fmt.Sprintf("Enter Name for Gate %d (or press Enter for \"Gate %d\"): ", newGateID, newGateID),
		func(s string) (interface{}, error) {
			if s == "" {
				return fmt.Sprintf("Gate %d", newGateID), nil
			}
			return s, nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}
	name := nameVal.(string)

	newGate := &domain.Gate{ID: newGateID, Name: name}
	_ = app.repo.AddGate(newGate)

	slots, _ := app.repo.GetSlots()
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].ID < slots[j].ID
	})

	fmt.Printf("Configuring distances from %s (ID %d) to all %d slots:\n\n", name, newGateID, len(slots))

	for i, s := range slots {
		fmt.Printf("  %2d. Slot #%-4d (%s)", i+1, s.ID, s.Size.String())
		if (i+1)%3 == 0 || i == len(slots)-1 {
			fmt.Println()
		} else {
			fmt.Print("    ")
		}
	}

	fmt.Println()
	distVal, ok := app.readParsed(
		fmt.Sprintf("Enter %d distances (space-separated, matching order above): ", len(slots)),
		func(s string) (interface{}, error) {
			fields := strings.Fields(s)
			if len(fields) != len(slots) {
				return nil, fmt.Errorf("expected %d values, got %d", len(slots), len(fields))
			}
			vals := make([]int, len(fields))
			for i, f := range fields {
				v, err := strconv.Atoi(f)
				if err != nil || v < 0 {
					return nil, fmt.Errorf("distance must be a non-negative integer, got %q", f)
				}
				vals[i] = v
			}
			return vals, nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}
	distances := distVal.([]int)

	fmt.Println("\nDistance Summary:")
	for i, s := range slots {
		s.Distances[newGateID] = distances[i]
		_ = app.repo.UpdateSlot(s)
		fmt.Printf("  Slot #%-4d (%s): %d units\n", s.ID, s.Size.String(), distances[i])
	}

	fmt.Printf("\nSuccessfully added %s (ID %d)!\n", name, newGateID)
}

func (app *CLIApp) handleAdvanceTime() {
	fmt.Println("\n--- ADVANCE SYSTEM TIME ---")
	fmt.Printf("Current Virtual Clock: %s\n", app.systemTime.Format("2006-01-02 15:04"))

	durVal, ok := app.readParsed(
		"Enter duration to jump forward (e.g., 3h, 45m, 1h30m): ",
		func(s string) (interface{}, error) {
			dur, err := time.ParseDuration(s)
			if err != nil {
				return nil, fmt.Errorf("invalid format. Use Go duration syntax (3h, 30m)")
			}
			if dur < 0 {
				return nil, fmt.Errorf("time travel into the past is not allowed")
			}
			return dur, nil
		},
		"q to cancel",
	)
	if !ok {
		return
	}

	duration := durVal.(time.Duration)
	app.systemTime = app.systemTime.Add(duration)

	fmt.Printf("\nSuccess! System time fast-forwarded by %s.\n", duration.String())
	fmt.Printf("New Virtual Clock: %s\n", app.systemTime.Format("2006-01-02 15:04 MST"))
}

func (app *CLIApp) readParsed(prompt string, parse func(string) (interface{}, error), cancelHint string) (interface{}, bool) {
	for {
		fmt.Print(prompt)
		if !app.scanner.Scan() {
			return nil, false
		}
		input := strings.TrimSpace(app.scanner.Text())
		if input == "q" || input == "Q" {
			return nil, false
		}

		val, err := parse(input)
		if err != nil {
			fmt.Printf("  Invalid input: %v. ", err)
			if cancelHint != "" {
				fmt.Printf("(%s)\n", cancelHint)
			} else {
				fmt.Println("Try again.")
			}
			continue
		}
		return val, true
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (app *CLIApp) waitForEnter() {
	fmt.Println("\n[Press Enter to return to the main menu...]")
	app.scanner.Scan()
}

func generateRandomPlate() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	plate := make([]byte, 8)
	for i := 0; i < 3; i++ {
		plate[i] = letters[rand.Intn(len(letters))]
	}
	plate[3] = '-'
	for i := 4; i < 8; i++ {
		plate[i] = digits[rand.Intn(len(digits))]
	}
	return string(plate)
}

func (app *CLIApp) printParkedVehicles() int {
	slots, _ := app.repo.GetSlots()

	sort.Slice(slots, func(i, j int) bool {
		return slots[i].ID < slots[j].ID
	})

	fmt.Println("Currently Parked Vehicles:")
	count := 0
	for _, s := range slots {
		if s.IsOccupied {
			fmt.Printf("  Slot #%d: %s (%s)\n", s.ID, s.CurrentVehicleID, s.Size.String())
			count++
		}
	}
	if count == 0 {
		fmt.Println("  (none)")
	}
	return count
}

func (app *CLIApp) printGates() {
	gates, _ := app.repo.GetGates()

	sort.Slice(gates, func(i, j int) bool {
		return gates[i].ID < gates[j].ID
	})

	fmt.Println("Available Gates:")
	for _, g := range gates {
		fmt.Printf("  ID %d: %s\n", g.ID, g.Name)
	}
}
