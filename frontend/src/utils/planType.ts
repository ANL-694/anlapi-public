/**
 * OpenAI ChatGPT 订阅档位的归一化和展示映射。
 * 同一档位可能来自不同的大小写、空格、下划线或连字符写法。
 */
export function normalizePlanType(value?: string | null): string {
  return (value || '').trim().toLowerCase().replace(/[\s_-]+/g, '')
}

/** OpenAI 专用档位标签；未知值交给调用方回退到原始值。 */
export function openAIPlanTypeLabel(value?: string | null): string {
  switch (normalizePlanType(value)) {
    case 'plus':
      return 'Plus'
    case 'chatgptpro':
    case 'pro':
      return 'Pro 20x'
    case 'prolite':
      return 'Pro 5x'
    case 'selfservebusinessprolite':
      return 'Business Premium'
    case 'team':
      return 'Business Standard'
    case 'free':
      return 'Free'
    default:
      return ''
  }
}
