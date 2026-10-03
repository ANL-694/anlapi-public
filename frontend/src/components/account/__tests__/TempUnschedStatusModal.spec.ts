import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import TempUnschedStatusModal from '../TempUnschedStatusModal.vue'
import type { Account } from '@/types'

const mocks = vi.hoisted(() => ({
  getTempUnschedulableStatus: vi.fn(),
  recoverState: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { accounts: mocks } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => mocks }))
vi.mock('@/utils/format', () => ({ formatDateTime: () => 'date' }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

enableAutoUnmount(afterEach)
beforeEach(() => vi.clearAllMocks())

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (value: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

const active = (message: string) => ({
  active: true,
  state: { until_unix: Date.now() / 1000 + 3600, error_message: message, rule_index: -1 }
})

async function open() {
  const wrapper = mount(TempUnschedStatusModal, {
    props: { show: false, account: { id: 1, name: 'first' } as Account },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        }
      }
    }
  })
  await wrapper.setProps({ show: true })
  return wrapper
}

describe('temporary unschedulable status requests', () => {
  it('does not let a late response replace the next account status', async () => {
    const old = deferred<ReturnType<typeof active>>()
    mocks.getTempUnschedulableStatus
      .mockReturnValueOnce(old.promise)
      .mockResolvedValueOnce(active('current-error'))
    const wrapper = await open()
    await wrapper.setProps({ account: { id: 2, name: 'second' } as Account })
    await flushPromises()

    old.resolve(active('old-error'))
    await flushPromises()
    expect(wrapper.text()).toContain('current-error')
    expect(wrapper.text()).not.toContain('old-error')
  })

  it('keeps recovery disabled while loading a different account', async () => {
    const current = deferred<ReturnType<typeof active>>()
    mocks.getTempUnschedulableStatus
      .mockResolvedValueOnce(active('old-error'))
      .mockReturnValueOnce(current.promise)
    const wrapper = await open()
    await flushPromises()

    await wrapper.setProps({ account: { id: 2 } as Account })
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()

    current.resolve(active('current-error'))
    await flushPromises()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeUndefined()
  })

  it('ignores old errors without dismissing the current loading state', async () => {
    const old = deferred<ReturnType<typeof active>>()
    const current = deferred<ReturnType<typeof active>>()
    mocks.getTempUnschedulableStatus
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise)
    const wrapper = await open()
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })

    old.reject(new Error('obsolete'))
    await flushPromises()
    expect(mocks.showError).not.toHaveBeenCalled()
    expect(wrapper.find('.animate-spin').exists()).toBe(true)

    current.resolve({ active: false })
    await flushPromises()
    expect(wrapper.text()).toContain('admin.accounts.tempUnschedulable.notActive')
  })
})
