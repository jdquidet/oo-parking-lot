import { useCallback, useEffect, useState } from 'react'
import { getParkingState } from './api/parking'
import { LotMap } from './components/LotMap'
import { ParkPanel } from './components/ParkPanel'
import { SystemHeader } from './components/SystemHeader'
import type { ParkingState } from './types'

const POLL_INTERVAL_MS = 5_000

function App() {
  const [state, setState] = useState<ParkingState>()
  const [error, setError] = useState<string>()
  const [isRefreshing, setIsRefreshing] = useState(false)

  const loadState = useCallback(async (signal?: AbortSignal) => {
    setIsRefreshing(true)

    try {
      const nextState = await getParkingState(signal)
      setState(nextState)
      setError(undefined)
    } catch (caught) {
      if (caught instanceof DOMException && caught.name === 'AbortError') return
      setError(
        caught instanceof Error ? caught.message : 'Unable to load system state',
      )
    } finally {
      if (!signal?.aborted) setIsRefreshing(false)
    }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    void loadState(controller.signal)

    const interval = window.setInterval(() => {
      void loadState(controller.signal)
    }, POLL_INTERVAL_MS)

    return () => {
      controller.abort()
      window.clearInterval(interval)
    }
  }, [loadState])

  const triggerRefresh = useCallback(() => {
    void loadState()
  }, [loadState])

  const occupiedPlates =
    state?.slots
      .filter((s) => s.is_occupied && s.current_vehicle_id)
      .map((s) => s.current_vehicle_id!) ?? []

  return (
    <div className="app-shell">
      <SystemHeader
        systemTime={state?.systemTime}
        isRefreshing={isRefreshing}
        onMutate={triggerRefresh}
      />

      <main>
        {error && (
          <div className="state-message is-error" role="alert">
            <span>Connection interrupted</span>
            <p>{error}. Confirm the Go server is running on port 8080.</p>
            <button type="button" onClick={triggerRefresh}>
              Retry
            </button>
          </div>
        )}

        {!state && !error && (
          <div className="state-message" role="status">
            <span>Reading parking state</span>
            <p>Synchronising slots, gates, and virtual clock.</p>
          </div>
        )}

        {state && (
          <div className="dashboard-grid">
            <div className="dashboard-col">
              <ParkPanel
                gates={state.gates}
                occupiedPlates={occupiedPlates}
                onMutate={triggerRefresh}
              />
            </div>
            <div className="dashboard-col">
              <LotMap gates={state.gates} slots={state.slots} />
            </div>
          </div>
        )}
      </main>

      <footer>
        <span>Object-oriented mall parking</span>
        <span>State refresh / 5 sec</span>
      </footer>
    </div>
  )
}

export default App
