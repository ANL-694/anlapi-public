import { describe, expect, it } from 'vitest'

import {
  formatReasoningEffort,
  formatReasoningEffortMapping,
  reasoningEffortValuesEqual,
} from '@/utils/format'

describe('formatReasoningEffort', () => {
  it('title-cases known effort values and handles empty values', () => {
    expect(formatReasoningEffort('max')).toBe('Max')
    expect(formatReasoningEffort('x-high')).toBe('XHigh')
    expect(formatReasoningEffort(null)).toBe('-')
  })
})

describe('formatReasoningEffortMapping', () => {
  it('shows one value when requested and forwarded values match', () => {
    expect(formatReasoningEffortMapping('max', 'max')).toBe('Max')
    expect(formatReasoningEffortMapping(null, 'high')).toBe('High')
  })

  it('shows requested then forwarded when a mapping changed the value', () => {
    expect(formatReasoningEffortMapping('max', 'xhigh')).toBe('Max → XHigh')
    expect(formatReasoningEffortMapping('high', 'medium')).toBe('High → Medium')
  })
})

describe('reasoningEffortValuesEqual', () => {
  it('treats x-high aliases as equal without equating unrelated values', () => {
    expect(reasoningEffortValuesEqual('x-high', 'xhigh')).toBe(true)
    expect(reasoningEffortValuesEqual('max', 'xhigh')).toBe(false)
  })
})
