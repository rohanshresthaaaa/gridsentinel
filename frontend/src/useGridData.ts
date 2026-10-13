import { useEffect, useMemo, useState } from 'react'
import type { Alarm, ConnectionState, Reading } from './types'

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) {
    throw new Error(`${path} failed with status ${res.status}`)
  }
  return res.json() as Promise<T>
}

// Plain function, no React state: just fetches both alarm lists.
function fetchAlarms(): Promise<[Alarm[], Alarm[]]> {
  return Promise.all([
    getJSON<Alarm[]>('/api/alarms/active'),
    getJSON<Alarm[]>('/api/alarms?limit=25'),
  ])
}

// useGridData loads the initial state from the REST API, then keeps it
// up to date from the live Server-Sent Events stream.
export function useGridData() {
  const [devices, setDevices] = useState<Record<string, Reading>>({})
  const [activeAlarms, setActiveAlarms] = useState<Alarm[]>([])
  const [history, setHistory] = useState<Alarm[]>([])
  const [connection, setConnection] = useState<ConnectionState>('connecting')

  useEffect(() => {
    // State is only set inside .then() callbacks, after the network responds.
    const loadAlarms = () => {
      fetchAlarms()
        .then(([active, recent]) => {
          setActiveAlarms(active)
          setHistory(recent)
        })
        .catch((err) => console.error('Could not load alarms:', err))
    }

    // 1. Initial snapshot.
    getJSON<Reading[]>('/api/devices')
      .then((list) => setDevices(Object.fromEntries(list.map((r) => [r.deviceId, r]))))
      .catch((err) => console.error('Could not load devices:', err))
    loadAlarms()

    // 2. Live updates. EventSource reconnects automatically if the connection drops.
    const source = new EventSource('/api/stream')
    source.onopen = () => setConnection('live')
    source.onerror = () => setConnection('offline')

    source.addEventListener('reading', (event) => {
      const reading: Reading = JSON.parse(event.data)
      setDevices((prev) => ({ ...prev, [reading.deviceId]: reading }))
    })

    // A pickup or dropout happened: reload alarms so both lists stay correct.
    source.addEventListener('alarm', loadAlarms)

    // 3. Cleanup: close the stream when the component goes away.
    return () => source.close()
  }, [])

  const deviceList = useMemo(
    () => Object.values(devices).sort((a, b) => a.deviceId.localeCompare(b.deviceId)),
    [devices],
  )

  return { devices: deviceList, activeAlarms, history, connection }
}