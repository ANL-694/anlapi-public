import { describe, expect, it } from 'vitest'
import { getKeyGroupProvider, KEY_GROUP_PROVIDERS } from '@/utils/keyGroupProviders'

describe('key group provider classification', () => {
  it('keeps the provider choices aligned with the current platform contract', () => {
    expect(KEY_GROUP_PROVIDERS).toEqual(['anthropic', 'openai', 'other'])
    expect(getKeyGroupProvider('anthropic')).toBe('anthropic')
    expect(getKeyGroupProvider('openai')).toBe('openai')
  })

  it.each(['gemini', 'antigravity', 'grok', 'kiro', 'custom', 'composite'] as const)(
    'classifies %s as other',
    (platform) => {
      expect(getKeyGroupProvider(platform)).toBe('other')
    }
  )
})
