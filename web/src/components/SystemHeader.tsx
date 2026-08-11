interface SystemHeaderProps {
  systemTime?: string
  isRefreshing: boolean
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
}: SystemHeaderProps) {
  return (
    <header className="system-header">
      <div>
        <p className="eyebrow">Ayala Corporation</p>
        <h1>Parking operations</h1>
      </div>

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
    </header>
  )
}
