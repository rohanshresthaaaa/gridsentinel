import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { makeAlarm } from '../test/fixtures'
import { AlarmTable } from './AlarmTable'

describe('AlarmTable', () => {
  it('shows the empty message and no table when there are no alarms', () => {
    render(<AlarmTable title="Active alarms" alarms={[]} emptyText="Nothing to show" />)

    expect(screen.getByText('Nothing to show')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('labels active and cleared alarms and shows the count', () => {
    const alarms = [
      makeAlarm({ id: 1 }),
      makeAlarm({
        id: 2,
        element: '50',
        name: 'Overcurrent',
        phase: 'A',
        active: false,
        droppedOutAt: '2026-10-12T23:00:05Z',
      }),
    ]
    render(<AlarmTable title="Recent events" alarms={alarms} emptyText="Nothing to show" />)

    expect(screen.getByText('ACTIVE')).toBeInTheDocument()
    expect(screen.getByText('CLEARED')).toBeInTheDocument()
    expect(screen.getByText('(2)')).toBeInTheDocument()
    expect(screen.getByText('50 Overcurrent · phase A')).toBeInTheDocument()
  })
})