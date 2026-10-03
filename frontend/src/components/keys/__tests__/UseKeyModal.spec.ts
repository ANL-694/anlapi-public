import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const { copyToClipboardMock } = vi.hoisted(() => ({
  copyToClipboardMock: vi.fn().mockResolvedValue(true)
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: copyToClipboardMock
  })
}))

import UseKeyModal from '../UseKeyModal.vue'

function mountUseKeyModal() {
  return mount(UseKeyModal, {
    props: {
      show: true,
      apiKey: 'sk-test',
      baseUrl: 'https://example.com/v1',
      platform: 'openai'
    },
    global: {
      stubs: {
        BaseDialog: {
          template: '<div><slot /><slot name="footer" /></div>'
        },
        Icon: {
          template: '<span />'
        }
      }
    }
  })
}

describe('UseKeyModal', () => {
  it('renders current Codex CLI OpenAI config without local access provider alias', () => {
    const wrapper = mountUseKeyModal()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('model = "gpt-5.5"')
    expect(codeBlock.text()).toContain('review_model = "gpt-5.5"')
    expect(codeBlock.text()).toContain('[model_providers.OpenAI]')
    expect(codeBlock.text()).not.toContain('[model_providers.codex_local_access]')
    expect(codeBlock.text()).toContain('base_url = "https://example.com/v1"')
    expect(codeBlock.text()).toContain('wire_api = "responses"')
  })

  it('renders WebSocket Codex CLI config without local access provider alias', async () => {
    const wrapper = mountUseKeyModal()

    const wsTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCliWs')
    )

    expect(wsTab).toBeDefined()
    await wsTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('[model_providers.OpenAI]')
    expect(codeBlock.text()).not.toContain('[model_providers.codex_local_access]')
    expect(codeBlock.text()).toContain('supports_websockets = true')
    expect(codeBlock.text()).toContain('[features]')
    expect(codeBlock.text()).toContain('responses_websockets_v2 = true')
  })

  it('renders GPT-5.4 mini entry in OpenCode config', async () => {
    const wrapper = mountUseKeyModal()

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('"name": "GPT-5.4 Mini"')
    expect(codeBlock.text()).not.toContain('"name": "GPT-5.4 Nano"')
  })

  it('renders GPT-6 and Claude Opus 5.5 entries in OpenCode config', async () => {
    const openaiWrapper = mountUseKeyModal()
    const opencodeTab = openaiWrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const openaiCode = openaiWrapper.find('pre code').text()
    expect(openaiCode).toContain('"name": "GPT-6 Sol"')
    expect(openaiCode).toContain('"name": "GPT-6 Luna"')

    const claudeWrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-anthropic-test',
        baseUrl: 'https://example.com/v1',
        platform: 'antigravity'
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })
    const claudeOpencodeTab = claudeWrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(claudeOpencodeTab).toBeDefined()
    await claudeOpencodeTab!.trigger('click')
    await nextTick()

    expect(claudeWrapper.findAll('pre code').map((code) => code.text()).join('\n'))
      .toContain('"name": "Claude Opus 5.5"')
  })

  it('does not emit the obsolete Claude attribution override', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-anthropic-test',
        baseUrl: 'https://example.com/v1',
        platform: 'anthropic'
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })

    const codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    expect(codeBlocks.join('\n')).toContain('CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1')
    expect(codeBlocks.join('\n')).not.toContain('CLAUDE_CODE_ATTRIBUTION_HEADER')
  })

  it('normalizes OpenAI Codex and Claude base URLs independently', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com',
        platform: 'openai',
        allowMessagesDispatch: true
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })

    const codexConfig = wrapper.findAll('pre code')
      .map((code) => code.text())
      .find((content) => content.includes('model_provider = "OpenAI"'))
    expect(codexConfig).toContain('base_url = "https://example.com/v1"')

    const claudeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.claudeCode')
    )
    expect(claudeTab).toBeDefined()
    await claudeTab!.trigger('click')
    await nextTick()

    const claudeConfig = wrapper.findAll('pre code')
      .map((code) => code.text())
      .find((content) => content.startsWith('export ANTHROPIC_BASE_URL'))
    expect(claudeConfig).toContain('ANTHROPIC_BASE_URL="https://example.com"')
  })

  it('renders a native TypeSafe System One curl example', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'ts-test',
        baseUrl: 'https://typesafe.example.com/',
        platform: 'typesafe'
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('https://typesafe.example.com/v1/systemone')
    expect(codeBlock.text()).toContain('Authorization: Bearer ts-test')
    expect(codeBlock.text()).toContain('"model": "jev-latest"')
    expect(codeBlock.text()).toContain('"questions"')
    expect(codeBlock.text()).not.toContain('ANTHROPIC_BASE_URL')
  })
})
