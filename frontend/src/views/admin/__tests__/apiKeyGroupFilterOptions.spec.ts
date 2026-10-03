import { describe, it, expect } from 'vitest'
import { buildApiKeyGroupFilterOptions } from '../apiKeyGroupFilterOptions'
import type { AdminGroup } from '@/types'

const labels = {
  all: 'All',
  exclusive: 'Exclusive',
  public: 'Public',
  subscription: 'Subscription',
  disabled: 'Disabled'
}

function group(partial: Partial<AdminGroup>): AdminGroup {
  return {
    id: 0,
    name: '',
    status: 'active',
    is_exclusive: false,
    subscription_type: 'standard',
    ...partial
  } as AdminGroup
}

describe('buildApiKeyGroupFilterOptions', () => {
  it('partitions active groups into exclusive/public/subscription sections', () => {
    const groups = [
      group({ id: 1, name: 'Exclusive', is_exclusive: true }),
      group({ id: 2, name: 'Public' }),
      group({ id: 3, name: 'Subscription', subscription_type: 'subscription' })
    ]

    expect(buildApiKeyGroupFilterOptions(groups, labels)).toEqual([
      { value: null, label: 'All' },
      { value: -1, label: 'Exclusive', kind: 'group', disabled: true },
      { value: 1, label: 'Exclusive' },
      { value: -2, label: 'Public', kind: 'group', disabled: true },
      { value: 2, label: 'Public' },
      { value: -3, label: 'Subscription', kind: 'group', disabled: true },
      { value: 3, label: 'Subscription' }
    ])
  })

  it('gives subscription groups precedence over exclusive groups', () => {
    const options = buildApiKeyGroupFilterOptions(
      [group({ id: 9, name: 'Shared', is_exclusive: true, subscription_type: 'subscription' })],
      labels
    )

    expect(options).toContainEqual({ value: 9, label: 'Shared' })
    expect(options.find((option) => option.label === 'Subscription')).toBeDefined()
    expect(options.find((option) => option.label === 'Exclusive')).toBeUndefined()
  })

  it('omits empty section headers', () => {
    const options = buildApiKeyGroupFilterOptions([group({ id: 2, name: 'Public' })], labels)

    expect(options.find((option) => option.label === 'Exclusive')).toBeUndefined()
    expect(options.find((option) => option.label === 'Subscription')).toBeUndefined()
    expect(options).toContainEqual({ value: -2, label: 'Public', kind: 'group', disabled: true })
  })

  it('keeps inactive groups in a separate disabled section', () => {
    const options = buildApiKeyGroupFilterOptions([
      group({ id: 1, name: 'Active', is_exclusive: true }),
      group({ id: 2, name: 'Inactive', status: 'inactive', is_exclusive: true })
    ], labels)

    expect(options).toContainEqual({ value: 1, label: 'Active' })
    expect(options).toContainEqual({ value: 2, label: 'Inactive' })
    expect(options).toContainEqual({ value: -4, label: 'Disabled', kind: 'group', disabled: true })
    expect(options.findIndex((option) => option.value === -1)).toBeLessThan(
      options.findIndex((option) => option.value === 2)
    )
  })

  it('uses distinct negative values for section headers', () => {
    const options = buildApiKeyGroupFilterOptions([
      group({ id: 1, is_exclusive: true }),
      group({ id: 2 }),
      group({ id: 3, subscription_type: 'subscription' }),
      group({ id: 4, status: 'inactive' })
    ], labels)
    const headers = options.filter((option) => option.kind === 'group').map((option) => option.value)

    expect(new Set(headers).size).toBe(headers.length)
    headers.forEach((value) => expect(value).toBeLessThan(0))
  })

  it('returns only the all option when no groups exist', () => {
    expect(buildApiKeyGroupFilterOptions([], labels)).toEqual([{ value: null, label: 'All' }])
  })
})
