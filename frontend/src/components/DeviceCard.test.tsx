import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { makeAlarm, makeReading } from '../test/fixtures'
import { DeviceCard } from './DeviceCard'

describe('DeviceCard', () => {
  it('shows NORMAL when the relay has no active alarms', () => {
    render(<DeviceCard reading={makeReading()} alarms={[]} />)

    expect(screen.getByText('RELAY-01')).toBeInTheDocument()
    expect(screen.getByText('NORMAL')).toBeInTheDocument()
    expect(screen.queryByText('ALARM')).not.toBeInTheDocument()
  })

  it('shows ALARM and lists the alarm when one is active', () => {
    render(<DeviceCard reading={makeReading()} alarms={[makeAlarm()]} />)

    expect(screen.getByText('ALARM')).toBeInTheDocument()
    expect(screen.getByText('27 Undervoltage · phase C')).toBeInTheDocument()
  })

  it('highlights only the faulted phase', () => {
    render(<DeviceCard reading={makeReading()} alarms={[makeAlarm({ phase: 'C' })]} />)

    expect(screen.getByText('C', { selector: 'td' }).closest('tr')).toHaveClass('flagged')
    expect(screen.getByText('A', { selector: 'td' }).closest('tr')).not.toHaveClass('flagged')
  })
})