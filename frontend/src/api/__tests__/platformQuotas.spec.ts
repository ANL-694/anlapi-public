import { describe, expect, it } from 'vitest'
import {
  createEmptyPlatformQuotaLimitSettings,
  hasInvalidPlatformQuotaLimitSettings
} from '@/api/platformQuotas'

describe('platform quota limit validation', () => {
  it('accepts empty, zero, and positive limits', () => {
    const settings = createEmptyPlatformQuotaLimitSettings()
    settings.anthropic.daily = 0
    settings.openai.weekly = 12.5
    expect(hasInvalidPlatformQuotaLimitSettings(settings)).toBe(false)
  })

  it.each([-1, Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY])(
    'rejects invalid limit %s',
    (value) => {
      const settings = createEmptyPlatformQuotaLimitSettings()
      settings.gemini.monthly = value
      expect(hasInvalidPlatformQuotaLimitSettings(settings)).toBe(true)
    }
  )
})
