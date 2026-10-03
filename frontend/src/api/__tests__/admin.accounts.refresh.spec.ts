import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import { refreshCredentials } from '@/api/admin/accounts'

describe('admin account refresh API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('normalizes the legacy direct-account response', async () => {
    const account = { id: 7, name: 'legacy account' }
    post.mockResolvedValueOnce({ data: account })

    await expect(refreshCredentials(7)).resolves.toEqual({ account })
    expect(post).toHaveBeenCalledWith('/admin/accounts/7/refresh')
  })

  it('preserves the account and temporary project warning', async () => {
    const result = {
      account: { id: 8, name: 'antigravity account' },
      message: 'project_id will be retried',
      warning: 'missing_project_id_temporary'
    }
    post.mockResolvedValueOnce({ data: result })

    await expect(refreshCredentials(8)).resolves.toEqual(result)
  })
})
