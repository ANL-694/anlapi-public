import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MonitorDetailDialog from '../MonitorDetailDialog.vue'

const mocks = vi.hoisted(() => ({ status: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/channelMonitor', () => ({ status: mocks.status }))
vi.mock('@/stores/app', () => ({ useAppStore: () => mocks }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
beforeEach(() => vi.clearAllMocks())

function deferred() {
  let resolve!: (value: unknown) => void
  let reject!: (value: unknown) => void
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

const detail = (model: string) => ({ models: [{ model, latest_status: 'operational' }] })

function open() {
  return mount(MonitorDetailDialog, {
    props: { show: true, monitorId: 1, title: 'Monitor' },
    global: { stubs: { BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /></div>' } } }
  })
}

describe('monitor detail request ownership', () => {
  it('keeps the new monitor response when the old response arrives last', async () => {
    const old = deferred()
    mocks.status.mockReturnValueOnce(old.promise).mockResolvedValueOnce(detail('new-model'))
    const wrapper = open()
    await wrapper.setProps({ monitorId: 2 })
    await flushPromises()
    old.resolve(detail('old-model'))
    await flushPromises()
    expect(wrapper.text()).toContain('new-model')
    expect(wrapper.text()).not.toContain('old-model')
  })

  it('ignores a previous failure while the current monitor is loading', async () => {
    const old = deferred()
    const current = deferred()
    mocks.status.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const wrapper = open()
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    old.reject(new Error('old failure'))
    await flushPromises()
    expect(mocks.showError).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('common.loading')
    current.resolve(detail('current'))
    await flushPromises()
    expect(wrapper.text()).toContain('current')
  })

  it('reports a failure for the current monitor', async () => {
    mocks.status.mockRejectedValueOnce(new Error('current failure'))
    const wrapper = open()
    await flushPromises()
    expect(mocks.showError).toHaveBeenCalledWith('current failure')
    expect(wrapper.text()).toContain('channelStatus.detailLoadError')
  })
})
