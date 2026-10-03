import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import type { ChannelMonitor } from '@/api/admin/channelMonitor'
import MonitorPrimaryModelCell from '../MonitorPrimaryModelCell.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

function makeRow(primaryModel: string): ChannelMonitor {
  return {
    id: 1,
    name: 'quota-monitor',
    provider: 'openai',
    endpoint: '',
    api_key_masked: '',
    primary_model: primaryModel,
    extra_models: [],
    group_name: '',
    enabled: true,
    interval_seconds: 60,
    jitter_seconds: 0,
    last_checked_at: null,
    created_by: 1,
    created_at: '2026-08-18T00:00:00Z',
    updated_at: '2026-08-18T00:00:00Z',
    primary_status: 'operational',
    primary_latency_ms: null,
    availability_7d: 100,
    availability_30d: 100,
    extra_models_status: [],
    template_id: null,
    extra_headers: {},
    body_override_mode: 'off',
    body_override: null
  }
}

function mountCell(primaryModel: string) {
  return mount(MonitorPrimaryModelCell, {
    props: { row: makeRow(primaryModel) },
    global: { stubs: { HelpTooltip: { template: '<div><slot name="trigger" /><slot /></div>' } } }
  })
}

describe('MonitorPrimaryModelCell placeholder model display', () => {
  it('replaces the quota placeholder with the localized mode label', () => {
    expect(mountCell('quota').text()).toContain('monitorCommon.checkMode.quota')
  })

  it('keeps a real model name unchanged', () => {
    const wrapper = mountCell('claude-sonnet-4-5')
    expect(wrapper.text()).toContain('claude-sonnet-4-5')
    expect(wrapper.text()).not.toContain('monitorCommon.checkMode.quota')
  })
})
