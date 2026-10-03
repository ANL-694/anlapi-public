import { describe, expect, it } from 'vitest'
import { getExpirationDateRelation } from '../subscriptionQuota'

describe('subscription expiry calendar labels', () => {
  it('uses local calendar dates for today and tomorrow', () => {
    const now = new Date(2026, 2, 7, 23, 30)

    expect(getExpirationDateRelation(new Date(2026, 2, 7, 23, 45), now)).toBe('today')
    expect(getExpirationDateRelation(new Date(2026, 2, 8, 3, 30), now)).toBe('tomorrow')
  })

  it('treats the exact expiry instant and elapsed expiries as expired', () => {
    const now = new Date(2026, 6, 30, 9, 0)

    expect(getExpirationDateRelation(now, now)).toBe('expired')
    expect(getExpirationDateRelation(new Date(2026, 6, 30, 8, 59), now)).toBe('expired')
  })

  it('returns later for future dates beyond tomorrow and null for invalid dates', () => {
    const now = new Date(2026, 6, 30, 9, 0)

    expect(getExpirationDateRelation(new Date(2026, 6, 31, 9, 0), now)).toBe('tomorrow')
    expect(getExpirationDateRelation(new Date(2026, 7, 2, 9, 0), now)).toBe('later')
    expect(getExpirationDateRelation(new Date('invalid'), now)).toBeNull()
    expect(getExpirationDateRelation(now, new Date('invalid'))).toBeNull()
  })
})
