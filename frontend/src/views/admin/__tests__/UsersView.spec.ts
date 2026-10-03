import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UsersView from '../UsersView.vue'

const {
  listUsers,
  toggleStatus,
  deleteUser,
  showError,
  showSuccess,
  getAllGroups,
  getBatchUsersUsage,
  listEnabledDefinitions,
  getBatchUserAttributes
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  toggleStatus: vi.fn(),
  deleteUser: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  getAllGroups: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  listEnabledDefinitions: vi.fn(),
  getBatchUserAttributes: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      toggleStatus,
      delete: deleteUser
    },
    groups: {
      getAll: getAllGroups
    },
    dashboard: {
      getBatchUsersUsage
    },
    userAttributes: {
      listEnabledDefinitions,
      getBatchUserAttributes
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: { count?: number }) =>
        params?.count === undefined ? key : `${key}:${params.count}`
    })
  }
})

const createAdminUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  id: 42,
  username: 'scoped-user',
  email: 'scoped@example.com',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  last_active_at: '2026-04-16T02:00:00Z',
  last_used_at: '2026-04-17T02:00:00Z',
  current_concurrency: 0,
  ...overrides
})

const DataTableStub = {
  props: ['columns', 'data', 'selectedKeys'],
  emits: ['sort', 'update:selectedKeys'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map(col => col.key).join(',') }}</div>
      <button data-test="sort-last-used" @click="$emit('sort', 'last_used_at', 'desc')">sort</button>
      <div v-for="row in data" :key="row.id">
        <div :data-test="'row-state-' + row.id">{{ row.status }}:{{ row.updated_at }}</div>
        <slot name="cell-concurrency" :value="row.concurrency" :row="row" />
        <slot name="cell-last_used_at" :value="row.last_used_at" :row="row" />
        <div :data-test="'actions-' + row.id"><slot name="cell-actions" :row="row" /></div>
      </div>
    </div>
  `
}

const UserConcurrencyCellStub = {
  props: ['current', 'max'],
  template: '<span data-test="user-concurrency">{{ current }} / {{ max }}</span>'
}

const BulkEditUserModalStub = {
  props: ['show', 'selectedIds'],
  emits: ['close', 'success'],
  template: `
    <div v-if="show" data-test="bulk-modal">
      <span data-test="bulk-modal-ids">{{ selectedIds.join(',') }}</span>
      <button data-test="bulk-success" @click="$emit('success', selectedIds.length)">success</button>
    </div>
  `
}

const ConfirmDialogStub = {
  props: ['show', 'message'],
  emits: ['confirm', 'cancel'],
  template: `<div v-if="show" data-test="delete-dialog">
    <span>{{ message }}</span>
    <button data-test="confirm-delete" @click="$emit('confirm')">confirm</button>
    <button data-test="cancel-delete" @click="$emit('cancel')">cancel</button>
  </div>`
}

const mountBulkDeleteView = () => mount(UsersView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: {
        template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
      },
      DataTable: DataTableStub,
      Pagination: true,
      ConfirmDialog: ConfirmDialogStub,
      EmptyState: true,
      GroupBadge: true,
      Select: true,
      UserAttributesConfigModal: true,
      UserConcurrencyCell: UserConcurrencyCellStub,
      UserCreateModal: true,
      UserEditModal: true,
      BulkEditUserModal: true,
      UserPlatformQuotaModal: true,
      UserApiKeysModal: true,
      UserAllowedGroupsModal: true,
      UserBalanceModal: true,
      UserBalanceHistoryModal: true,
      GroupReplaceModal: true,
      Icon: true,
      Teleport: true
    }
  }
})

describe('admin UsersView', () => {
  beforeEach(() => {
    localStorage.clear()

    listUsers.mockReset()
    toggleStatus.mockReset()
    deleteUser.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    getAllGroups.mockReset()
    getBatchUsersUsage.mockReset()
    listEnabledDefinitions.mockReset()
    getBatchUserAttributes.mockReset()

    listUsers.mockResolvedValue({
      items: [createAdminUser()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
    getBatchUsersUsage.mockResolvedValue({ stats: {} })
    listEnabledDefinitions.mockResolvedValue([])
    getBatchUserAttributes.mockResolvedValue({ values: {} })
  })

  it('cancels bulk deletion without deleting or clearing selected users', async () => {
    const wrapper = mountBulkDeleteView()
    await flushPromises()
    const table = wrapper.findComponent(DataTableStub)
    table.vm.$emit('update:selectedKeys', [42])
    await wrapper.vm.$nextTick()

    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    expect(wrapper.get('[data-test="delete-dialog"]').text()).toContain('admin.users.bulkDelete.confirm:1')
    expect(deleteUser).not.toHaveBeenCalled()

    await wrapper.get('[data-test="cancel-delete"]').trigger('click')
    expect(wrapper.find('[data-test="delete-dialog"]').exists()).toBe(false)
    expect(deleteUser).not.toHaveBeenCalled()
    expect(table.props('selectedKeys')).toEqual([42])
    wrapper.unmount()
  })

  it('deletes selected users and retains failed users for retry', async () => {
    listUsers.mockResolvedValue({
      items: [createAdminUser(), createAdminUser({ id: 43, email: 'other@example.com' })],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    deleteUser.mockImplementation(async (id: number) => {
      if (id === 43) throw new Error('Cannot delete user')
    })
    const wrapper = mountBulkDeleteView()
    await flushPromises()
    const table = wrapper.findComponent(DataTableStub)
    table.vm.$emit('update:selectedKeys', [42, 43])
    await wrapper.vm.$nextTick()
    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(deleteUser.mock.calls).toEqual([[42], [43]])
    expect(table.props('selectedKeys')).toEqual([43])
    expect(showSuccess).toHaveBeenCalledWith('admin.users.bulkDelete.success:1')
    expect(showError).toHaveBeenCalledWith('admin.users.bulkDelete.failed:1')
    expect(wrapper.find('[data-test="delete-dialog"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('keeps users selected during an in-flight bulk deletion', async () => {
    let finishDelete!: () => void
    deleteUser.mockImplementation(() => new Promise<void>((resolve) => { finishDelete = resolve }))
    const wrapper = mountBulkDeleteView()
    await flushPromises()
    const table = wrapper.findComponent(DataTableStub)
    table.vm.$emit('update:selectedKeys', [42])
    await wrapper.vm.$nextTick()
    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="bulk-delete-users"]').attributes('disabled')).toBeDefined()
    table.vm.$emit('update:selectedKeys', [42, 43])
    finishDelete()
    await flushPromises()

    expect(table.props('selectedKeys')).toEqual([43])
    wrapper.unmount()
  })

  it('updates only the toggled row in place without reloading the list', async () => {
    listUsers.mockResolvedValue({
      items: [createAdminUser(), createAdminUser({ id: 43, email: 'other@example.com' })],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    toggleStatus.mockResolvedValue(createAdminUser({ status: 'disabled', updated_at: '2026-04-17T01:00:00Z' }))
    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: UserConcurrencyCellStub,
          UserCreateModal: true,
          UserEditModal: true,
          BulkEditUserModal: BulkEditUserModalStub,
          UserPlatformQuotaModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })
    await flushPromises()
    const toggle = wrapper.get('[data-test="actions-42"]').findAll('button').find(button => button.text() === 'admin.users.disable')
    expect(toggle).toBeDefined()
    await toggle!.trigger('click')
    await flushPromises()
    expect(toggleStatus).toHaveBeenCalledWith(42, 'disabled')
    expect(listUsers).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-test="row-state-42"]').text()).toBe('disabled:2026-04-17T01:00:00Z')
    expect(wrapper.get('[data-test="row-state-43"]').text()).toContain('active:')
    expect(showSuccess).toHaveBeenCalledWith('admin.users.userDisabled')
    expect(showError).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('shows active, used, and created activity columns in order and requests last_used_at sort', async () => {
    localStorage.setItem('user-hidden-columns', JSON.stringify(['concurrency']))

    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: UserConcurrencyCellStub,
          UserCreateModal: true,
          UserEditModal: true,
          BulkEditUserModal: BulkEditUserModalStub,
          UserPlatformQuotaModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text()
    const visibleColumns = columns.split(',')
    expect(visibleColumns.slice(-4, -1)).toEqual(['last_active_at', 'last_used_at', 'created_at'])
    expect(visibleColumns).not.toContain('last_login_at')
    expect(visibleColumns).toContain('concurrency')
    expect(wrapper.get('[data-test="user-concurrency"]').text()).toBe('0 / 1')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })
})
