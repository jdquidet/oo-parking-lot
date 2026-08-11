import type { ParkingState } from '../types'
import { request } from './client'

export function getParkingState(signal?: AbortSignal): Promise<ParkingState> {
  return request<ParkingState>('/api/state', { signal })
}
