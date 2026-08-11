import { useCallback, useState } from 'react'
import { addGate, removeGate } from '../api/parking'
import type { Gate, ParkingSlot } from '../types'

interface GateManagerProps {
  gates: Gate[]
  slots: ParkingSlot[]
  onMutate: () => void
}

export function GateManager({
  gates,
  slots,
  onMutate,
}: GateManagerProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [newGateName, setNewGateName] = useState('')
  const [distances, setDistances] = useState<Record<number, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string>()
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  const sortedSlots = [...slots].sort((a, b) => a.id - b.id)

  const handleAddGate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newGateName.trim()) return

    const distanceList = sortedSlots
      .map((slot) => ({
        slot_id: slot.id,
        distance: parseInt(distances[slot.id] || '0', 10),
      }))
      .filter((d) => d.distance > 0)

    if (distanceList.length === 0) {
      setError('All distances must be positive')
      return
    }

    setIsSubmitting(true)
    setError(undefined)

    try {
      await addGate({
        name: newGateName.trim(),
        distances: distanceList,
      })
      setNewGateName('')
      setDistances({})
      setIsExpanded(false)
      onMutate()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add gate')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDeleteGate = useCallback(
    async (gateId: number) => {
      setIsDeleting(true)
      setError(undefined)
      try {
        await removeGate(gateId)
        setDeleteConfirm(null)
        onMutate()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to remove gate')
      } finally {
        setIsDeleting(false)
      }
    },
    [onMutate],
  )

  return (
    <section className="gate-section" aria-labelledby="gate-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Configuration / 04</p>
          <h2 id="gate-heading">Gates</h2>
        </div>
      </div>

      <div className="gate-list">
        {gates.length === 0 ? (
          <p className="empty-state">No gates configured</p>
        ) : (
          gates
            .sort((a, b) => a.id - b.id)
            .map((gate) => (
              <div key={gate.id} className="gate-row">
                <div className="gate-info">
                  <span className="gate-id">G{gate.id}</span>
                  <span className="gate-name">{gate.name}</span>
                </div>
                <button
                  type="button"
                  className="gate-remove"
                  onClick={() => setDeleteConfirm(gate.id)}
                  aria-label={`Remove ${gate.name}`}
                >
                  ✕
                </button>
              </div>
            ))
        )}
      </div>

      {deleteConfirm !== null && (
        <div className="delete-confirm">
          <p>Remove this gate and update all slot distances?</p>
          <div className="confirm-actions">
            <button
              type="button"
              className="confirm-cancel"
              onClick={() => setDeleteConfirm(null)}
              disabled={isDeleting}
            >
              Cancel
            </button>
            <button
              type="button"
              className="confirm-delete"
              onClick={() => void handleDeleteGate(deleteConfirm)}
              disabled={isDeleting}
            >
              {isDeleting ? '...' : 'Remove'}
            </button>
          </div>
        </div>
      )}

      <button
        type="button"
        className={`expand-btn ${isExpanded ? 'is-active' : ''}`}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isExpanded ? 'Cancel' : '+ Add Gate'}
      </button>

      {isExpanded && (
        <form className="add-gate-form" onSubmit={(e) => void handleAddGate(e)}>
          <div className="form-row">
            <label htmlFor="gate-name">Gate Name</label>
            <input
              id="gate-name"
              type="text"
              placeholder="Gate D"
              value={newGateName}
              onChange={(e) => setNewGateName(e.target.value)}
              disabled={isSubmitting}
              required
            />
          </div>

          <div className="distances-grid">
            <span className="grid-label">Distances to all slots</span>
            {sortedSlots.map((slot) => (
              <div key={slot.id} className="distance-input">
                <label htmlFor={`dist-${slot.id}`}>
                  Slot #{slot.id}
                </label>
                <input
                  id={`dist-${slot.id}`}
                  type="number"
                  min="1"
                  value={distances[slot.id] || ''}
                  onChange={(e) =>
                    setDistances({ ...distances, [slot.id]: e.target.value })
                  }
                  disabled={isSubmitting}
                  required
                />
              </div>
            ))}
          </div>

          {error && <span className="form-error">{error}</span>}

          <button type="submit" className="submit-btn" disabled={isSubmitting}>
            {isSubmitting ? 'Creating...' : 'Create Gate'}
          </button>
        </form>
      )}
    </section>
  )
}
