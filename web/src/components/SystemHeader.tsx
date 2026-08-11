import { useState } from 'react'
import { advanceTime } from '../api/parking'

interface SystemHeaderProps {
  systemTime?: string
  isRefreshing: boolean
  onMutate: () => void
}

const clockFormatter = new Intl.DateTimeFormat('en-CA', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

function formatSystemTime(value?: string): string {
  if (!value) return '---- -- -- · --:--'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '---- -- -- · --:--'

  return clockFormatter.format(date).replace(',', ' ·')
}

export function SystemHeader({
  systemTime,
  isRefreshing,
  onMutate,
}: SystemHeaderProps) {
  const [minutes, setMinutes] = useState('60')
  const [isAdvancing, setIsAdvancing] = useState(false)
  const [error, setError] = useState<string>()

  const handleAdvance = async (e: React.FormEvent) => {
    e.preventDefault()
    const mins = parseInt(minutes, 10)
    if (isNaN(mins) || mins <= 0) {
      setError('Invalid')
      return
    }

    setIsAdvancing(true)
    setError(undefined)
    try {
      await advanceTime(mins)
      onMutate()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error')
    } finally {
      setIsAdvancing(false)
    }
  }

  return (
    <header className="system-header">
      <div>
        <p className="eyebrow">Ayala Corporation</p>
        <h1>Parking operations</h1>
      </div>

      <div className="clock-section">
        <div className="clock" aria-live="polite">
          <span className="clock-label">
            Virtual clock
            <span
              className={`refresh-mark ${isRefreshing ? 'is-active' : ''}`}
              aria-label={isRefreshing ? 'Refreshing system state' : undefined}
            />
          </span>
          <time dateTime={systemTime}>{formatSystemTime(systemTime)}</time>
        </div>

        <form className="advance-form" onSubmit={(e) => void handleAdvance(e)}>
          <span className="index-label">Advance clock</span>
          <div className="advance-controls">
            <input
              type="number"
              min="1"
              step="1"
              value={minutes}
              onChange={(e) => setMinutes(e.target.value)}
              disabled={isAdvancing}
              aria-label="Minutes to advance"
            />
            <span>min</span>
            <button type="submit" disabled={isAdvancing}>
              {isAdvancing ? '...' : '+'}
            </button>
          </div>
          {error && <span className="advance-error">{error}</span>}
        </form>
      </div>
    </header>
  )
}
