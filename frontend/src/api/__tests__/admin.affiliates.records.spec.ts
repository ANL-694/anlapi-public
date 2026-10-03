import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  listInviteRecords,
  listRebateRecords,
  listTransferRecords,
  type AffiliateInviteRecord,
  type AffiliateRebateRecord,
  type AffiliateTransferRecord,
} from '../admin/affiliates'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('../client', () => ({ apiClient: { get } }))

describe('admin affiliate records API contract', () => {
  beforeEach(() => get.mockReset())

  it('passes common filters to the invite records endpoint', async () => {
    const data = { items: [] as AffiliateInviteRecord[], total: 0, page: 2, page_size: 10, pages: 0 }
    get.mockResolvedValue({ data })

    await expect(listInviteRecords({
      page: 2,
      page_size: 10,
      search: 'inviter@example.com',
      start_at: '2026-09-01',
      end_at: '2026-09-30',
      sort_by: 'created_at',
      sort_order: 'asc',
      timezone: 'Asia/Shanghai',
    })).resolves.toEqual(data)

    expect(get).toHaveBeenCalledWith('/admin/affiliates/invites', {
      params: {
        page: 2,
        page_size: 10,
        search: 'inviter@example.com',
        start_at: '2026-09-01',
        end_at: '2026-09-30',
        sort_by: 'created_at',
        sort_order: 'asc',
        timezone: 'Asia/Shanghai',
      },
    })
  })

  it('uses the same filter contract for rebate and transfer records', async () => {
    const rebateData = { items: [] as AffiliateRebateRecord[], total: 0, page: 1, page_size: 20, pages: 0 }
    const transferData = { items: [] as AffiliateTransferRecord[], total: 0, page: 1, page_size: 20, pages: 0 }
    get.mockResolvedValueOnce({ data: rebateData }).mockResolvedValueOnce({ data: transferData })

    await expect(listRebateRecords()).resolves.toEqual(rebateData)
    await expect(listTransferRecords({ search: 'withdraw' })).resolves.toEqual(transferData)

    expect(get).toHaveBeenNthCalledWith(1, '/admin/affiliates/rebates', {
      params: {
        page: 1,
        page_size: 20,
        search: '',
        start_at: undefined,
        end_at: undefined,
        sort_by: undefined,
        sort_order: undefined,
        timezone: undefined,
      },
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/affiliates/transfers', {
      params: {
        page: 1,
        page_size: 20,
        search: 'withdraw',
        start_at: undefined,
        end_at: undefined,
        sort_by: undefined,
        sort_order: undefined,
        timezone: undefined,
      },
    })
  })
})
