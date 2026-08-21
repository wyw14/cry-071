import type { FeedbackStatus, Priority } from '@/api/types'

export const statusLabels: Record<FeedbackStatus, string> = {
  pending_acceptance: '待受理', needs_information: '待补充', in_progress: '处理中',
  pending_confirmation: '待确认', closed: '已关闭', rejected: '已驳回'
}

export const priorityLabels: Record<Priority, string> = { low: '低', normal: '一般', high: '高', critical: '紧急' }

export function isOverdue(status: FeedbackStatus, dueAt: string, now = new Date()): boolean {
  if (status === 'closed' || status === 'rejected') return false
  return new Date(dueAt).getTime() < now.getTime()
}
