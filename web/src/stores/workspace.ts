import { defineStore } from 'pinia'
import type { Feedback, FeedbackStatus, Priority } from '@/api/types'

export const useWorkspaceStore = defineStore('workspace', {
  state: () => ({
    queue: [] as Feedback[],
    selected: new Set<string>(),
    queryToken: localStorage.getItem('feedback-query-token') ?? '',
    filters: { area_id: '', status: '' as FeedbackStatus | '', priority: '' as Priority | '', overdue: false },
    loading: false,
    error: ''
  }),
  actions: {
    rememberToken(token: string) { this.queryToken = token; localStorage.setItem('feedback-query-token', token) },
    toggle(id: string) { const copy = new Set(this.selected); copy.has(id) ? copy.delete(id) : copy.add(id); this.selected = copy },
    clearSelection() { this.selected = new Set() }
  }
})
