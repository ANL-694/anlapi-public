import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const headerSource = readFileSync(resolve(dir, '../AppHeader.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('configured URL sanitization', () => {
  it('sanitizes the document URL in the header and key usage view', () => {
    expect(headerSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(headerSource).toContain('sanitizeUrl(appStore.docUrl)')
    expect(keyUsageViewSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(keyUsageViewSource).toContain('sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl')
  })

  it('sanitizes the HomeView logo source', () => {
    expect(homeViewSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(homeViewSource).toContain('sanitizeUrl(')
    expect(homeViewSource).toContain('allowRelative: true')
  })
})
