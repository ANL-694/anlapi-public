<template>
  <div class="min-w-0 max-w-full space-y-1" data-testid="opencode-go-quota-cell">
    <UsageProgressBar
      v-if="quota?.rolling.status === 'ok'"
      label="5h"
      :utilization="quota.rolling.percent"
      :resets-at="quota.rolling.resets_at"
      color="indigo"
      data-testid="opencode-go-quota-rolling"
    />
    <UsageProgressBar
      v-if="quota?.weekly.status === 'ok'"
      label="7d"
      :utilization="quota.weekly.percent"
      :resets-at="quota.weekly.resets_at"
      color="emerald"
      data-testid="opencode-go-quota-weekly"
    />
    <UsageProgressBar
      v-if="quota?.monthly.status === 'ok'"
      label="30d"
      :utilization="quota.monthly.percent"
      :resets-at="quota.monthly.resets_at"
      color="purple"
      data-testid="opencode-go-quota-monthly"
    />

    <div class="flex items-center gap-1.5">
      <span v-if="error" class="truncate text-[10px] text-amber-600 dark:text-amber-400" data-testid="opencode-go-quota-error">
        {{ error }}
      </span>
      <span v-else-if="!hasWindows && !loading" class="text-[10px] text-gray-400 dark:text-gray-500">
        {{ t('admin.accounts.openCodeQuota.unknown') }}
      </span>
      <button
        type="button"
        class="inline-flex shrink-0 items-center gap-0.5 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 transition-colors hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
        :disabled="loading"
        data-testid="opencode-go-quota-refresh"
        @click="refresh"
      >
        <svg
          class="h-2.5 w-2.5"
          :class="{ 'animate-spin': loading }"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
        {{ loading ? t('admin.accounts.openCodeQuota.refreshing') : t('admin.accounts.openCodeQuota.refresh') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { OpenCodeGoQuotaUsageData, OpenCodeGoQuotaWindow } from '@/api/admin/accounts'
import type { Account } from '@/types'
import UsageProgressBar from './UsageProgressBar.vue'

const props = defineProps<{ account: Account }>()
const { t } = useI18n()

const loading = ref(false)
const error = ref<string | null>(null)

const readPercent = (value: unknown): number | null => {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null
  return Math.max(0, value)
}

const readResetAt = (value: unknown): string | null => {
  if (typeof value !== 'string' || value.trim() === '') return null
  return value
}

const readWindow = (extra: Record<string, unknown>, percentKey: string, resetKey: string): OpenCodeGoQuotaWindow => {
  const percent = readPercent(extra[percentKey])
  return {
    status: percent === null ? 'unknown' : 'ok',
    percent: percent ?? 0,
    resets_at: readResetAt(extra[resetKey])
  }
}

const readCachedQuota = (account: Account): OpenCodeGoQuotaUsageData | null => {
  const extra = account.extra as Record<string, unknown> | undefined
  if (!extra) return null
  const data = {
    rolling: readWindow(extra, 'opencode_go_5h_used_percent', 'opencode_go_5h_reset_at'),
    weekly: readWindow(extra, 'opencode_go_weekly_used_percent', 'opencode_go_weekly_reset_at'),
    monthly: readWindow(extra, 'opencode_go_monthly_used_percent', 'opencode_go_monthly_reset_at')
  }
  return [data.rolling, data.weekly, data.monthly].some((window) => window.status === 'ok') ? data : null
}

const quota = ref<OpenCodeGoQuotaUsageData | null>(readCachedQuota(props.account))
const hasWindows = computed(() => {
  if (!quota.value) return false
  return [quota.value.rolling, quota.value.weekly, quota.value.monthly].some((window) => window.status === 'ok')
})

watch(
  () => props.account,
  (account) => {
    if (!loading.value) quota.value = readCachedQuota(account)
  },
  { deep: true }
)

const refresh = async () => {
  if (loading.value) return
  loading.value = true
  error.value = null
  try {
    const result = await adminAPI.accounts.queryOpenCodeGoQuota(props.account.id)
    quota.value = result.data ?? null
    if (!result.data && result.error) error.value = t('admin.accounts.openCodeQuota.queryFailed')
  } catch {
    error.value = t('admin.accounts.openCodeQuota.queryFailed')
  } finally {
    loading.value = false
  }
}
</script>
