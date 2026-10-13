import type { Alarm, Reading } from '../types'

// Sample data for tests. Pass overrides to change only what a test cares about.

export function makeAlarm(overrides: Partial<Alarm> = {}): Alarm {
  return {
    id: 1,
    deviceId: 'RELAY-01',
    element: '27',
    name: 'Undervoltage',
    phase: 'C',
    value: 96,
    threshold: 105.6,
    unit: 'V',
    pickedUpAt: '2026-10-12T23:00:00Z',
    droppedOutAt: null,
    active: true,
    ...overrides,
  }
}

export function makeReading(overrides: Partial<Reading> = {}): Reading {
  const phase = (angleDeg: number) => ({ voltageV: 120, currentA: 50, angleDeg })
  return {
    deviceId: 'RELAY-01',
    timestamp: '2026-10-12T23:00:00Z',
    phaseA: phase(0),
    phaseB: phase(-120),
    phaseC: phase(120),
    frequencyHz: 60,
    imbalancePercent: 0.1,
    ...overrides,
  }
}