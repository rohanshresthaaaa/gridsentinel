import type { Alarm, PhaseReading, Reading } from '../types'
import { alarmLabel } from '../format'

interface Props {
  reading: Reading
  alarms: Alarm[] // active alarms for this device only
}

const PHASES = ['A', 'B', 'C'] as const

export function DeviceCard({ reading, alarms }: Props) {
  const inAlarm = alarms.length > 0

  const phases: Record<(typeof PHASES)[number], PhaseReading> = {
    A: reading.phaseA,
    B: reading.phaseB,
    C: reading.phaseC,
  }

  // Highlight exactly the values that caused an alarm.
  const phaseFlagged = (phase: string) => alarms.some((a) => a.phase === phase)
  const frequencyFlagged = alarms.some((a) => a.element.startsWith('81'))
  const imbalanceFlagged = alarms.some((a) => a.element === '47')

  return (
    <article className={`card ${inAlarm ? 'card--alarm' : ''}`}>
      <header className="card__header">
        <h2>{reading.deviceId}</h2>
        <span className={`badge ${inAlarm ? 'badge--alarm' : 'badge--ok'}`}>
          {inAlarm ? 'ALARM' : 'NORMAL'}
        </span>
      </header>

      <table>
        <thead>
          <tr>
            <th>Phase</th>
            <th>Voltage</th>
            <th>Current</th>
            <th>Angle</th>
          </tr>
        </thead>
        <tbody>
          {PHASES.map((p) => (
            <tr key={p} className={phaseFlagged(p) ? 'flagged' : undefined}>
              <td>{p}</td>
              <td>{phases[p].voltageV.toFixed(1)} V</td>
              <td>{phases[p].currentA.toFixed(1)} A</td>
              <td>{phases[p].angleDeg.toFixed(0)}°</td>
            </tr>
          ))}
        </tbody>
      </table>

      <dl className="metrics">
        <div className={frequencyFlagged ? 'flagged' : undefined}>
          <dt>Frequency</dt>
          <dd>{reading.frequencyHz.toFixed(3)} Hz</dd>
        </div>
        <div className={imbalanceFlagged ? 'flagged' : undefined}>
          <dt>Imbalance</dt>
          <dd>{reading.imbalancePercent.toFixed(2)} %</dd>
        </div>
      </dl>

      {inAlarm && (
        <ul className="card__alarms">
          {alarms.map((a) => (
            <li key={a.id}>{alarmLabel(a)}</li>
          ))}
        </ul>
      )}
    </article>
  )
}