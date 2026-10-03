import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import KeysView from '../KeysView.vue'

const {
  list,
  update,
  getDashboardApiKeysUsage,
  getAvailable,
  getUserGroupRates,
  getPublicSettings
} = vi.hoisted(() => ({
  list: vi.fn(),
  update: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailable: vi.fn(),
  getUserGroupRates: vi.fn(),
  getPublicSettings: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/api', () => ({
  keysAPI: { list, update },
  authAPI: { getPublicSettings },
  usageAPI: { getDashboardApiKeysUsage },
  userGroupsAPI: { getAvailable, getUserGroupRates }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn()
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

const DataTableStub = {
  props: {
    columns: {
      type: Array,
      default: () => []
    },
    data: {
      type: Array,
      default: () => []
    },
    selectedKeys: {
      type: Array,
      default: () => []
    },
    selectable: Boolean,
    serverSideSort: Boolean
  },
  emits: ['update:selectedKeys', 'sort'],
  template: '<div><slot v-if="data.length" name="cell-actions" :row="data[0]" /></div>'
}

const ApiKeyTestModalStub = {
  name: 'ApiKeyTestModal',
  props: ['show', 'apiKey'],
  template: '<div data-test="api-key-test-modal" />'
}

const BulkEditKeysModalStub = {
  name: 'BulkEditKeysModal',
  props: ['show', 'selectedKeys', 'groups'],
  emits: ['close', 'updated'],
  template: '<div data-test="bulk-edit-modal" />'
}

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
}

const ConfirmDialogStub = {
  props: ['show', 'title'],
  emits: ['confirm', 'cancel'],
  template: `<div v-if="show" :data-title="title">
    <button class="confirm-dialog-confirm" @click="$emit('confirm')">confirm</button>
  </div>`,
}

const SelectStub = {
  props: ['options', 'modelValue'],
  emits: ['update:modelValue'],
  template: '<div data-test="select-stub"><select><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select></div>'
}

describe('KeysView model testing entry', () => {
  const apiKey = {
    id: 42,
    key: 'sk-plaintext-must-not-be-routed',
    name: 'Usage test key',
    status: 'active',
    is_system_managed: false
  }

  beforeEach(() => {
    list.mockReset()
    update.mockReset()
    getDashboardApiKeysUsage.mockReset()
    getAvailable.mockReset()
    getUserGroupRates.mockReset()
    getPublicSettings.mockReset()

    list.mockResolvedValue({ items: [apiKey], total: 1, pages: 1 })
    getDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    getAvailable.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    getPublicSettings.mockResolvedValue({ hide_ccs_import_button: true })
  })

  it('opens the API key model test modal for the selected key', async () => {
    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UiPage: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          EmptyState: true,
          Select: true,
          SearchInput: true,
          Icon: true,
          UseKeyModal: true,
          ApiKeyTestModal: ApiKeyTestModalStub,
          BulkEditKeysModal: BulkEditKeysModalStub,
          EndpointPopover: true,
          EndpointCards: true,
          GroupBadge: true,
          GroupOptionItem: true
        }
      }
    })

    try {
      await flushPromises()

      const testButton = wrapper
        .findAll('button.keys-action-button')
        .find((button) => button.text() === 'keys.testModel')

      expect(testButton).toBeDefined()

      await testButton!.trigger('click')

      const testModal = wrapper.getComponent(ApiKeyTestModalStub)
      expect(testModal.props('show')).toBe(true)
      expect(testModal.props('apiKey')).toMatchObject({ id: 42, name: 'Usage test key' })
      expect(wrapper.find('button[aria-label="keys.usage"]').exists()).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })

  it('opens bulk edit for selected visible keys and clears successful selections', async () => {
    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UiPage: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          EmptyState: true,
          Select: true,
          SearchInput: true,
          Icon: true,
          UseKeyModal: true,
          ApiKeyTestModal: ApiKeyTestModalStub,
          BulkEditKeysModal: BulkEditKeysModalStub,
          EndpointPopover: true,
          EndpointCards: true,
          GroupBadge: true,
          GroupOptionItem: true
        }
      }
    })

    try {
      await flushPromises()
      const table = wrapper.findComponent(DataTableStub)
      expect(table.props('selectable')).toBe(true)

      table.vm.$emit('update:selectedKeys', [42, 999])
      await flushPromises()

      const bulkButton = wrapper.get('[data-test="bulk-edit-keys"]')
      await bulkButton.trigger('click')
      const modal = wrapper.findComponent(BulkEditKeysModalStub)
      expect(modal.props('show')).toBe(true)
      expect(modal.props('selectedKeys')).toEqual([
        expect.objectContaining({ id: 42, name: 'Usage test key' })
      ])

      modal.vm.$emit('updated', [42])
      await flushPromises()
      expect(wrapper.find('[data-test="bulk-edit-keys"]').exists()).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })

  it('syncs quota and status from the reset response', async () => {
    const key = {
      ...apiKey,
      quota: 10,
      quota_used: 10,
      status: 'quota_exhausted',
    }
    list.mockResolvedValueOnce({ items: [key], total: 1, pages: 1 })
    update.mockResolvedValue({ ...key, quota_used: 0, status: 'active' })

    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UiPage: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          EmptyState: true,
          Select: true,
          SearchInput: true,
          Icon: true,
          UseKeyModal: true,
          ApiKeyTestModal: ApiKeyTestModalStub,
          BulkEditKeysModal: BulkEditKeysModalStub,
          EndpointPopover: true,
          EndpointCards: true,
          GroupBadge: true,
          GroupOptionItem: true,
        },
      },
    })

    try {
      await flushPromises()
      const editButton = wrapper.findAll('button.keys-action-button').find((button) => button.text() === 'common.edit')
      await editButton!.trigger('click')
      await wrapper.get('button[title="keys.resetQuotaUsed"]').trigger('click')
      await wrapper.get('[data-title="keys.resetQuotaTitle"] .confirm-dialog-confirm').trigger('click')
      await flushPromises()

      expect(update).toHaveBeenCalledWith(42, { reset_quota: true })
      const vm = wrapper.vm as any
      expect(vm.selectedKey).toMatchObject({ quota_used: 0, status: 'active' })
      expect(vm.formData.status).toBe('active')
    } finally {
      wrapper.unmount()
    }
  })

  it('filters create-key groups by the selected provider while keeping edit mode unrestricted', async () => {
    getAvailable.mockResolvedValue([
      { id: 1, name: 'Claude group', platform: 'anthropic', description: null, rate_multiplier: 1, subscription_type: 'standard', scope: 'public' },
      { id: 2, name: 'GPT group', platform: 'openai', description: null, rate_multiplier: 1, subscription_type: 'standard', scope: 'public' },
      { id: 3, name: 'Gemini group', platform: 'gemini', description: null, rate_multiplier: 1, subscription_type: 'standard', scope: 'public' },
    ])

    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UiPage: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          EmptyState: true,
          Select: SelectStub,
          SearchInput: true,
          Icon: true,
          PlatformIcon: true,
          UseKeyModal: true,
          ApiKeyTestModal: ApiKeyTestModalStub,
          BulkEditKeysModal: BulkEditKeysModalStub,
          EndpointPopover: true,
          EndpointCards: true,
          GroupBadge: true,
          GroupOptionItem: true
        }
      }
    })

    try {
      await flushPromises()
      await wrapper.get('[data-tour="keys-create-btn"]').trigger('click')
      await flushPromises()

      const providerInputs = wrapper.findAll('input[name="key-provider"]')
      expect(providerInputs).toHaveLength(3)

      const selects = wrapper.findAllComponents(SelectStub)
      const createGroupSelect = selects[selects.length - 1]
      expect(createGroupSelect.props('options')).toEqual([
        expect.objectContaining({ label: 'Claude group', platform: 'anthropic' })
      ])

      await wrapper.get('input[name="key-provider"][value="openai"]').setValue(true)
      await flushPromises()
      expect(wrapper.findAllComponents(SelectStub).at(-1)!.props('options')).toEqual([
        expect.objectContaining({ label: 'GPT group', platform: 'openai' })
      ])

      const cancelButton = wrapper.findAll('button').find((button) => button.text() === 'common.cancel')
      await cancelButton!.trigger('click')
      const editButton = wrapper.findAll('button.keys-action-button').find((button) => button.text() === 'common.edit')
      await editButton!.trigger('click')
      await flushPromises()
      expect(wrapper.find('input[name="key-provider"]').exists()).toBe(false)
      const editGroupSelect = wrapper.findAllComponents(SelectStub).find((select) => (
        select.props('options') as Array<{ platform?: string }>
      ).some((option) => option.platform === 'openai'))
      expect(editGroupSelect).toBeDefined()
      expect(editGroupSelect!.props('options')).toHaveLength(3)
    } finally {
      wrapper.unmount()
    }
  })

  it('requests server-side sorting when the group column is clicked', async () => {
    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UiPage: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          EmptyState: true,
          Select: true,
          SearchInput: true,
          Icon: true,
          UseKeyModal: true,
          ApiKeyTestModal: ApiKeyTestModalStub,
          BulkEditKeysModal: BulkEditKeysModalStub,
          EndpointPopover: true,
          EndpointCards: true,
          GroupBadge: true,
          GroupOptionItem: true
        }
      }
    })

    try {
      await flushPromises()
      list.mockClear()
      wrapper.findComponent(DataTableStub).vm.$emit('sort', 'group', 'asc')
      await flushPromises()
      expect(list).toHaveBeenCalledWith(
        1,
        20,
        expect.objectContaining({ sort_by: 'group', sort_order: 'asc' }),
        expect.objectContaining({ signal: expect.any(AbortSignal) })
      )
    } finally {
      wrapper.unmount()
    }
  })
})
