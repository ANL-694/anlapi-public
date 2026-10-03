import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RedeemView from '../RedeemView.vue'

const { redeem, getHistory, getPublicSettings, refreshUser, fetchActiveSubscriptions, showError, showWarning, showSuccess } = vi.hoisted(() => ({
  redeem: vi.fn(),
  getHistory: vi.fn(),
  getPublicSettings: vi.fn(),
  refreshUser: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api', () => ({
  redeemAPI: { redeem, getHistory },
  authAPI: { getPublicSettings },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { balance: 10, concurrency: 2, points_balance: 0 },
    refreshUser,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({ fetchActiveSubscriptions }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showWarning, showSuccess }),
}))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))

const mountView = () => mount(RedeemView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      UiPage: { template: '<div><slot /></div>' },
      UiMetricStrip: { template: '<div><slot /></div>' },
      UiMetric: true,
      Icon: true,
      Pagination: {
        props: ['page', 'total', 'pageSize'],
        template: `
          <div data-testid="history-pagination">
            <button class="previous" :disabled="page <= 1" @click="$emit('update:page', page - 1)">previous</button>
            <button class="next" :disabled="page * pageSize >= total" @click="$emit('update:page', page + 1)">next</button>
            <button class="size-50" @click="$emit('update:page-size', 50)">size 50</button>
          </div>
        `,
      },
    },
  },
})

const submitCode = async () => {
  const wrapper = mountView()
  await flushPromises()
  await wrapper.get('input#code').setValue(' REDEEM-CODE ')
  await wrapper.get('form').trigger('submit')
  await flushPromises()
  return wrapper
}

describe('RedeemView account refresh handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    getPublicSettings.mockResolvedValue({ contact_info: '' })
    fetchActiveSubscriptions.mockResolvedValue(undefined)
    redeem.mockResolvedValue({ type: 'balance', value: 5, new_balance: 15, message: 'ok' })
    refreshUser.mockResolvedValue(undefined)
  })

  it('keeps the redemption successful when account refresh fails', async () => {
    refreshUser.mockRejectedValue(new Error('refresh unavailable'))

    const wrapper = await submitCode()

    expect(redeem).toHaveBeenCalledWith('REDEEM-CODE')
    expect(showWarning).toHaveBeenCalledWith('redeem.userRefreshFailed')
    expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
    expect(showError).not.toHaveBeenCalled()
    expect(wrapper.find('.border-emerald-200').exists()).toBe(true)
    expect(wrapper.get('input#code').element.value).toBe('')
    wrapper.unmount()
  })

  it('still reports a redeem request failure as a failed redemption', async () => {
    redeem.mockRejectedValue({ response: { data: { detail: 'invalid code' } } })

    const wrapper = await submitCode()

    expect(showError).toHaveBeenCalledWith('redeem.redeemFailed')
    expect(showWarning).not.toHaveBeenCalled()
    expect(showSuccess).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('invalid code')
    expect(wrapper.get('input#code').element.value).toBe(' REDEEM-CODE ')
    wrapper.unmount()
  })

  it('loads redemption history pages from the server and resets page on size changes', async () => {
    getHistory
      .mockResolvedValueOnce({
        items: [{ id: 1, code: 'PAGE-ONE', type: 'balance', value: 5, used_at: '2026-03-08T00:00:00Z' }],
        total: 41,
        page: 1,
        page_size: 20,
        pages: 3,
      })
      .mockResolvedValueOnce({ items: [], total: 41, page: 2, page_size: 20, pages: 3 })
      .mockResolvedValueOnce({ items: [], total: 41, page: 1, page_size: 50, pages: 1 })

    const wrapper = mountView()
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 20)
    expect(wrapper.text()).toContain('PAGE-ONE')

    await wrapper.get('[data-testid="history-pagination"] .next').trigger('click')
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(2, 20)

    await wrapper.get('[data-testid="history-pagination"] .size-50').trigger('click')
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 50)
    wrapper.unmount()
  })

  it('keeps loaded history usable when a pagination request fails', async () => {
    getHistory
      .mockResolvedValueOnce({ items: [{ id: 1, code: 'STABLE-ROW', type: 'balance', value: 5, used_at: '2026-03-08T00:00:00Z' }], total: 41, page: 1, page_size: 20, pages: 3 })
      .mockRejectedValueOnce(new Error('network unavailable'))

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="history-pagination"] .size-50').trigger('click')
    await flushPromises()

    expect(wrapper.find('p.font-mono').text()).toBe('STABLE-R...')
    expect(showError).toHaveBeenCalledWith('redeem.historyLoadFailed')
    wrapper.unmount()
  })
})
