import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SubscriptionsView from '../SubscriptionsView.vue'

const { listSubscriptions, assignSubscription, getAllGroups, listUsers, showError } = vi.hoisted(() => ({
  listSubscriptions: vi.fn(),
  assignSubscription: vi.fn(),
  getAllGroups: vi.fn(),
  listUsers: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    subscriptions: { list: listSubscriptions, assign: assignSubscription },
    groups: { getAll: getAllGroups },
    users: { list: listUsers },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess: vi.fn() }),
}))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: '<select @change="$emit(\'change\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
}

const DataTableStub = { props: ['data'], template: '<div />' }

const mountView = () => mount(SubscriptionsView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: { template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /></div>' },
      DataTable: DataTableStub,
      RouterLink: { template: '<a><slot /></a>' },
      Pagination: true,
      BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
      ConfirmDialog: true,
      EmptyState: true,
      Select: SelectStub,
      GroupBadge: true,
      GroupOptionItem: true,
      Icon: true,
      Teleport: true,
    },
  },
})

describe('SubscriptionsView assignment user search', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    vi.clearAllMocks()
    listSubscriptions.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getAllGroups.mockResolvedValue([])
    listUsers.mockResolvedValue({ items: [{ id: 42, email: 'reader@example.com' }], total: 1, pages: 1 })
    assignSubscription.mockResolvedValue({})
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it.each(['another', ''])('clears the assignment user immediately when input changes to %j', async (keyword) => {
    const wrapper = mountView()
    try {
      await flushPromises()
      await wrapper.findAll('button').find((button) => button.text() === 'admin.subscriptions.assignSubscription')!.trigger('click')
      const form = wrapper.get('#assign-subscription-form')
      const search = wrapper.get('[data-assign-user-search] input')
      const vm = wrapper.vm as any
      vm.assignForm.group_id = 3
      await search.trigger('focus')
      await search.setValue('reader')
      await vi.advanceTimersByTimeAsync(300)
      await flushPromises()
      await wrapper.get('[data-assign-user-search] button').trigger('click')

      await search.setValue(keyword)
      await form.trigger('submit')
      await flushPromises()

      expect(assignSubscription).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('admin.subscriptions.pleaseSelectUser')
      expect(listUsers).toHaveBeenCalledWith(1, 30, {
        search: 'reader',
        sort_by: 'email',
        sort_order: 'asc',
      })
    } finally {
      wrapper.unmount()
    }
  })
})
