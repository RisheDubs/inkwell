import { describe, expect, it } from 'vitest'
import { formatEntryDate, todayLocal } from './dates'

describe('formatEntryDate', () => {
  it('shows the calendar day the user picked, in their locale', () => {
    expect(formatEntryDate('2026-11-03', 'en-GB')).toBe('3 Nov 2026')
    expect(formatEntryDate('2026-11-03', 'en-US')).toBe('Nov 3, 2026')
  })

  it('never shifts the day, even for the first of the month', () => {
    expect(formatEntryDate('2026-12-01', 'en-GB')).toBe('1 Dec 2026')
  })

  it('shows an unexpected value as-is instead of "Invalid Date"', () => {
    expect(formatEntryDate('not-a-date', 'en-GB')).toBe('not-a-date')
  })
})

describe('todayLocal', () => {
  it("uses the user's own calendar day, even late at night", () => {
    // new Date(year, monthIndex, day, hour, minute) is local time; month 10 is November.
    expect(todayLocal(new Date(2026, 10, 3, 23, 30))).toBe('2026-11-03')
    expect(todayLocal(new Date(2026, 0, 5, 0, 15))).toBe('2026-01-05')
  })
})