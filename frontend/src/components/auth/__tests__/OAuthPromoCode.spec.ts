import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import EmailOAuthButtons from '@/components/auth/EmailOAuthButtons.vue'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'

const routeState = vi.hoisted(() => ({
  query: { redirect: '/dashboard' } as Record<string, unknown>,
}))

const locationState = vi.hoisted(() => ({
  current: { href: 'http://localhost/register' },
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params?.providerName ? `${key}:${params.providerName}` : key,
  }),
}))

function searchParams(): URLSearchParams {
  return new URL(locationState.current.href, 'http://localhost').searchParams
}

describe('registration OAuth promo code forwarding', () => {
  beforeEach(() => {
    routeState.query = { redirect: '/dashboard?from=register' }
    locationState.current = { href: 'http://localhost/register' }
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState.current,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('forwards a trimmed promo code through email OAuth', async () => {
    const wrapper = mount(EmailOAuthButtons, {
      props: { githubEnabled: true, promoCode: '  PROMO-42  ' },
    })

    await wrapper.get('button').trigger('click')

    expect(searchParams().get('promo_code')).toBe('PROMO-42')
    expect(searchParams().get('redirect')).toBe('/dashboard?from=register')
  })

  it('omits an empty promo code from LinuxDo OAuth', async () => {
    const wrapper = mount(LinuxDoOAuthSection, {
      props: { promoCode: '   ' },
    })

    await wrapper.get('button').trigger('click')

    expect(searchParams().has('promo_code')).toBe(false)
    expect(searchParams().get('redirect')).toBe('/dashboard?from=register')
  })
})
