import { describe, expect, it } from 'vitest'
import { getAccountExpiryTimestamp } from '../accountExpiry'

describe('getAccountExpiryTimestamp', () => {
  it('adds one calendar month and clamps month-end dates', () => {
    const now = new Date('2026-01-31T12:34:56.000Z')
    const result = new Date(getAccountExpiryTimestamp(1, now) * 1000)
    expect(result.toISOString()).toBe('2026-02-28T12:34:56.000Z')
  })

  it('adds twelve calendar months without changing the time of day', () => {
    const now = new Date('2026-02-28T12:34:56.000Z')
    const result = new Date(getAccountExpiryTimestamp(12, now) * 1000)
    expect(result.toISOString()).toBe('2027-02-28T12:34:56.000Z')
  })
})
