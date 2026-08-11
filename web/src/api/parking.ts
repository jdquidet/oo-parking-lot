import type { ParkingSession, ParkingSlot, ParkingState, VehicleSize } from '../types'
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
