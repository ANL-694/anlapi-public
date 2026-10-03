import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getUserOverview,
  withdrawUserQuota,
  type AffiliateUserOverview,
  type AffiliateWithdrawResult,
} from '../admin/affiliates'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('../client', () => ({ apiClient: { get, post } }))

describe('admin affiliate withdrawal API contract', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('loads a read-only user overview', async () => {
    const overview: AffiliateUserOverview = {
      user_id: 42,
      email: 'inviter@example.com',
      username: 'inviter',
      aff_code: 'INVITER',
      rebate_rate_percent: 10,
      invited_count: 2,
      rebated_invitee_count: 1,
      available_quota: 12.5,
      history_quota: 20,
    }
    get.mockResolvedValue({ data: overview })

    await expect(getUserOverview(42)).resolves.toEqual(overview)
    expect(get).toHaveBeenCalledWith('/admin/affiliates/users/42/overview')
  })

  it('sends the idempotency key and exposes replay response header', async () => {
    const result: AffiliateWithdrawResult = {
      ledger_id: 9,
      user_id: 42,
      amount: 2.5,
      available_quota_after: 10,
      frozen_quota_after: 0,
      history_quota_after: 20,
    }
    post.mockResolvedValue({
      data: result,
      headers: { 'x-idempotency-replayed': 'true' },
    })

    await expect(withdrawUserQuota(42, { amount: 2.5 }, 'offline-withdraw-1')).resolves.toEqual({
      result,
      replayed: true,
    })
    expect(post).toHaveBeenCalledWith(
      '/admin/affiliates/users/42/withdraw',
      { amount: 2.5 },
      { headers: { 'Idempotency-Key': 'offline-withdraw-1' } },
    )
  })
})
