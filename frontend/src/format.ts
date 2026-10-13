import type { Alarm } from './types'

// The API sends UTC timestamps. toLocaleTimeString shows them in the
// viewer's local time zone, so 01:16 UTC appears as 7:16 PM in Idaho.
export function formatTime(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString()
}

export function alarmLabel(a: Pick<Alarm, 'element' | 'name' | 'phase'>): string {
  return a.phase ? `${a.element} ${a.name} · phase ${a.phase}` : `${a.element} ${a.name}`
}

export function alarmsForDevice(alarms: Alarm[], deviceId: string): Alarm[] {
  return alarms.filter((a) => a.deviceId === deviceId)
}