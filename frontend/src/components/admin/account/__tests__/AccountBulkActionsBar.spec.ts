import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => `${key}${params ? ` ${JSON.stringify(params)}` : ''}`,
  }),
}))

describe('AccountBulkActionsBar', () => {
  it('allows selecting all results before any row is selected', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: { selectedIds: [], totalResults: 45 },
    })

    const button = wrapper.findAll('button').find(item => item.text().includes('selectAllResults'))
    expect(button).toBeDefined()
    await button!.trigger('click')
    expect(wrapper.emitted('select-all-results')).toHaveLength(1)
  })

  it('keeps the upstream billing probe action for selected accounts', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: { selectedIds: [1], totalResults: 45 },
    })

    const button = wrapper.findAll('button').find(item => item.text().includes('probeUpstreamBilling'))
    expect(button).toBeDefined()
    await button!.trigger('click')
    expect(wrapper.emitted('probe-upstream-billing')).toHaveLength(1)
  })
})
