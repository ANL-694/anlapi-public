<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold text-[var(--app-text)]">{{ t('nav.affiliateRecords') }}</h1>
            <p class="mt-1 text-sm text-[var(--app-muted)]">{{ tabDescription }}</p>
          </div>
          <div class="flex items-center gap-2">
            <button
              v-for="tab in tabs"
              :key="tab.value"
              type="button"
              class="btn"
              :class="activeTab === tab.value ? 'btn-primary' : 'btn-secondary'"
              @click="selectTab(tab.value)"
            >
              {{ tab.label }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadRecords">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card flex flex-wrap items-end gap-3 p-4">
          <div class="min-w-[220px] flex-1">
            <label class="input-label">{{ t('admin.affiliates.records.search') }}</label>
            <input
              v-model.trim="filters.search"
              type="search"
              class="input"
              :placeholder="t('admin.affiliates.records.searchPlaceholder')"
              @keyup.enter="search"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.affiliates.records.startAt') }}</label>
            <input v-model="filters.start_at" type="date" class="input" @change="search" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.affiliates.records.endAt') }}</label>
            <input v-model="filters.end_at" type="date" class="input" @change="search" />
          </div>
          <button type="button" class="btn btn-primary" :disabled="loading" @click="search">
            <Icon name="search" size="sm" class="mr-1.5" />
            {{ t('common.search') }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="resetFilters">
            {{ t('common.reset') }}
          </button>
        </div>
        <div v-if="errorMessage" class="mt-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">
          {{ errorMessage }}
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :server-side-sort="true"
          :default-sort-key="'created_at'"
          :default-sort-order="'desc'"
          :row-key="rowKey"
          @sort="handleSort"
        >
          <template #cell-inviter="{ row }">
            <div class="min-w-[180px]">
              <div class="truncate font-medium text-[var(--app-text)]">{{ row.inviter_email || '—' }}</div>
              <div class="truncate text-xs text-[var(--app-muted)]">{{ row.inviter_username || '—' }}</div>
            </div>
          </template>
          <template #cell-invitee="{ row }">
            <div class="min-w-[180px]">
              <div class="truncate font-medium text-[var(--app-text)]">{{ row.invitee_email || '—' }}</div>
              <div class="truncate text-xs text-[var(--app-muted)]">{{ row.invitee_username || '—' }}</div>
            </div>
          </template>
          <template #cell-user="{ row }">
            <div class="min-w-[180px]">
              <div class="truncate font-medium text-[var(--app-text)]">{{ row.user_email || '—' }}</div>
              <div class="truncate text-xs text-[var(--app-muted)]">{{ row.username || '—' }}</div>
            </div>
          </template>
          <template #cell-order="{ row }">
            <span class="font-mono text-xs">{{ row.out_trade_no || row.order_id || '—' }}</span>
          </template>
          <template #cell-aff_code="{ value }">
            <span class="font-mono text-xs">{{ value || '—' }}</span>
          </template>
          <template #cell-action="{ value }">
            <span class="badge" :class="value === 'withdraw' ? 'badge-warning' : 'badge-info'">{{ value }}</span>
          </template>
          <template #cell-order_amount="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-pay_amount="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-total_rebate="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-rebate_amount="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-amount="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-balance_after="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-available_quota_after="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-frozen_quota_after="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-history_quota_after="{ value }"><span>{{ formatMoney(value) }}</span></template>
          <template #cell-created_at="{ value }"><span class="whitespace-nowrap">{{ formatDateTime(value) }}</span></template>
          <template #empty>
            <div class="flex flex-col items-center gap-2 py-10 text-[var(--app-muted)]">
              <Icon name="chart" size="xl" class="h-10 w-10 opacity-50" />
              <span>{{ t('empty.noData') }}</span>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="total > 0"
          :page="page"
          :total="total"
          :page-size="pageSize"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import {
  affiliatesAPI,
  type AffiliateInviteRecord,
  type AffiliateRebateRecord,
  type AffiliateTransferRecord,
} from '@/api/admin/affiliates'
import type { Column } from '@/components/common/types'

type TabValue = 'invites' | 'rebates' | 'transfers'

const { t } = useI18n()
const activeTab = ref<TabValue>('invites')
const loading = ref(false)
const errorMessage = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const sortBy = ref('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')
const inviteRecords = ref<AffiliateInviteRecord[]>([])
const rebateRecords = ref<AffiliateRebateRecord[]>([])
const transferRecords = ref<AffiliateTransferRecord[]>([])
const filters = reactive({ search: '', start_at: '', end_at: '' })

const tabs = computed(() => [
  { value: 'invites' as const, label: t('nav.affiliateInviteRecords') },
  { value: 'rebates' as const, label: t('nav.affiliateRebateRecords') },
  { value: 'transfers' as const, label: t('nav.affiliateTransferRecords') },
])

const tabDescription = computed(() => {
  if (activeTab.value === 'invites') return t('admin.affiliates.invitesDescription')
  if (activeTab.value === 'rebates') return t('admin.affiliates.rebatesDescription')
  return t('admin.affiliates.transfersDescription')
})

const records = computed(() => {
  if (activeTab.value === 'invites') return inviteRecords.value
  if (activeTab.value === 'rebates') return rebateRecords.value
  return transferRecords.value
})

const columns = computed<Column[]>(() => {
  if (activeTab.value === 'invites') {
    return [
      { key: 'inviter', label: t('admin.affiliates.records.inviter') },
      { key: 'invitee', label: t('admin.affiliates.records.invitee') },
      { key: 'aff_code', label: t('admin.affiliates.records.affCode'), sortable: true },
      { key: 'total_rebate', label: t('admin.affiliates.records.totalRebate'), sortable: true },
      { key: 'created_at', label: t('admin.affiliates.records.invitedAt'), sortable: true },
    ]
  }
  if (activeTab.value === 'rebates') {
    return [
      { key: 'order', label: t('admin.affiliates.records.order'), sortable: true },
      { key: 'inviter', label: t('admin.affiliates.records.inviter') },
      { key: 'invitee', label: t('admin.affiliates.records.invitee') },
      { key: 'order_amount', label: t('admin.affiliates.records.orderAmount'), sortable: true },
      { key: 'pay_amount', label: t('admin.affiliates.records.payAmount'), sortable: true },
      { key: 'rebate_amount', label: t('admin.affiliates.records.rebateAmount'), sortable: true },
      { key: 'payment_type', label: t('admin.affiliates.records.paymentType') },
      { key: 'order_status', label: t('admin.affiliates.records.orderStatus') },
      { key: 'created_at', label: t('admin.affiliates.records.rebatedAt'), sortable: true },
    ]
  }
  return [
    { key: 'user', label: t('admin.affiliates.records.user') },
    { key: 'action', label: t('admin.affiliates.records.orderStatus'), sortable: true },
    { key: 'amount', label: t('admin.affiliates.records.transferAmount'), sortable: true },
    { key: 'balance_after', label: t('admin.affiliates.records.balanceAfter'), sortable: true },
    { key: 'available_quota_after', label: t('admin.affiliates.records.availableQuotaAfter'), sortable: true },
    { key: 'frozen_quota_after', label: t('admin.affiliates.records.frozenQuotaAfter'), sortable: true },
    { key: 'history_quota_after', label: t('admin.affiliates.records.historyQuotaAfter'), sortable: true },
    { key: 'created_at', label: t('admin.affiliates.records.transferredAt'), sortable: true },
  ]
})

const rowKey = (row: AffiliateInviteRecord | AffiliateRebateRecord | AffiliateTransferRecord) =>
  'ledger_id' in row ? `transfer-${row.ledger_id}` : 'order_id' in row ? `rebate-${row.order_id ?? row.created_at}` : `invite-${row.inviter_id}-${row.invitee_id}`

function formatMoney(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  const number = Number(value)
  return Number.isFinite(number) ? `$${number.toFixed(4)}` : '—'
}

function search() {
  page.value = 1
  loadRecords()
}

function resetFilters() {
  filters.search = ''
  filters.start_at = ''
  filters.end_at = ''
  search()
}

function selectTab(tab: TabValue) {
  if (activeTab.value === tab) return
  activeTab.value = tab
  page.value = 1
  sortBy.value = 'created_at'
  sortOrder.value = 'desc'
  loadRecords()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortBy.value = key
  sortOrder.value = order
  loadRecords()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  loadRecords()
}

function handlePageSizeChange(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  loadRecords()
}

async function loadRecords() {
  loading.value = true
  errorMessage.value = ''
  const params = {
    page: page.value,
    page_size: pageSize.value,
    search: filters.search,
    start_at: filters.start_at || undefined,
    end_at: filters.end_at || undefined,
    sort_by: sortBy.value,
    sort_order: sortOrder.value,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  }
  try {
    if (activeTab.value === 'invites') {
      const result = await affiliatesAPI.listInviteRecords(params)
      total.value = result.total
      page.value = result.page
      pageSize.value = result.page_size
      inviteRecords.value = result.items
    } else if (activeTab.value === 'rebates') {
      const result = await affiliatesAPI.listRebateRecords(params)
      total.value = result.total
      page.value = result.page
      pageSize.value = result.page_size
      rebateRecords.value = result.items
    } else {
      const result = await affiliatesAPI.listTransferRecords(params)
      total.value = result.total
      page.value = result.page
      pageSize.value = result.page_size
      transferRecords.value = result.items
    }
  } catch {
    errorMessage.value = t('admin.affiliates.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(loadRecords)
</script>
