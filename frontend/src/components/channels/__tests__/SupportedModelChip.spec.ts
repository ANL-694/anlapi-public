import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SupportedModelChip from '../SupportedModelChip.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('SupportedModelChip video pricing', () => {
  it.each(['availableChannels.pricing', 'admin.availableChannels.pricing'])(
    'shows per-second prices with the %s prefix',
    async (pricingKeyPrefix) => {
      const wrapper = mount(SupportedModelChip, {
        attachTo: document.body,
        props: {
          pricingKeyPrefix,
          model: {
            name: 'video-test',
            platform: '',
            pricing: {
              billing_mode: 'video',
              input_price: null,
              output_price: null,
              cache_write_price: null,
              cache_read_price: null,
              image_input_price: null,
              image_output_price: null,
              per_request_price: 0.05,
              intervals: [{
                tier_label: '480p',
                min_tokens: 0,
                max_tokens: null,
                input_price: null,
                output_price: null,
                cache_write_price: null,
                cache_read_price: null,
                per_request_price: 0.12
              }]
            }
          }
        }
      })

      try {
        await wrapper.find('[tabindex="0"]').trigger('mouseenter')
        await nextTick()
        const tooltip = document.body.querySelector('[role="tooltip"]')
        expect(tooltip?.textContent).toContain(`${pricingKeyPrefix}.billingModeVideo`)
        expect(tooltip?.textContent).toContain(`${pricingKeyPrefix}.videoPrice`)
        expect(tooltip?.textContent).toContain(`$0.05 ${pricingKeyPrefix}.unitPerSecond`)
        expect(tooltip?.textContent).toContain(`$0.12 ${pricingKeyPrefix}.unitPerSecond`)
      } finally {
        wrapper.unmount()
      }
    }
  )
})
