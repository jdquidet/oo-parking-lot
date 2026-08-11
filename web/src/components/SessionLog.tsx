import { useEffect, useState } from 'react'
import { getSessions } from '../api/parking'
import type { ParkingSession } from '../types'
import { slotSizeLabels } from '../types'

interface SessionLogProps {
  gates: Array<{ id: number; name: string }>
}

export function SessionLog({ gates }: SessionLogProps) {
  const [sessions, setSessions] = useState<ParkingSession[]>([])
  const [filter, setFilter] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string>()

  useEffect(() => {
    const loadSessions = async () => {
      setIsLoading(true)
      setError(undefined)
      try {
        const data = await getSessions(filter || undefined)
        setSessions(
          data.sort((a, b) => {
            const aTime = new Date(a.entry_time).getTime()
            const bTime = new Date(b.entry_time).getTime()
            return bTime - aTime
          }),
        )
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load sessions')
      } finally {
        setIsLoading(false)
      }
    }

    const timer = setTimeout(() => {
      void loadSessions()
    }, 300)

    return () => clearTimeout(timer)
  }, [filter])

  const gateMap = new Map(gates.map((g) => [g.id, g]))

  const filteredSessions = sessions.filter((s) =>
    filter ? s.vehicle_id.toUpperCase().includes(filter.toUpperCase()) : true,
  )

  return (
    <section className="log-section" aria-labelledby="log-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Records / 03</p>
          <h2 id="log-heading">Session logs</h2>
        </div>
      </div>

      <div className="log-filter">
        <label htmlFor="filter-plate">Filter by plate</label>
        <input
          id="filter-plate"
          type="text"
          placeholder="ABC-1234"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          autoComplete="off"
        />
        {filter && (
          <button
            type="button"
            className="clear-filter"
            onClick={() => setFilter('')}
            aria-label="Clear filter"
          >
            ✕
          </button>
        )}
      </div>

      {error && (
        <div className="log-error" role="alert">
          {error}
        </div>
      )}

      {isLoading && (
        <div className="log-loading" role="status">
          Loading sessions...
        </div>
      )}

      {filteredSessions.length === 0 && !isLoading && (
        <div className="log-empty">
          {filter
            ? `No sessions found for ${filter}`
            : 'No parking sessions recorded yet'}
        </div>
      )}

      <div className="log-timeline">
        {filteredSessions.map((session, idx) => {
          const entryTime = new Date(session.entry_time)
          const exitTime = session.exit_time
            ? new Date(session.exit_time)
            : null
          const gateLabel = gateMap.get(session.gate_id)?.name || `Gate #${session.gate_id}`

          return (
            <article
              className={`log-entry ${session.is_active ? 'is-active' : 'is-completed'}`}
              key={session.id}
            >
              <div className="entry-marker" aria-hidden="true" />

              <div className="entry-header">
                <span className="entry-id">{session.id}</span>
                <span
                  className={`entry-status ${session.is_active ? 'is-active' : ''}`}
                >
                  {session.is_active ? 'ACTIVE' : 'COMPLETED'}
                </span>
              </div>

              <dl className="entry-details">
                <div>
                  <dt>Vehicle</dt>
                  <dd>
                    {session.vehicle_id} <i>({slotSizeLabels[session.vehicle_size]})</i>
                  </dd>
                </div>
                <div>
                  <dt>Assignment</dt>
                  <dd>
                    Slot #{session.slot_id} ({slotSizeLabels[session.slot_size]}) · {gateLabel}
                  </dd>
                </div>
                <div>
                  <dt>In</dt>
                  <dd>{entryTime.toLocaleString('en-CA')}</dd>
                </div>
                <div>
                  <dt>Out</dt>
                  <dd>{exitTime ? exitTime.toLocaleString('en-CA') : '—'}</dd>
                </div>
                {exitTime && (
                  <div>
                    <dt>Duration</dt>
                    <dd>{formatDuration(entryTime, exitTime)}</dd>
                  </div>
                )}
                {!session.is_active && (
                  <div>
                    <dt>Fee</dt>
                    <dd className="fee-amount">PHP {session.total_fee_charged.toFixed(2)}</dd>
                  </div>
                )}
              </dl>

              {idx < filteredSessions.length - 1 && (
                <div className="entry-divider" aria-hidden="true" />
              )}
            </article>
          )
        })}
      </div>
    </section>
  )
}

function formatDuration(start: Date, end: Date): string {
  const ms = end.getTime() - start.getTime()
  const hours = Math.floor(ms / (1000 * 60 * 60))
  const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60))
  if (hours === 0) return `${minutes}m`
  return `${hours}h ${minutes}m`
}
