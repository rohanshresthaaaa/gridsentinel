import type { Alarm } from '../types'
import { alarmLabel, formatTime } from '../format'

interface Props {
  title: string
  alarms: Alarm[]
  emptyText: string
}

export function AlarmTable({ title, alarms, emptyText }: Props) {
  return (
    <section className="panel">
      <h2>
        {title} <span className="count">({alarms.length})</span>
      </h2>

      {alarms.length === 0 ? (
        <p className="empty">{emptyText}</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Device</th>
                <th>Element</th>
                <th>Value</th>
                <th>Setting</th>
                <th>Picked up</th>
                <th>Dropped out</th>
              </tr>
            </thead>
            <tbody>
              {alarms.map((a) => (
                <tr key={a.id}>
                  <td>
                    <span className={`badge ${a.active ? 'badge--alarm' : 'badge--muted'}`}>
                      {a.active ? 'ACTIVE' : 'CLEARED'}
                    </span>
                  </td>
                  <td>{a.deviceId}</td>
                  <td>{alarmLabel(a)}</td>
                  <td>
                    {a.value.toFixed(2)} {a.unit}
                  </td>
                  <td>
                    {a.threshold.toFixed(2)} {a.unit}
                  </td>
                  <td>{formatTime(a.pickedUpAt)}</td>
                  <td>{formatTime(a.droppedOutAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}