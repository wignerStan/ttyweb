import { describe, expect, it } from 'vitest'
import { getTagColorClass } from './tagColors'

describe('getTagColorClass', () => {
  it('returns correct color classes for known tags', () => {
    expect(getTagColorClass('bug')).toBe('tag--red')
    expect(getTagColorClass('feature')).toBe('tag--blue')
    expect(getTagColorClass('enhancement')).toBe('tag--purple')
    expect(getTagColorClass('docs')).toBe('tag--cyan')
    expect(getTagColorClass('refactor')).toBe('tag--yellow')
    expect(getTagColorClass('performance')).toBe('tag--green')
    expect(getTagColorClass('security')).toBe('tag--red')
    expect(getTagColorClass('design')).toBe('tag--purple')
    expect(getTagColorClass('testing')).toBe('tag--cyan')
    expect(getTagColorClass('devops')).toBe('tag--yellow')
    expect(getTagColorClass('ux')).toBe('tag--blue')
    expect(getTagColorClass('api')).toBe('tag--green')
  })

  it('returns deterministic hash-based color for unknown tags', () => {
    const result1 = getTagColorClass('unknown-tag-xyz')
    const result2 = getTagColorClass('unknown-tag-xyz')
    expect(result1).toBe(result2)
    expect(result1).toMatch(/^tag--(blue|purple|cyan|green|yellow|red)$/)
  })

  it('is case-insensitive', () => {
    expect(getTagColorClass('BUG')).toBe('tag--red')
    expect(getTagColorClass('Feature')).toBe('tag--blue')
    expect(getTagColorClass('ENHANCEMENT')).toBe('tag--purple')
  })

  it('trims whitespace', () => {
    expect(getTagColorClass('  bug  ')).toBe('tag--red')
    expect(getTagColorClass('\tfeature\n')).toBe('tag--blue')
  })
})
