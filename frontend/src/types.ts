// These types mirror the JSON produced by backend/internal/api/server.go.

export interface PhaseReading {
  voltageV: number
  currentA: number
  angleDeg: number
}

export interface Reading {
  deviceId: string
  timestamp: string // ISO 8601 date string
  phaseA: PhaseReading
  phaseB: PhaseReading
  phaseC: PhaseReading
  frequencyHz: number
  imbalancePercent: number
}

export interface Alarm {
  id: number
  deviceId: string
  element: string // ANSI device number, e.g. "27"
  name: string
  phase: string // "A", "B", "C", or "" when not phase-specific
  value: number
  threshold: number
  unit: string
  pickedUpAt: string
  droppedOutAt: string | null // null while the alarm is active
  active: boolean
}

export type ConnectionState = 'connecting' | 'live' | 'offline'