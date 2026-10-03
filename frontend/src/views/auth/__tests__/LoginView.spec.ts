import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const {
  authLoginMock,
  authStateMock,
  currentRouteMock,
  getPublicSettingsMock,
  isTotp2FARequiredMock,
  passkeyLoginMock,
  pushMock,
  twoFactorLoginMock,
} = vi.hoisted(() => ({
  authLoginMock: vi.fn(),
  authStateMock: { isAdmin: false },
  currentRouteMock: { value: { query: {} as Record<string, unknown> } },
  getPublicSettingsMock: vi.fn(),
  isTotp2FARequiredMock: vi.fn(() => false),
  passkeyLoginMock: vi.fn(),
  pushMock: vi.fn(),
  twoFactorLoginMock: vi.fn(),
}))

const publicSettings = {
  registration_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  backend_mode_enabled: false,
  password_reset_enabled: false,
  passkey_enabled: false,
  login_agreement_enabled: false,
  login_agreement_documents: [],
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock, currentRoute: currentRouteMock }),
}))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    get isAdmin() {
      return authStateMock.isAdmin
    },
    login: authLoginMock,
    loginWithPasskey: passkeyLoginMock,
    login2FA: twoFactorLoginMock,
  }),
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showWarning: vi.fn() }),
}))

vi.mock('@/api/auth', async () => ({
  ...await vi.importActual<typeof import('@/api/auth')>('@/api/auth'),
  getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
  isTotp2FARequired: () => isTotp2FARequiredMock(),
}))

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        DingTalkOAuthSection: true,
        EmailOAuthButtons: true,
        Icon: true,
        LinuxDoOAuthSection: true,
        LoginAgreementPrompt: true,
        OidcOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TotpLoginModal: {
          template: `<button data-test="verify-2fa" @click="$emit('verify', '123456')">Verify</button>`,
          methods: { setVerifying() {}, setError() {} },
        },
        TurnstileWidget: true,
        WechatOAuthSection: true,
        transition: false,
      },
    },
  })
}

describe('LoginView registration entry', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    pushMock.mockReset()
    authLoginMock.mockReset()
    passkeyLoginMock.mockReset()
    twoFactorLoginMock.mockReset()
    isTotp2FARequiredMock.mockReset()
    isTotp2FARequiredMock.mockReturnValue(false)
    authStateMock.isAdmin = false
    currentRouteMock.value.query = {}
    getPublicSettingsMock.mockResolvedValue(publicSettings)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows the registration entry when registration is enabled', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('auth.signUp')
  })

  it('hides the registration entry when registration is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, registration_enabled: false })

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).not.toContain('auth.signUp')
  })

  it.each([
    { role: 'regular user', isAdmin: false, expectedPath: '/dashboard' },
    { role: 'administrator', isAdmin: true, expectedPath: '/admin/dashboard' },
  ])('sends a $role to the role-specific dashboard after password login', async ({ isAdmin, expectedPath }) => {
    authLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = isAdmin
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('#email').setValue('user@anlapi.local')
    await wrapper.find('#password').setValue('secret123')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith(expectedPath)
  })

  it('keeps the intended in-app redirect after password login', async () => {
    currentRouteMock.value.query = { redirect: '/keys' }
    authLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = true
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('#email').setValue('admin@anlapi.local')
    await wrapper.find('#password').setValue('secret123')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith('/keys')
  })

  it('sends an administrator to the admin dashboard after Passkey login', async () => {
    vi.stubGlobal('PublicKeyCredential', class {})
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, passkey_enabled: true })
    passkeyLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = true
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('.btn-secondary').trigger('click')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith('/admin/dashboard')
  })

  it('keeps the intended in-app redirect after Passkey login', async () => {
    vi.stubGlobal('PublicKeyCredential', class {})
    currentRouteMock.value.query = { redirect: '/keys' }
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, passkey_enabled: true })
    passkeyLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = true
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('.btn-secondary').trigger('click')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith('/keys')
  })

  it('sends an administrator to the admin dashboard after 2FA login', async () => {
    isTotp2FARequiredMock.mockReturnValueOnce(true)
    authLoginMock.mockResolvedValue({ two_factor_required: true, temp_token: 'temporary-token' })
    twoFactorLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = true
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('#email').setValue('admin@anlapi.local')
    await wrapper.find('#password').setValue('secret123')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await wrapper.find('[data-test="verify-2fa"]').trigger('click')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith('/admin/dashboard')
  })

  it('keeps the intended in-app redirect after 2FA login', async () => {
    currentRouteMock.value.query = { redirect: '/keys' }
    isTotp2FARequiredMock.mockReturnValueOnce(true)
    authLoginMock.mockResolvedValue({ two_factor_required: true, temp_token: 'temporary-token' })
    twoFactorLoginMock.mockImplementation(async () => {
      authStateMock.isAdmin = true
      return {}
    })

    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.find('#email').setValue('admin@anlapi.local')
    await wrapper.find('#password').setValue('secret123')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await wrapper.find('[data-test="verify-2fa"]').trigger('click')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith('/keys')
  })
})
