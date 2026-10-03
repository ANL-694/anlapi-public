<template>
  <AppLayout>
    <div class="custom-page-layout">
      <div class="card flex-1 min-h-0 overflow-hidden">
        <div v-if="loading" class="flex h-full items-center justify-center py-12">
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="!menuItem"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notFoundTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notFoundDesc') }}
            </p>
          </div>
        </div>

        <div v-else-if="!isValidUrl" class="flex h-full items-center justify-center p-10 text-center">
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notConfiguredTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <div v-else ref="embedShell" class="custom-embed-shell">
          <a
            v-if="!menuItem?.hide_open_button"
            ref="openButton"
            :href="embeddedUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm custom-open-fab"
            :style="openButtonPosition ? { left: `${openButtonPosition.x}px`, top: `${openButtonPosition.y}px`, right: 'auto' } : undefined"
            @pointerdown="startButtonDrag"
            @pointermove="moveButtonDrag"
            @pointerup="endButtonDrag"
            @pointercancel="endButtonDrag"
            @lostpointercapture="endButtonDrag"
            @dragstart.prevent
            @click="handleOpenButtonClick"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('customPage.openInNewTab') }}
          </a>
          <iframe
            :src="embeddedUrl"
            class="custom-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useResizeObserver } from '@vueuse/core'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'

const { t, locale } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(false)
const pageTheme = ref<'light' | 'dark'>('light')
let themeObserver: MutationObserver | null = null

const embedShell = ref<HTMLElement | null>(null)
const openButton = ref<HTMLAnchorElement | null>(null)
const openButtonPosition = ref<{ x: number; y: number } | null>(null)
let buttonDrag: { pointerId: number; startX: number; startY: number; x: number; y: number } | null = null
let suppressButtonClick = false

function setOpenButtonPosition(x: number, y: number) {
  const shell = embedShell.value
  const button = openButton.value
  if (!shell || !button) return
  openButtonPosition.value = {
    x: Math.max(0, Math.min(x, shell.clientWidth - button.offsetWidth)),
    y: Math.max(0, Math.min(y, shell.clientHeight - button.offsetHeight)),
  }
}

function startButtonDrag(event: PointerEvent) {
  if (event.button !== 0 || !event.isPrimary || !openButton.value) return
  const button = openButton.value
  suppressButtonClick = false
  buttonDrag = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    x: button.offsetLeft,
    y: button.offsetTop,
  }
  button.setPointerCapture(event.pointerId)
}

function moveButtonDrag(event: PointerEvent) {
  if (!buttonDrag || event.pointerId !== buttonDrag.pointerId) return
  const dx = event.clientX - buttonDrag.startX
  const dy = event.clientY - buttonDrag.startY
  if (!suppressButtonClick && Math.hypot(dx, dy) < 4) return
  suppressButtonClick = true
  setOpenButtonPosition(buttonDrag.x + dx, buttonDrag.y + dy)
  event.preventDefault()
}

function endButtonDrag(event: PointerEvent) {
  if (!buttonDrag || event.pointerId !== buttonDrag.pointerId) return
  buttonDrag = null
  if (openButton.value?.hasPointerCapture(event.pointerId)) {
    openButton.value.releasePointerCapture(event.pointerId)
  }
}

function handleOpenButtonClick(event: MouseEvent) {
  if (suppressButtonClick && event.detail !== 0) event.preventDefault()
  suppressButtonClick = false
}

useResizeObserver([embedShell, openButton], () => {
  if (openButtonPosition.value) {
    setOpenButtonPosition(openButtonPosition.value.x, openButtonPosition.value.y)
  }
})

const menuItemId = computed(() => route.params.id as string)

const menuItem = computed(() => {
  const id = menuItemId.value
  // Try public settings first (contains user-visible items)
  const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  const found = publicItems.find((item) => item.id === id) ?? null
  if (found) return found
  // For admin users, also check admin settings (contains admin-only items)
  if (authStore.isAdmin) {
    return adminSettingsStore.customMenuItems.find((item) => item.id === id) ?? null
  }
  return null
})

const embeddedUrl = computed(() => {
  if (!menuItem.value) return ''
  return buildEmbeddedUrl(
    menuItem.value.url,
    authStore.user?.id,
    authStore.token,
    pageTheme.value,
    locale.value,
  )
})

const isValidUrl = computed(() => {
  const url = embeddedUrl.value
  return url.startsWith('http://') || url.startsWith('https://')
})

onMounted(async () => {
  pageTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      pageTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  if (appStore.publicSettingsLoaded) return
  loading.value = true
  try {
    await appStore.fetchPublicSettings()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.custom-page-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.custom-embed-shell {
  @apply relative;
  @apply h-full w-full overflow-hidden rounded-2xl;
  @apply bg-gradient-to-b from-gray-50 to-white dark:from-dark-900 dark:to-dark-950;
  @apply p-0;
}

.custom-open-fab {
  @apply absolute right-3 top-3 z-10 w-max max-w-full touch-none select-none transition-colors;
  @apply shadow-sm backdrop-blur supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-dark-800/80;
}

.custom-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}
</style>
