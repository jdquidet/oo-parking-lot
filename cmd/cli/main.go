package main

import (
	"bufio"
	"fmt"

	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

	// Attempt to load previous state
	_ = repo.LoadFromFile("state.json")

	svc := service.NewParkingService(repo)
	app := &CLIApp{
		repo:       repo,
		service:    svc,
		scanner:    bufio.NewScanner(os.Stdin),
		systemTime: time.Now().Truncate(time.Minute),
	}

	// If no gates were loaded, initialize default map
	gates, _ := repo.GetGates()
	if len(gates) == 0 {
		app.seedDefaultMap()
	}

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
		fmt.Println("5. Remove Gate")
		fmt.Println("6. Display Session Logs")
		fmt.Println("7. Advance System Time")
		fmt.Println("8. Exit")
		fmt.Print("Select an option (1-8): ")

		if !app.scanner.Scan() {
			return
		}
		choice := strings.TrimSpace(app.scanner.Text())

		shouldWait := false
		shouldSave := false
		switch choice {
		case "1":
			shouldWait = app.handlePark()
			shouldSave = true
		case "2":
			shouldWait = app.handleUnpark()
			shouldSave = true
		case "3":
			shouldWait = app.handleDisplayMap()
		case "4":
			shouldWait = app.handleAddGate()
			shouldSave = true
		case "5":
			shouldWait = app.handleRemoveGate()
			shouldSave = true
		case "6":
			shouldWait = app.handleDisplaySessionLogs()
		case "7":
			shouldWait = app.handleAdvanceTime()
			shouldSave = true
		case "8":
			fmt.Println("Exiting system. Goodbye!")
			return
		default:
			continue
		}

		if shouldSave {
			_ = app.repo.SaveToFile("state.json")
		}

		if shouldWait {
			app.waitForEnter()
		}
	}
}

func (app *CLIApp) handlePark() bool {
	fmt.Println("\n--- PARK VEHICLE ---")

	app.printGates()
	gateVal, ok := app.readParsed(
		"Enter Entry Gate ID (q to cancel): ",
		func(s string) (any, error) {
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
		return false
	}
	gateID := gateVal.(int)

	sizeVal, ok := app.readParsed(
		"Select Vehicle Size (0=Small [S], 1=Medium [M], 2=Large [L], q to cancel): ",
		func(s string) (any, error) {
			v, err := strconv.Atoi(s)
			if err != nil || v < 0 || v > 2 {
				return nil, fmt.Errorf("enter 0, 1, or 2")
			}
			return v, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	vSize := domain.VehicleSize(sizeVal.(int))

	if _, err := app.service.FindAvailableSlot(vSize, gateID); err != nil {
		fmt.Printf("\nParking unavailable: %v.\n", err)
		return true
	}

	plateVal, ok := app.readParsed(
		"Enter Vehicle License Plate (letters, numbers, and dashes; q to cancel): ",
		func(s string) (any, error) {
			return domain.ValidateLicensePlate(s)
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	plate := plateVal.(string)

	vehicle := domain.Vehicle{LicensePlate: plate, Size: vSize}
	session, slot, err := app.service.Park(vehicle, gateID, app.systemTime)
	if err != nil {
		fmt.Printf("\nPark Error: %v\n", err)
		return true
	}

	fmt.Println("\nPARKING SUCCESSFUL!")
	fmt.Printf("  Ticket ID:     %s\n", session.ID)
	fmt.Printf("  License Plate: %s\n", session.VehicleID)
	fmt.Printf("  Assigned Slot: #%d (%s)\n", slot.ID, slot.Size.String())
	fmt.Printf("  Entry Time:    %s\n", session.EntryTime.Format("2006-01-02 15:04:05 MST"))
	return true
}

func (app *CLIApp) handleUnpark() bool {
	fmt.Println("\n--- UNPARK VEHICLE ---")

	if app.printParkedVehicles() == 0 {
		return true
	}

	plateVal, ok := app.readParsed(
		"Enter Vehicle License Plate (q to cancel): ",
		func(s string) (any, error) {
			plate, err := domain.ValidateLicensePlate(s)
			if err != nil {
				return nil, err
			}
			if _, err := app.repo.GetActiveSessionByVehicle(plate); err != nil {
				return nil, fmt.Errorf("no parked vehicle found with plate %s", plate)
			}
			return plate, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	plate := plateVal.(string)

	session, fee, err := app.service.Unpark(plate, app.systemTime)
	if err != nil {
		fmt.Printf("\nUnpark Error: %v\n", err)
		return true
	}

	fmt.Println("\nUNPARK RECEIPT")
	fmt.Println("=====================================")
	fmt.Printf("Vehicle Plate:    %s (%s)\n", session.VehicleID, session.VehicleSize.String())
	fmt.Printf("Slot ID Released: #%d (%s)\n", session.SlotID, session.SlotSize.String())
	fmt.Printf("Entry Time:       %s\n", session.EntryTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Exit Time:        %s\n", session.ExitTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Fee Due:    PHP %.2f\n", fee)
	fmt.Println("=====================================")
	return true
}

func (app *CLIApp) handleDisplayMap() bool {
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
	return true
}

func (app *CLIApp) handleAddGate() bool {
	fmt.Println("\n--- ADD NEW GATE ---")
	gates, err := app.repo.GetGates()
	if err != nil {
		fmt.Printf("Unable to list gates: %v\n", err)
		return true
	}
	newGateID := 1
	for _, gate := range gates {
		if gate.ID >= newGateID {
			newGateID = gate.ID + 1
		}
	}

	nameVal, ok := app.readParsed(
		fmt.Sprintf("Enter Name for Gate %d (press Enter for \"Gate %d\", q to cancel): ", newGateID, newGateID),
		func(s string) (any, error) {
			if s == "" {
				return fmt.Sprintf("Gate %d", newGateID), nil
			}
			return s, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	name := nameVal.(string)

	slots, err := app.repo.GetSlots()
	if err != nil {
		fmt.Printf("Unable to list slots: %v\n", err)
		return true
	}
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].ID < slots[j].ID
	})

	fmt.Printf("Configuring distances from %s (ID %d) to all %d slots:\n\n", name, newGateID, len(slots))
	for i, slot := range slots {
		fmt.Printf("  %2d. Slot #%-4d (%s)", i+1, slot.ID, slot.Size.String())
		if (i+1)%3 == 0 || i == len(slots)-1 {
			fmt.Println()
		} else {
			fmt.Print("    ")
		}
	}

	fmt.Println()
	distVal, ok := app.readParsed(
		fmt.Sprintf("Enter %d positive distances (space-separated, matching order above; q to cancel): ", len(slots)),
		func(s string) (any, error) {
			fields := strings.Fields(s)
			if len(fields) != len(slots) {
				return nil, fmt.Errorf("expected %d values, got %d", len(slots), len(fields))
			}
			vals := make([]int, len(fields))
			for i, field := range fields {
				value, err := strconv.Atoi(field)
				if err != nil || value <= 0 {
					return nil, fmt.Errorf("distance must be a positive integer, got %q", field)
				}
				vals[i] = value
			}
			return vals, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	distances := distVal.([]int)

	newGate := &domain.Gate{ID: newGateID, Name: name}
	if err := app.repo.AddGate(newGate); err != nil {
		fmt.Printf("Unable to add gate: %v\n", err)
		return true
	}
	for i, slot := range slots {
		if slot.Distances == nil {
			slot.Distances = make(domain.DistanceMap)
		}
		slot.Distances[newGateID] = distances[i]
		if err := app.repo.UpdateSlot(slot); err != nil {
			_ = app.repo.RemoveGate(newGateID)
			fmt.Printf("Unable to configure gate distances: %v\n", err)
			return true
		}
	}

	fmt.Println("\nDistance Summary:")
	for i, slot := range slots {
		fmt.Printf("  Slot #%-4d (%s): %d units\n", slot.ID, slot.Size.String(), distances[i])
	}
	fmt.Printf("\nSuccessfully added %s (ID %d)!\n", name, newGateID)
	return true
}

func (app *CLIApp) handleRemoveGate() bool {
	fmt.Println("\n--- REMOVE GATE ---")
	if app.printGates() == 0 {
		return true
	}

	gateVal, ok := app.readParsed(
		"Enter Gate ID to remove (q to cancel): ",
		func(s string) (any, error) {
			gateID, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("must be a number")
			}
			if _, err := app.repo.GetGate(gateID); err != nil {
				return nil, fmt.Errorf("gate %d not found", gateID)
			}
			return gateID, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	gateID := gateVal.(int)
	gate, _ := app.repo.GetGate(gateID)

	confirmVal, ok := app.readParsed(
		fmt.Sprintf("Remove %s (ID %d) and its slot distances? (y/n, q to cancel): ", gate.Name, gate.ID),
		func(s string) (any, error) {
			switch strings.ToLower(s) {
			case "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			default:
				return nil, fmt.Errorf("enter y or n")
			}
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	if !confirmVal.(bool) {
		fmt.Println("Gate removal cancelled.")
		return true
	}

	if err := app.repo.RemoveGate(gateID); err != nil {
		fmt.Printf("Unable to remove gate: %v\n", err)
		return true
	}
	fmt.Printf("Successfully removed %s (ID %d).\n", gate.Name, gate.ID)
	return true
}

func (app *CLIApp) handleDisplaySessionLogs() bool {
	fmt.Println("\n--- PARKING SESSION LOGS ---")
	sessions, err := app.repo.GetSessions()
	if err != nil {
		fmt.Printf("Unable to load session logs: %v\n", err)
		return true
	}
	if len(sessions) == 0 {
		fmt.Println("No parking sessions recorded.")
		return true
	}

	filterVal, ok := app.readParsed(
		"Enter a license plate to filter, press Enter to show all, or q to cancel: ",
		func(s string) (any, error) {
			if s == "" {
				return "", nil
			}
			return domain.ValidateLicensePlate(s)
		},
		"q to cancel",
	)
	if !ok {
		return false
	}
	plateFilter := filterVal.(string)

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].EntryTime.Equal(sessions[j].EntryTime) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].EntryTime.After(sessions[j].EntryTime)
	})

	type sessionLogRow struct {
		headerLeft         string
		headerRight        string
		coloredHeaderRight string
		vehicle            string
		assignment         string
		entry              string
		exit               string
		fee                string
	}

	rows := make([]sessionLogRow, 0, len(sessions))
	for _, session := range sessions {
		if plateFilter != "" && session.VehicleID != plateFilter {
			continue
		}

		status := "COMPLETED"
		statusColor := ""
		exitTime := "-"
		if session.IsActive {
			status = "ACTIVE"
			statusColor = "\033[32m"
		} else if session.ExitTime != nil {
			exitTime = session.ExitTime.Format("2006-01-02 15:04:05")
		}

		gateLabel := fmt.Sprintf("Gate #%d", session.GateID)
		if gate, err := app.repo.GetGate(session.GateID); err == nil && gate.Name != "" {
			gateLabel = fmt.Sprintf("%s", gate.Name)
		}

		headerRight := fmt.Sprintf("[ %s ]", status)
		coloredHeaderRight := headerRight
		if statusColor != "" {
			coloredHeaderRight = fmt.Sprintf("[ %s%s\033[0m ]", statusColor, status)
		}

		rows = append(rows, sessionLogRow{
			headerLeft:         fmt.Sprintf("TICKET %s", session.ID),
			headerRight:        headerRight,
			coloredHeaderRight: coloredHeaderRight,
			vehicle:            fmt.Sprintf("%s (%s)", session.VehicleID, session.VehicleSize.String()),
			assignment:         fmt.Sprintf("Slot #%d (%s)  •  Entered at %s", session.SlotID, session.SlotSize.String(), gateLabel),
			entry:              fmt.Sprintf("In:  %s", session.EntryTime.Format("2006-01-02 15:04:05")),
			exit:               fmt.Sprintf("Out: %s", exitTime),
			fee:                fmt.Sprintf("PHP %.2f", session.TotalFeeCharged),
		})
	}

	if len(rows) == 0 {
		fmt.Printf("No parking sessions found for license plate %s.\n", plateFilter)
		return true
	}

	displayWidth := utf8.RuneCountInString
	leftWidth := displayWidth("Assignment")
	rightWidth := 0
	for _, row := range rows {
		for _, value := range []string{row.vehicle, row.assignment, row.entry, row.exit, row.fee} {
			if width := displayWidth(value); width > rightWidth {
				rightWidth = width
			}
		}
	}

	totalWidth := leftWidth + rightWidth + 5
	for _, row := range rows {
		headerWidth := displayWidth(row.headerLeft) + displayWidth(row.headerRight) + 4
		if headerWidth > totalWidth {
			totalWidth = headerWidth
		}
	}
	rightWidth = totalWidth - leftWidth - 5

	const timeline = "  │  "
	fmt.Println("\nNEWEST")
	fmt.Printf("%s┌%s┐\n", timeline, strings.Repeat("─", totalWidth))
	for i, row := range rows {
		headerPadding := totalWidth - displayWidth(row.headerLeft) - displayWidth(row.headerRight) - 2
		fmt.Printf("  ●  │ %s%s%s │\n", row.headerLeft, strings.Repeat(" ", headerPadding), row.coloredHeaderRight)
		fmt.Printf("%s├%s┬%s┤\n", timeline, strings.Repeat("─", leftWidth+2), strings.Repeat("─", rightWidth+2))
		fmt.Printf("%s│ %-*s │ %-*s │\n", timeline, leftWidth, "Vehicle", rightWidth, row.vehicle)
		fmt.Printf("%s│ %-*s │ %-*s │\n", timeline, leftWidth, "Assignment", rightWidth, row.assignment)
		fmt.Printf("%s│ %-*s │ %-*s │\n", timeline, leftWidth, "Timeline", rightWidth, row.entry)
		fmt.Printf("%s│ %-*s │ %-*s │\n", timeline, leftWidth, "", rightWidth, row.exit)
		fmt.Printf("%s│ %-*s │ %-*s │\n", timeline, leftWidth, "Fee", rightWidth, row.fee)

		if i < len(rows)-1 {
			fmt.Printf("%s├%s┴%s┤\n", timeline, strings.Repeat("─", leftWidth+2), strings.Repeat("─", rightWidth+2))
		} else {
			fmt.Printf("%s└%s┴%s┘\n", timeline, strings.Repeat("─", leftWidth+2), strings.Repeat("─", rightWidth+2))
		}
	}
	fmt.Println("  ▼")
	fmt.Println("OLDER")
	return true
}

func (app *CLIApp) handleAdvanceTime() bool {
	fmt.Println("\n--- ADVANCE SYSTEM TIME ---")
	fmt.Printf("Current Virtual Clock: %s\n", app.systemTime.Format("2006-01-02 15:04"))

	durVal, ok := app.readParsed(
		"Enter a positive duration to jump forward (e.g., 3h, 45m, 1h30m; q to cancel): ",
		func(s string) (any, error) {
			dur, err := time.ParseDuration(s)
			if err != nil {
				return nil, fmt.Errorf("invalid format; use proper duration syntax such as 3h or 30m")
			}
			if dur <= 0 {
				return nil, fmt.Errorf("enter a positive duration")
			}
			return dur, nil
		},
		"q to cancel",
	)
	if !ok {
		return false
	}

	duration := durVal.(time.Duration)
	app.systemTime = app.systemTime.Add(duration)

	fmt.Printf("\nSuccess! System time fast-forwarded by %s.\n", duration.String())
	fmt.Printf("New Virtual Clock: %s\n", app.systemTime.Format("2006-01-02 15:04 MST"))
	return true
}

func (app *CLIApp) readParsed(prompt string, parse func(string) (any, error), cancelHint string) (any, bool) {
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

func (app *CLIApp) printGates() int {
	gates, err := app.repo.GetGates()
	if err != nil {
		fmt.Printf("Unable to list gates: %v\n", err)
		return 0
	}

	sort.Slice(gates, func(i, j int) bool {
		return gates[i].ID < gates[j].ID
	})

	fmt.Println("Available Gates:")
	for _, gate := range gates {
		fmt.Printf("  ID %d: %s\n", gate.ID, gate.Name)
	}
	if len(gates) == 0 {
		fmt.Println("  (none)")
	}
	return len(gates)
}
