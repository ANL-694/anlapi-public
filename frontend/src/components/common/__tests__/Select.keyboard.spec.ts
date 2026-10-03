import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import Select from '../Select.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

let wrapper: ReturnType<typeof mount>

afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
})

async function press(key: string) {
  document.activeElement!.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  await nextTick()
}

async function open(searchable: boolean) {
  wrapper = mount(Select, {
    attachTo: document.body,
    props: {
      modelValue: 'alpha',
      searchable,
      options: [
        { value: 'alpha', label: 'Alpha' },
        { value: 'disabled', label: 'Disabled', disabled: true },
        { value: 'beta', label: 'Beta' }
      ]
    },
    global: { stubs: { Icon: true } }
  })
  wrapper.get<HTMLButtonElement>('button').element.focus()
  await press('ArrowDown')
  await nextTick()
}

describe('Select keyboard focus', () => {
  it.each([false, true])('selects an enabled option with searchable=%s', async (searchable) => {
    await open(searchable)
    const expectedFocus = searchable
      ? document.querySelector('.select-search-input')
      : document.querySelector('[role="listbox"]')
    expect(document.activeElement).toBe(expectedFocus)

    await press('ArrowDown')
    await press('Enter')
    expect(wrapper.emitted('update:modelValue')).toEqual([['beta']])
    expect(wrapper.get('button').attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(wrapper.get('button').element)
  })

  it('allows Escape to close a non-searchable dropdown and restore trigger focus', async () => {
    await open(false)
    await press('Escape')
    expect(wrapper.get('button').attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(wrapper.get('button').element)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
