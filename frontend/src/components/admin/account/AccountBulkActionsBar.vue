<template>
  <div class="bulk-actions-bar">
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium text-[var(--app-text)]">
        {{ selectedIds.length > 0
          ? t(allResultsSelected ? 'admin.accounts.bulkActions.selectedAll' : 'admin.accounts.bulkActions.selected', { count: selectedIds.length })
          : t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
        <button type="button" class="bulk-actions-link" @click="$emit('select-page')">
          {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
        </button>
        <template v-if="!allResultsSelected && totalResults > selectedIds.length">
          <span class="text-[var(--app-muted)]" aria-hidden="true">/</span>
          <button
            type="button"
            class="bulk-actions-link"
            :disabled="selectingAll"
            @click="$emit('select-all-results')"
          >
            {{ selectingAll
              ? t('admin.accounts.bulkActions.selectingAll')
              : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults }) }}
          </button>
        </template>
        <span class="text-[var(--app-muted)]" aria-hidden="true">/</span>
        <button type="button" class="bulk-actions-link" @click="$emit('clear')">
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
      <template v-else-if="!allResultsSelected && totalResults > 0">
        <button
          type="button"
          class="bulk-actions-link"
          :disabled="selectingAll"
          @click="$emit('select-all-results')"
        >
          {{ selectingAll
            ? t('admin.accounts.bulkActions.selectingAll')
            : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults }) }}
        </button>
      </template>
    </div>

    <div class="flex flex-wrap justify-end gap-2">
      <template v-if="selectedIds.length > 0">
        <button type="button" class="btn btn-danger btn-sm" @click="$emit('delete')">
          {{ t('admin.accounts.bulkActions.delete') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="$emit('reset-status')">
          {{ t('admin.accounts.bulkActions.resetStatus') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="$emit('refresh-token')">
          {{ t('admin.accounts.bulkActions.refreshToken') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="$emit('probe-upstream-billing')">
          {{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}
        </button>
        <button type="button" class="btn btn-success btn-sm" @click="$emit('toggle-schedulable', true)">
          {{ t('admin.accounts.bulkActions.enableScheduling') }}
        </button>
        <button type="button" class="btn btn-warning btn-sm" @click="$emit('toggle-schedulable', false)">
          {{ t('admin.accounts.bulkActions.disableScheduling') }}
        </button>
        <button type="button" class="btn btn-primary btn-sm" @click="$emit('edit-selected')">
          {{ t('admin.accounts.bulkActions.edit') }}
        </button>
      </template>
      <button type="button" class="btn btn-primary btn-sm" @click="$emit('edit-filtered')">
        {{ t('admin.accounts.bulkEdit.submit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{
  selectedIds: number[]
  totalResults?: number
  selectingAll?: boolean
  allResultsSelected?: boolean
}>(), {
  totalResults: 0,
  selectingAll: false,
  allResultsSelected: false,
})
defineEmits<{
  delete: []
  'edit-selected': []
  'edit-filtered': []
  clear: []
  'select-page': []
  'select-all-results': []
  'toggle-schedulable': [enabled: boolean]
  'reset-status': []
  'refresh-token': []
  'probe-upstream-billing': []
}>()

const { t } = useI18n()
</script>

<style scoped>
.bulk-actions-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 0.75rem 0;
  border-top: 1px solid var(--ui-border);
  border-bottom: 1px solid var(--ui-border);
}

.bulk-actions-link {
  color: var(--ui-text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
}

.bulk-actions-link:hover {
  color: var(--ui-text);
  text-decoration: underline;
}

@media (max-width: 640px) {
  .bulk-actions-bar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
