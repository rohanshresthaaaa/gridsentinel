import './App.css'
import { AlarmTable } from './components/AlarmTable'
import { DeviceCard } from './components/DeviceCard'
import { alarmsForDevice } from './format'
import { useGridData } from './useGridData'

const CONNECTION_TEXT = {
  connecting: 'Connecting…',
  live: 'Live',
  offline: 'Offline, retrying…',
} as const

function App() {
  const { devices, activeAlarms, history, connection } = useGridData()

  return (
    <div className="app">
      <header className="topbar">
        <div>
          <h1>GridSentinel</h1>
          <p className="subtitle">Substation protection monitor</p>
        </div>
        <span className={`connection connection--${connection}`}>
          ● {CONNECTION_TEXT[connection]}
        </span>
      </header>

      <main className="content">
        <section className="device-grid">
          {devices.length === 0 ? (
            <p className="empty">Waiting for relay data…</p>
          ) : (
            devices.map((d) => (
              <DeviceCard
                key={d.deviceId}
                reading={d}
                alarms={alarmsForDevice(activeAlarms, d.deviceId)}
              />
            ))
          )}
        </section>

        <AlarmTable
          title="Active alarms"
          alarms={activeAlarms}
          emptyText="No active alarms. All relays are normal."
        />
        <AlarmTable
          title="Recent events"
          alarms={history}
          emptyText="No events recorded yet."
        />
      </main>
    </div>
  )
}

export default App