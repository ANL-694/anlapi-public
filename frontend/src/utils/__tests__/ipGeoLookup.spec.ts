import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fetchBatch, fetchOne, formatGeoLabel, getEntry, isPrivateIp } from '../ipGeoLookup'

describe('isPrivateIp', () => {
  it('identifies private and reserved IPv4 ranges', () => {
    expect(isPrivateIp('10.0.0.1')).toBe(true)
    expect(isPrivateIp('127.0.0.1')).toBe(true)
    expect(isPrivateIp('192.168.1.1')).toBe(true)
    expect(isPrivateIp('172.16.0.1')).toBe(true)
    expect(isPrivateIp('169.254.1.1')).toBe(true)
  })

  it('identifies private IPv6 ranges without overmatching nearby public ranges', () => {
    expect(isPrivateIp('::1')).toBe(true)
    expect(isPrivateIp('fe80::1')).toBe(true)
    expect(isPrivateIp('fc00::1')).toBe(true)
    expect(isPrivateIp('fdff::1')).toBe(true)
    expect(isPrivateIp('fec0::1')).toBe(false)
    expect(isPrivateIp('fe7f::1')).toBe(false)
  })

  it('does not flag public addresses', () => {
    expect(isPrivateIp('8.8.8.8')).toBe(false)
    expect(isPrivateIp('172.32.0.1')).toBe(false)
    expect(isPrivateIp('121.35.47.43')).toBe(false)
  })
})

describe('formatGeoLabel', () => {
  it('joins available country, region, and city fields', () => {
    expect(formatGeoLabel({ countryCode: 'CN', region: 'Guangdong', city: 'Shenzhen' })).toBe('CN · Guangdong · Shenzhen')
    expect(formatGeoLabel({ countryCode: 'CN' })).toBe('CN')
  })
})

describe('fetchOne', () => {
  beforeEach(() => {
    localStorage.clear()
    global.fetch = vi.fn()
  })

  it('marks a private IP without making a network request', async () => {
    await fetchOne('192.168.50.1')
    expect(getEntry('192.168.50.1')).toEqual({ status: 'private' })
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('stores a successful geolocation result and reuses it from cache', async () => {
    const fetchMock = global.fetch as ReturnType<typeof vi.fn>
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        ip: '8.8.8.8',
        country_code: 'US',
        region: 'California',
        city: 'Mountain View',
        organization: 'Example ISP',
      }),
    })

    await fetchOne('8.8.8.8')
    expect(getEntry('8.8.8.8')).toEqual(expect.objectContaining({
      status: 'success',
      label: 'US · California · Mountain View',
      detail: expect.objectContaining({ organization: 'Example ISP' }),
    }))

    await fetchOne('8.8.8.8')
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })

  it('marks missing country data and rejected requests as errors', async () => {
    const fetchMock = global.fetch as ReturnType<typeof vi.fn>
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ip: '192.0.2.55' }),
    })
    await fetchOne('192.0.2.55')
    expect(getEntry('192.0.2.55').status).toBe('error')

    fetchMock.mockRejectedValueOnce(new Error('network down'))
    await fetchOne('198.51.100.7')
    expect(getEntry('198.51.100.7').status).toBe('error')
  })
})

describe('fetchBatch', () => {
  beforeEach(() => {
    localStorage.clear()
    global.fetch = vi.fn()
  })

  it('deduplicates IPs and skips private addresses', async () => {
    const fetchMock = global.fetch as ReturnType<typeof vi.fn>
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => [{ ip: '203.0.113.10', country_code: 'US', region: 'Texas', city: 'Dallas' }],
    })

    await fetchBatch(['203.0.113.10', '203.0.113.10', '10.0.0.5'])

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain('ip=203.0.113.10')
    expect(calledUrl).not.toContain('203.0.113.10,203.0.113.10')
    expect(getEntry('10.0.0.5').status).toBe('private')
  })

  it('splits more than 50 IPs into multiple chunk requests', async () => {
    const ips = Array.from({ length: 61 }, (_, i) => `203.0.${Math.floor(i / 250)}.${(i % 250) + 1}`)
    const fetchMock = global.fetch as ReturnType<typeof vi.fn>
    fetchMock.mockImplementation(async (url: string) => ({
      ok: true,
      json: async () => new URL(url).searchParams.get('ip')!.split(',').map((ip) => ({ ip, country_code: 'US' })),
    }))

    await fetchBatch(ips)

    expect(global.fetch).toHaveBeenCalledTimes(2)
    expect(new URL(fetchMock.mock.calls[0][0]).searchParams.get('ip')!.split(',')).toHaveLength(50)
    expect(new URL(fetchMock.mock.calls[1][0]).searchParams.get('ip')!.split(',')).toHaveLength(11)
  })

  it('marks missing batch results as errors while preserving a successful batch status', async () => {
    const fetchMock = global.fetch as ReturnType<typeof vi.fn>
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => [{ ip: '203.0.113.20', country_code: 'US' }],
    })

    const ok = await fetchBatch(['203.0.113.20', '203.0.113.21'])

    expect(ok).toBe(true)
    expect(getEntry('203.0.113.20').status).toBe('success')
    expect(getEntry('203.0.113.21').status).toBe('error')
  })
})
