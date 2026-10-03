import type { GroupPlatform } from '@/types'

export type KeyGroupProvider = 'anthropic' | 'openai' | 'other'

export const KEY_GROUP_PROVIDERS = ['anthropic', 'openai', 'other'] as const

const PROVIDER_BY_PLATFORM: Record<GroupPlatform, KeyGroupProvider> = {
  anthropic: 'anthropic',
  openai: 'openai',
  gemini: 'other',
  antigravity: 'other',
  grok: 'other',
  kiro: 'other',
  typesafe: 'other',
  opencode_go: 'other',
  custom: 'other',
  composite: 'other'
}

export function getKeyGroupProvider(platform: GroupPlatform): KeyGroupProvider {
  return PROVIDER_BY_PLATFORM[platform]
}

export const KEY_GROUP_PROVIDER_ICONS: Record<KeyGroupProvider, GroupPlatform[]> = {
  anthropic: ['anthropic'],
  openai: ['openai'],
  other: ['gemini', 'grok', 'typesafe', 'opencode_go']
}
