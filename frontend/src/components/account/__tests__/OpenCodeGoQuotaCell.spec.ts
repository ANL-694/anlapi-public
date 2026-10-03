import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpenCodeGoQuotaCell from '../OpenCodeGoQuotaCell.vue'
import type { Account } from '@/types'

const { queryOpenCodeGoQuota } = vi.hoisted(() => ({
  queryOpenCodeGoQuota: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      queryOpenCodeGoQuota
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const makeAccount = (extra: Record<string, unknown> = {}): Account => ({
  id: 42,
  name: 'OpenCode Go',
  platform: 'opencode_go',
  account_level: 'platform',
  type: 'apikey',
  extra,
  proxy_id: null,
  concurrency: 1,
  priority: 1,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-09-25T00:00:00Z',
  updated_at: '2026-09-25T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null
})

const global = {
  stubs: {
    UsageProgressBar: {
      props: ['label', 'utilization', 'resetsAt', 'color'],
      template: '<div class="usage-bar">{{ label }}|{{ utilization }}</div>'
    }
  }
}

describe('OpenCodeGoQuotaCell', () => {
  beforeEach(() => queryOpenCodeGoQuota.mockReset())

  it('renders cached 5h, 7d and 30d windows without querying on mount', () => {
    const wrapper = mount(OpenCodeGoQuotaCell, {
      props: {
        account: makeAccount({
          opencode_go_5h_used_percent: 12,
          opencode_go_weekly_used_percent: 34,
          opencode_go_monthly_used_percent: 56
        })
      },
      global
    })

    expect(wrapper.findAll('.usage-bar').map((bar) => bar.text())).toEqual(['5h|12', '7d|34', '30d|56'])
    expect(queryOpenCodeGoQuota).not.toHaveBeenCalled()
  })

  it('refreshes the quota on demand and renders the sanitized response', async () => {
    queryOpenCodeGoQuota.mockResolvedValue({
      account_id: 42,
      plan: 'go',
      data: {
        rolling: { status: 'ok', percent: 21, resets_at: '2026-09-26T00:00:00Z' },
        weekly: { status: 'ok', percent: 43, resets_at: null },
        monthly: { status: 'ok', percent: 65, resets_at: null }
      },
      fetched_at: '2026-09-25T00:00:00Z',
      persisted: true,
      credential_ok: true
    })

    const wrapper = mount(OpenCodeGoQuotaCell, {
      props: { account: makeAccount() },
      global
    })

    await wrapper.get('[data-testid="opencode-go-quota-refresh"]').trigger('click')
    await flushPromises()

    expect(queryOpenCodeGoQuota).toHaveBeenCalledWith(42)
    expect(wrapper.findAll('.usage-bar').map((bar) => bar.text())).toEqual(['5h|21', '7d|43', '30d|65'])
  })
})
