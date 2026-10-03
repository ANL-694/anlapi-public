import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getHistory } from '../redeem'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('../client', () => ({ apiClient: { get } }))

describe('redemption history pagination', () => {
  beforeEach(() => vi.clearAllMocks())

  it.each([
    [undefined, undefined, 1, 20],
    [3, 50, 3, 50],
    [2, 100, 2, 100]
  ])('sends page %s and size %s to the server', async (page, size, expectedPage, expectedSize) => {
    const response = { items: [], total: 105, page: expectedPage, page_size: expectedSize, pages: 6 }
    get.mockResolvedValue({ data: response })

    expect(await getHistory(page, size)).toEqual(response)
    expect(get).toHaveBeenCalledWith('/redeem/history', {
      params: { page: expectedPage, page_size: expectedSize }
    })
  })

  it('normalizes the legacy array response during a rolling backend upgrade', async () => {
    const legacyItems = [{ id: 1, code: 'LEGACY', type: 'balance', value: 5 }]
    get.mockResolvedValue({ data: legacyItems })

    await expect(getHistory(2, 50)).resolves.toEqual({
      items: legacyItems,
      total: 1,
      page: 1,
      page_size: 1,
      pages: 1
    })
  })
})
