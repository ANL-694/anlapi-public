import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('configured logo sanitization', () => {
  it('sanitizes logos in the sidebar, home view, and key usage view', () => {
    for (const source of [sidebarSource, homeViewSource, keyUsageViewSource]) {
      expect(source).toContain("import { sanitizeUrl } from '@/utils/url'")
      expect(source).toContain('allowRelative: true')
      expect(source).toContain('allowDataUrl: true')
    }
    expect(sidebarSource).toContain('sanitizeUrl(appStore.siteLogo')
  })
})
