import { describe, expect, it } from 'vitest'
import { isOverdue, statusLabels } from './status'

describe('feedback display rules', () => {
  it('uses all six Chinese workflow labels', () => {
    expect(Object.values(statusLabels)).toEqual(['待受理', '待补充', '处理中', '待确认', '已关闭', '已驳回'])
  })

  it('does not mark terminal feedback overdue', () => {
    const now = new Date('2026-08-22T12:00:00Z')
    expect(isOverdue('in_progress', '2026-08-22T11:00:00Z', now)).toBe(true)
    expect(isOverdue('closed', '2026-08-22T11:00:00Z', now)).toBe(false)
    expect(isOverdue('rejected', '2026-08-22T11:00:00Z', now)).toBe(false)
  })
})
