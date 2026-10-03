import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BackupRecord, BackupScheduleConfig } from '../admin/backup'
import { deleteBackup, getSchedule, updateSchedule } from '../admin/backup'

const { get, put, remove } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  remove: vi.fn()
}))

vi.mock('../client', () => ({ apiClient: { get, put, delete: remove } }))

describe('backup API contract', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
    remove.mockReset()
  })

  it('preserves optional monthly archive settings in schedule responses and updates', async () => {
    const schedule: BackupScheduleConfig = {
      enabled: true,
      cron_expr: '0 2 * * *',
      retain_days: 14,
      retain_count: 10,
      monthly_archive: {
        enabled: true,
        days: [1, 15, 1],
        include_month_end: true,
        retain_count: 12
      }
    }
    get.mockResolvedValue({ data: schedule })
    put.mockResolvedValue({ data: schedule })

    await expect(getSchedule()).resolves.toEqual(schedule)
    await expect(updateSchedule(schedule)).resolves.toEqual(schedule)
    expect(get).toHaveBeenCalledWith('/admin/backups/schedule')
    expect(put).toHaveBeenCalledWith('/admin/backups/schedule', schedule)
  })

  it('accepts legacy records without monthly archive metadata', () => {
    const record: BackupRecord = {
      id: 'backup-1',
      status: 'completed',
      backup_type: 'manual',
      file_name: 'backup.sql.gz',
      s3_key: 'backups/backup.sql.gz',
      size_bytes: 128,
      triggered_by: 'admin',
      started_at: '2026-09-25T00:00:00Z'
    }

    expect(record.monthly_archive).toBeUndefined()
  })

  it('only sends explicit archive deletion confirmation when requested', async () => {
    await deleteBackup('archive-1')
    await deleteBackup('archive-1', true)

    expect(remove).toHaveBeenNthCalledWith(1, '/admin/backups/archive-1', undefined)
    expect(remove).toHaveBeenNthCalledWith(2, '/admin/backups/archive-1', { params: { delete_archived: true } })
  })
})
