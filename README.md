# OO Parking Lot Allocation System

An object-oriented mall parking lot management system with a Go backend and a React frontend. It assigns vehicles to the closest available slot relative to their entry gate, tracks parking sessions, and computes fees based on elapsed time.

## Architecture

- `cmd/cli/` — interactive terminal application (main menu, park/unpark, occupancy map, gate management, session logs, virtual clock)
- `cmd/server/` — HTTP API server (serves the web frontend's data, listens on `:8080`)
- `web/` — React + TypeScript + Tailwind frontend (Vite dev server)
- `pkg/` — domain models, repository, and service logic (shared by CLI and server)
- `state.json` — persisted parking-lot state (gates, slots, sessions); created automatically and gitignored

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+ (see `go.mod`)
- [Node.js](https://nodejs.org/) 20+ and npm (for the web frontend)

## Running the CLI

```bash
go run ./cmd/cli
```

The CLI loads `state.json` if present; otherwise it seeds a default parking lot (gates A, B, C and 9 slots) on first run. State is saved automatically after each operation.

## Running the Web App

Run the API server and the frontend dev server in two terminals.

Terminal 1 — API server:

```bash
go run ./cmd/server
```

Terminal 2 — frontend:

```bash
cd web
npm install
npm run dev
```

Then open the URL Vite prints (typically `http://localhost:5173`). The Vite dev server proxies `/api` requests to `http://localhost:8080`, so no extra configuration is needed.

## Running the Tests

```bash
go test ./...
```

For the frontend:

```bash
cd web
npm run lint
npm run build
```

## API Endpoints

| Method   | Path                | Description                       |
| -------- | ------------------- | --------------------------------- |
| `GET`    | `/api/state`        | Gates, slots, and system time     |
| `POST`   | `/api/park`         | Park a vehicle                    |
| `POST`   | `/api/unpark`       | Unpark a vehicle                  |
| `GET`    | `/api/sessions`     | Session logs (optional `?plate=`) |
| `POST`   | `/api/gates`        | Add a gate                        |
| `DELETE` | `/api/gates/{id}`   | Remove a gate                     |
| `POST`   | `/api/time/advance` | Advance the virtual clock         |
