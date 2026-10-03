import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import Select from '../Select.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const originalInnerWidth = window.innerWidth
let wrapper: ReturnType<typeof mount> | undefined

const setViewportWidth = (width: number) => {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
}

const mockTriggerRect = (left: number, width: number) => {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: left,
    y: 20,
    top: 20,
    right: left + width,
    bottom: 60,
    left,
    width,
    height: 40,
    toJSON: () => ({}),
  })
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  setViewportWidth(originalInnerWidth)
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('Select dropdown viewport constraints', () => {
  it('preserves the 200px minimum width when space is available', async () => {
    setViewportWidth(1024)
    mockTriggerRect(20, 80)
    wrapper = mount(Select, { props: { modelValue: null, options: [{ value: 'example', label: 'Example' }] } })

    await wrapper.get('button').trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown-portal')
    expect(dropdown?.style.left).toBe('20px')
    expect(dropdown?.style.minWidth).toBe('200px')
    expect(dropdown?.style.maxWidth).toBe('996px')
  })

  it('shrinks the dropdown to fit near the right viewport edge', async () => {
    setViewportWidth(320)
    mockTriggerRect(220, 80)
    wrapper = mount(Select, { props: { modelValue: null, options: [{ value: 'example', label: 'Example' }] } })

    await wrapper.get('button').trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown-portal')
    expect(dropdown?.style.left).toBe('220px')
    expect(dropdown?.style.minWidth).toBe('92px')
    expect(dropdown?.style.maxWidth).toBe('92px')
  })
})

describe('Select remote search', () => {
  it('debounces remote search and keeps local options until the parent updates them', async () => {
    vi.useFakeTimers()
    wrapper = mount(Select, {
      props: {
        modelValue: null,
        remote: true,
        searchable: true,
        options: [
          { value: 'alpha', label: 'Alpha account' },
          { value: 'beta', label: 'Beta account' },
        ],
      },
    })

    await wrapper.get('button').trigger('click')
    await nextTick()
    const input = document.body.querySelector<HTMLInputElement>('.select-search-input')!
    input.value = 'zzz'
    input.dispatchEvent(new Event('input'))
    await nextTick()
    expect(wrapper.emitted('search')).toBeUndefined()

    await vi.advanceTimersByTimeAsync(300)
    expect(wrapper.emitted('search')).toEqual([['zzz']])
    expect([...document.body.querySelectorAll('.select-option-label')].map((el) => el.textContent)).toEqual([
      'Alpha account',
      'Beta account',
    ])
  })

  it('keeps local filtering when remote mode is disabled', async () => {
    vi.useFakeTimers()
    wrapper = mount(Select, {
      props: {
        modelValue: null,
        searchable: true,
        options: [
          { value: 'alpha', label: 'Alpha account' },
          { value: 'beta', label: 'Beta account' },
        ],
      },
    })

    await wrapper.get('button').trigger('click')
    await nextTick()
    const input = document.body.querySelector<HTMLInputElement>('.select-search-input')!
    input.value = 'alpha'
    input.dispatchEvent(new Event('input'))
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)

    expect(wrapper.emitted('search')).toBeUndefined()
    expect([...document.body.querySelectorAll('.select-option-label')].map((el) => el.textContent)).toEqual(['Alpha account'])
  })
})
