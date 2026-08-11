export type VehicleSize = 0 | 1 | 2
export type SlotSize = 0 | 1 | 2

export interface Gate {
  id: number
  name: string
}

export interface ParkingSlot {
  id: number
  size: SlotSize
  distances: Record<string, number>
  is_occupied: boolean
  current_vehicle_id?: string
}

export interface ParkingSession {
  id: string
  vehicle_id: string
  vehicle_size: VehicleSize
  slot_id: number
  slot_size: SlotSize
  gate_id: number
  entry_time: string
  exit_time?: string
  total_fee_charged: number
  is_active: boolean
}

export interface ParkingState {
  gates: Gate[]
  slots: ParkingSlot[]
  systemTime: string
}

export const slotSizeLabels: Record<SlotSize, string> = {
  0: 'SP',
  1: 'MP',
  2: 'LP',
}
