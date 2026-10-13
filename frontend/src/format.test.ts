import { describe, expect, it } from 'vitest'
import { alarmLabel, alarmsForDevice, formatTime } from './format'
import { makeAlarm } from './test/fixtures'

describe('alarmLabel', () => {
  it('includes the phase when there is one', () => {
    expect(alarmLabel(makeAlarm())).toBe('27 Undervoltage · phase C')
  })

  it('omits the phase for system-wide elements', () => {
    const alarm = makeAlarm({ element: '81U', name: 'Underfrequency', phase: '' })
    expect(alarmLabel(alarm)).toBe('81U Underfrequency')
  })
})

describe('formatTime', () => {
  it('shows a dash when there is no time (alarm still active)', () => {
    expect(formatTime(null)).toBe('—')
  })
})

describe('alarmsForDevice', () => {
  it('returns only the alarms for the requested device', () => {
    const alarms = [
      makeAlarm({ id: 1, deviceId: 'RELAY-01' }),
      makeAlarm({ id: 2, deviceId: 'RELAY-02' }),
      makeAlarm({ id: 3, deviceId: 'RELAY-01' }),
    ]
    expect(alarmsForDevice(alarms, 'RELAY-01').map((a) => a.id)).toEqual([1, 3])
  })
})