import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { daysUntil, proxyExpiryBadgeClass, proxyExpiryLabelKey } from '../proxyExpiry'

const now = new Date('2026-06-02T00:00:00Z')
const isoInDays = (days: number): string => new Date(now.getTime() + days * 86400000).toISOString()

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(now)
})

afterEach(() => vi.useRealTimers())

describe('proxy expiry helpers', () => {
  it('calculates whole days until expiry', () => {
    expect(daysUntil(isoInDays(10))).toBe(10)
    expect(daysUntil(isoInDays(-3))).toBe(-3)
  })

  it('uses danger, warning and neutral badge classes at their boundaries', () => {
    expect(proxyExpiryBadgeClass(isoInDays(30), 'expired')).toBe('badge badge-danger')
    expect(proxyExpiryBadgeClass(isoInDays(3), 'active')).toBe('badge badge-danger')
    expect(proxyExpiryBadgeClass(isoInDays(7), 'active')).toBe('badge badge-warning')
    expect(proxyExpiryBadgeClass(isoInDays(30), 'active')).toBe('text-gray-500')
  })

  it('returns translated label keys for overdue and future deadlines', () => {
    expect(proxyExpiryLabelKey(isoInDays(-3), 'active')).toEqual({
      key: 'admin.proxies.overdueDays',
      params: { days: 3 }
    })
    expect(proxyExpiryLabelKey(isoInDays(5), 'active')).toEqual({
      key: 'admin.proxies.expiringInDays',
      params: { days: 5 }
    })
    expect(proxyExpiryLabelKey(isoInDays(30), 'active')).toEqual({
      key: 'admin.proxies.remainingDays',
      params: { days: 30 }
    })
  })
})
