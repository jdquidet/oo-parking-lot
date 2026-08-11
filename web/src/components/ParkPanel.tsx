import { useState } from 'react'
import { parkVehicle, unparkVehicle } from '../api/parking'
import type { Gate, ParkingSession, ParkingSlot, VehicleSize } from '../types'
import { slotSizeLabels } from '../types'

interface ParkPanelProps {
  gates: Gate[]
  occupiedPlates: string[]
  onMutate: () => void
}

type Tab = 'park' | 'unpark'

export function ParkPanel({ gates, occupiedPlates, onMutate }: ParkPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('park')

  // Park State
  const [gateId, setGateId] = useState<string>(gates[0]?.id.toString() ?? '')
  const [size, setSize] = useState<VehicleSize>(0)
  const [parkPlate, setParkPlate] = useState('')

  // Unpark State
  const [unparkPlate, setUnparkPlate] = useState('')

  // Shared status state
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string>()

  // Receipts
  const [parkReceipt, setParkReceipt] = useState<{
    session: ParkingSession
    slot: ParkingSlot
  }>()
  const [unparkReceipt, setUnparkReceipt] = useState<{
    session: ParkingSession
    fee: number
  }>()

  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab)
    setError(undefined)
    setParkReceipt(undefined)
    setUnparkReceipt(undefined)
  }

  const handlePark = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!parkPlate.trim()) return

    setIsSubmitting(true)
    setError(undefined)
    setParkReceipt(undefined)
    setUnparkReceipt(undefined)

    try {
      const res = await parkVehicle({
        gate_id: parseInt(gateId, 10),
        license_plate: parkPlate.trim().toUpperCase(),
        vehicle_size: size,
      })
      setParkReceipt(res)
      setParkPlate('')
      onMutate()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to park')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleUnpark = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!unparkPlate.trim()) return

    setIsSubmitting(true)
    setError(undefined)
    setParkReceipt(undefined)
    setUnparkReceipt(undefined)

    try {
      const res = await unparkVehicle(unparkPlate.trim().toUpperCase())
      setUnparkReceipt(res)
      setUnparkPlate('')
      onMutate()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpark')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="panel-section" aria-labelledby="operations-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Operations / 02</p>
          <h2 id="operations-heading">Terminal</h2>
        </div>
      </div>

      <div className="terminal-tabs">
        <button
          type="button"
          className={activeTab === 'park' ? 'is-active' : ''}
          onClick={() => handleTabChange('park')}
        >
          Park
        </button>
        <button
          type="button"
          className={activeTab === 'unpark' ? 'is-active' : ''}
          onClick={() => handleTabChange('unpark')}
        >
          Unpark
        </button>
      </div>

      <div className="terminal-body">
        {activeTab === 'park' && (
          <form className="terminal-form" onSubmit={(e) => void handlePark(e)}>
            <div className="form-row">
              <label htmlFor="gate-select">Entry Gate</label>
              <select
                id="gate-select"
                value={gateId}
                onChange={(e) => setGateId(e.target.value)}
                disabled={isSubmitting}
              >
                {gates.map((g) => (
                  <option key={g.id} value={g.id}>
                    G{g.id} · {g.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="form-row">
              <label>Vehicle Size</label>
              <div className="size-toggles">
                {(Object.keys(slotSizeLabels) as Array<unknown> as VehicleSize[]).map(
                  (s) => (
                    <button
                      key={s}
                      type="button"
                      className={size === s ? 'is-active' : ''}
                      onClick={() => setSize(s)}
                      disabled={isSubmitting}
                    >
                      {slotSizeLabels[s].charAt(0)}
                    </button>
                  ),
                )}
              </div>
            </div>

            <div className="form-row">
              <label htmlFor="park-plate">License Plate</label>
              <input
                id="park-plate"
                type="text"
                placeholder="ABC-1234"
                value={parkPlate}
                onChange={(e) => setParkPlate(e.target.value)}
                disabled={isSubmitting}
                autoComplete="off"
                required
              />
            </div>

            <div className="form-actions">
              {error && <span className="terminal-error">{error}</span>}
              <button type="submit" className="submit-btn" disabled={isSubmitting}>
                {isSubmitting ? 'Processing...' : 'Issue Ticket'}
              </button>
            </div>
          </form>
        )}

        {activeTab === 'unpark' && (
          <form className="terminal-form" onSubmit={(e) => void handleUnpark(e)}>
            <div className="form-row">
              <label htmlFor="unpark-plate">License Plate</label>
              <input
                id="unpark-plate"
                type="text"
                list="active-plates"
                placeholder="ABC-1234"
                value={unparkPlate}
                onChange={(e) => setUnparkPlate(e.target.value)}
                disabled={isSubmitting}
                autoComplete="off"
                required
              />
              <datalist id="active-plates">
                {occupiedPlates.map((p) => (
                  <option key={p} value={p} />
                ))}
              </datalist>
            </div>

            <div className="form-actions">
              {error && <span className="terminal-error">{error}</span>}
              <button type="submit" className="submit-btn" disabled={isSubmitting}>
                {isSubmitting ? 'Processing...' : 'Process Exit'}
              </button>
            </div>
          </form>
        )}

        {/* Receipts */}
        {parkReceipt && (
          <div className="receipt-block">
            <div className="receipt-header">ENTRY TICKET</div>
            <dl>
              <dt>ID</dt>
              <dd>{parkReceipt.session.id}</dd>
              <dt>Plate</dt>
              <dd>{parkReceipt.session.vehicle_id}</dd>
              <dt>Slot</dt>
              <dd>
                #{parkReceipt.slot.id} ({slotSizeLabels[parkReceipt.slot.size]})
              </dd>
              <dt>Time</dt>
              <dd>
                {new Date(parkReceipt.session.entry_time).toLocaleString('en-CA')}
              </dd>
            </dl>
          </div>
        )}

        {unparkReceipt && (
          <div className="receipt-block is-exit">
            <div className="receipt-header">EXIT RECEIPT</div>
            <dl>
              <dt>ID</dt>
              <dd>{unparkReceipt.session.id}</dd>
              <dt>Plate</dt>
              <dd>{unparkReceipt.session.vehicle_id}</dd>
              <dt>Entry</dt>
              <dd>
                {new Date(unparkReceipt.session.entry_time).toLocaleString('en-CA')}
              </dd>
              <dt>Exit</dt>
              <dd>
                {unparkReceipt.session.exit_time
                  ? new Date(unparkReceipt.session.exit_time).toLocaleString(
                      'en-CA',
                    )
                  : '—'}
              </dd>
            </dl>
            <div className="receipt-fee">
              <span>Total Due</span>
              <strong>PHP {unparkReceipt.fee.toFixed(2)}</strong>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}
