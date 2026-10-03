import { describe, expect, it, vi } from 'vitest'

import { fetchAllAccountIds } from '../accountSelection'

describe('fetchAllAccountIds', () => {
  it('loads every page with the same filters and returns unique IDs', async () => {
    const fetchPage = vi.fn(async (page: number, pageSize: number) => {
      const start = (page - 1) * pageSize + 1
      const end = Math.min(page * pageSize, 1001)
      return {
        items: Array.from({ length: end - start + 1 }, (_, index) => ({ id: start + index })),
        total: 1001,
        pages: 2,
      }
    })

    const ids = await fetchAllAccountIds(fetchPage, { platform: 'grok' })

    expect(ids).toHaveLength(1001)
    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(fetchPage).toHaveBeenNthCalledWith(1, 1, 1000, {
      platform: 'grok',
      lite: '1',
      include_scheduler_score: '0',
    })
  })

  it('rejects incomplete results', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ items: [{ id: 1 }, { id: 1 }], total: 2, pages: 1 })

    await expect(fetchAllAccountIds(fetchPage, {})).rejects.toThrow('账号列表结果不完整')
  })
})
