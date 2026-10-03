import { describe, it, expect } from 'vitest'
import { applyInterceptWarmup, buildPlanTypeOptions, planTypeDisplayLabel } from '../credentialsBuilder'

describe('applyInterceptWarmup', () => {
  it('create + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, true, 'create')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('create + enabled=false: should not add the field', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, false, 'create')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, true, 'edit')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('edit + enabled=false + field exists: should delete the field', () => {
    const creds: Record<string, unknown> = { api_key: 'sk', intercept_warmup_requests: true }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=false + field absent: should not throw', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('should not affect other fields', () => {
    const creds: Record<string, unknown> = {
      api_key: 'sk',
      base_url: 'url',
      intercept_warmup_requests: true
    }
    applyInterceptWarmup(creds, false, 'edit')
    expect(creds.api_key).toBe('sk')
    expect(creds.base_url).toBe('url')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })
})

describe('OpenAI plan type labels', () => {
  it('maps current ChatGPT tiers and normalizes aliases', () => {
    expect(planTypeDisplayLabel('pro')).toBe('Pro 20x')
    expect(planTypeDisplayLabel('CHATGPTPRO')).toBe('Pro 20x')
    expect(planTypeDisplayLabel(' Pro Lite ')).toBe('Pro 5x')
    expect(planTypeDisplayLabel('self-serve-business-pro-lite')).toBe('Business Premium')
    expect(planTypeDisplayLabel('team')).toBe('Business Standard')
  })

  it('keeps the current canonical value while avoiding duplicate labels', () => {
    const options = buildPlanTypeOptions('chatgptpro', 'Automatic')
    expect(options.filter((option) => option.label === 'Pro 20x')).toHaveLength(1)
    expect(options.map((option) => option.value)).toEqual([
      '',
      'plus',
      'chatgptpro',
      'prolite',
      'self_serve_business_prolite',
      'free'
    ])
  })
})
