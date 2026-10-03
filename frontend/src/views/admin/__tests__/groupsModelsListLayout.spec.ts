import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const groupsViewSource = readFileSync(resolve(testDirectory, '../GroupsView.vue'), 'utf8')
const channelsViewSource = readFileSync(resolve(testDirectory, '../ChannelsView.vue'), 'utf8')
const intervalSource = readFileSync(
  resolve(testDirectory, '../../../components/admin/channel/IntervalRow.vue'),
  'utf8'
)
const pricingSource = readFileSync(
  resolve(testDirectory, '../../../components/admin/channel/PricingEntryCard.vue'),
  'utf8'
)

describe('admin pricing layout', () => {
  it('keeps pricing controls responsive in the ANL admin structure', () => {
    expect(groupsViewSource.match(/width="wide"/g)).toHaveLength(2)
    expect(channelsViewSource).toContain('flex flex-wrap items-start justify-between gap-2')
    expect(channelsViewSource).toContain('shrink-0 whitespace-nowrap')
    expect(intervalSource).toContain('pricing-interval-grid')
    expect(pricingSource).toContain('pricing-default-grid')
  })
})
