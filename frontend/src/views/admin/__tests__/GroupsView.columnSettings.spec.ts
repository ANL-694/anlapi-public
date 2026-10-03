import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import GroupsView from '../GroupsView.vue'

const { listGroups, getAllGroups, getUsageSummary, getCapacitySummary, getLiveCapability, getModelsListCandidates, listAccounts } = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getAllGroups: vi.fn(),
  getUsageSummary: vi.fn(),
  getCapacitySummary: vi.fn(),
  getLiveCapability: vi.fn(),
  getModelsListCandidates: vi.fn(),
  listAccounts: vi.fn(),
}))

const authState = { isSimpleMode: false }

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: listGroups,
      getAll: getAllGroups,
      getUsageSummary,
      getCapacitySummary,
      getLiveCapability,
      getModelsListCandidates,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      duplicate: vi.fn(),
      updateSortOrder: vi.fn(),
    },
    accounts: { list: listAccounts, getById: vi.fn() },
  },
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))
vi.mock('@/stores/onboarding', () => ({ useOnboardingStore: () => ({ isCurrentStep: vi.fn(() => false), nextStep: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const group = {
  id: 1,
  name: 'Core',
  platform: 'anthropic',
  rate_multiplier: 1,
  is_exclusive: false,
  status: 'active',
  account_count: 1,
} as any

const mountView = async () => {
  const wrapper = mount(GroupsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: {
          props: ['columns', 'data'],
          template: '<div data-test="columns">{{ columns.map((column) => column.key).join(",") }}</div>',
        },
        Select: true,
        Pagination: true,
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        ConfirmDialog: true,
        EmptyState: true,
        PlatformIcon: true,
        Icon: true,
        UiIconButton: { props: ['label'], emits: ['click'], template: '<button :title="label" @click="$emit(\'click\')"><slot /></button>' },
        GroupRateMultipliersModal: true,
        GroupRateScheduleModal: true,
        GroupRPMOverridesModal: true,
        CompositeRoutesModal: true,
        ReasoningEffortPolicyFields: true,
        VueDraggable: { template: '<div><slot /></div>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('GroupsView column settings', () => {
  beforeEach(() => {
    localStorage.clear()
    authState.isSimpleMode = false
    listGroups.mockReset().mockResolvedValue({ items: [group], total: 1, page: 1, page_size: 20, pages: 1 })
    getAllGroups.mockReset().mockResolvedValue([])
    getUsageSummary.mockReset().mockResolvedValue([])
    getCapacitySummary.mockReset().mockResolvedValue([])
    getLiveCapability.mockReset().mockResolvedValue({ supported: false })
    getModelsListCandidates.mockReset().mockResolvedValue([])
    listAccounts.mockReset().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
  })

  afterEach(() => localStorage.clear())

  it('hides id by default and records the current settings version', async () => {
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="columns"]').text()).toBe(
      'name,platform,billing_type,rate_multiplier,is_exclusive,account_count,usage,status,actions',
    )
    expect(localStorage.getItem('group-hidden-columns')).toBe('["id"]')
    expect(localStorage.getItem('group-column-settings-version')).toBe('2')
  })

  it('filters invalid saved keys and migrates older preferences', async () => {
    localStorage.setItem('group-hidden-columns', '["usage","removed","name","actions"]')

    const wrapper = await mountView()

    expect(wrapper.get('[data-test="columns"]').text()).toBe(
      'name,platform,billing_type,rate_multiplier,is_exclusive,account_count,status,actions',
    )
    expect(JSON.parse(localStorage.getItem('group-hidden-columns') || '[]')).toEqual(
      expect.arrayContaining(['usage', 'id']),
    )
    expect(localStorage.getItem('group-column-settings-version')).toBe('2')
  })

  it('persists a column toggle from the settings menu', async () => {
    const wrapper = await mountView()
    await wrapper.get('button[title="admin.groups.columnSettings"]').trigger('click')
    const usageToggle = wrapper.findAll('button').find((button) => button.text().includes('usage'))
    expect(usageToggle).toBeTruthy()
    await usageToggle!.trigger('click')

    expect(JSON.parse(localStorage.getItem('group-hidden-columns') || '[]')).toEqual(
      expect.arrayContaining(['id', 'usage']),
    )
  })
})
