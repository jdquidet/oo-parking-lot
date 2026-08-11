import type { ParkingSession, ParkingSlot, ParkingState, VehicleSize, Gate } from '../types'
import { request } from './client'

export function getParkingState(signal?: AbortSignal): Promise<ParkingState> {
  return request<ParkingState>('/api/state', { signal })
}

export function parkVehicle(req: {
  gate_id: number
  license_plate: string
  vehicle_size: VehicleSize
}) {
  return request<{ session: ParkingSession; slot: ParkingSlot }>('/api/park', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export function unparkVehicle(license_plate: string) {
  return request<{ session: ParkingSession; fee: number }>('/api/unpark', {
    method: 'POST',
    body: JSON.stringify({ license_plate }),
  })
}

export function advanceTime(minutes: number) {
  return request<{ systemTime: string }>('/api/time/advance', {
    method: 'POST',
    body: JSON.stringify({ minutes }),
  })
}

export function getSessions(plate?: string, signal?: AbortSignal) {
  const url = new URL('/api/sessions', location.origin)
  if (plate) url.searchParams.set('plate', plate)
  return request<ParkingSession[]>(url.toString(), { signal })
}

export function addGate(req: {
  name: string
  distances: Array<{ slot_id: number; distance: number }>
}) {
  return request<Gate>('/api/gates', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export function removeGate(gateId: number) {
  return request<{ success: boolean }>(`/api/gates/${gateId}`, {
    method: 'DELETE',
  })
}
