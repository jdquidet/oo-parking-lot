import type { Gate, ParkingSlot } from '../types'
import { slotSizeLabels } from '../types'

interface LotMapProps {
  gates: Gate[]
  slots: ParkingSlot[]
}

function groupSlots(slots: ParkingSlot[]): Array<[number, ParkingSlot[]]> {
  const groups = new Map<number, ParkingSlot[]>()

  for (const slot of [...slots].sort((a, b) => a.id - b.id)) {
    const level = Math.floor(slot.id / 100)
    groups.set(level, [...(groups.get(level) ?? []), slot])
  }

  return [...groups.entries()]
}

export function LotMap({ gates, slots }: LotMapProps) {
  const orderedGates = [...gates].sort((a, b) => a.id - b.id)
  const occupiedCount = slots.filter((slot) => slot.is_occupied).length
  const occupancyPercent = slots.length
    ? Math.round((occupiedCount / slots.length) * 100)
    : 0

  return (
    <section className="lot-section" aria-labelledby="lot-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">System state / 01</p>
          <h2 id="lot-heading">Parking lot occupancy</h2>
        </div>

        <dl className="summary-strip">
          <div>
            <dt>Capacity</dt>
            <dd>{slots.length.toString().padStart(2, '0')}</dd>
          </div>
          <div>
            <dt>Occupied</dt>
            <dd>{occupiedCount.toString().padStart(2, '0')}</dd>
          </div>
          <div>
            <dt>Load</dt>
            <dd>{occupancyPercent}%</dd>
          </div>
        </dl>
      </div>

      <div className="gate-index" aria-label="Gate index">
        <span className="index-label">Distance index</span>
        {orderedGates.map((gate) => (
          <span key={gate.id}>
            G{gate.id} <i>{gate.name}</i>
          </span>
        ))}
      </div>

      <div className="lot-grid">
        {groupSlots(slots).map(([level, levelSlots]) => (
          <section className="lot-level" key={level}>
            <header>
              <span>Zone {level.toString().padStart(2, '0')}</span>
              <span>
                {levelSlots.filter((slot) => !slot.is_occupied).length}/
                {levelSlots.length} available
              </span>
            </header>

            <div className="slot-row">
              {levelSlots.map((slot) => (
                <article
                  className={`parking-slot ${slot.is_occupied ? 'is-occupied' : ''}`}
                  key={slot.id}
                >
                  <div className="slot-title">
                    <strong>#{slot.id}</strong>
                    <span>{slotSizeLabels[slot.size]}</span>
                  </div>

                  <p className="slot-status">
                    <span aria-hidden="true" />
                    {slot.is_occupied ? 'Occupied' : 'Available'}
                  </p>

                  <p className="vehicle-plate">
                    {slot.current_vehicle_id ?? '—'}
                  </p>

                  <dl className="distance-list">
                    {orderedGates.map((gate) => (
                      <div key={gate.id}>
                        <dt>G{gate.id}</dt>
                        <dd>{slot.distances[String(gate.id)] ?? '—'}</dd>
                      </div>
                    ))}
                  </dl>
                </article>
              ))}
            </div>
          </section>
        ))}
      </div>
    </section>
  )
}
