export type FeedbackStatus = 'pending_acceptance' | 'needs_information' | 'in_progress' | 'pending_confirmation' | 'closed' | 'rejected'
export type Priority = 'low' | 'normal' | 'high' | 'critical'

export interface Feedback {
  ID: string
  AcceptanceNumber: string
  AreaID: string
  FacilityCategoryID: string
  SubjectCode: string
  Priority: Priority
  Title: string
  Description: string
  Location: string
  Status: FeedbackStatus
  AssigneeID: string
  Version: number
  DueAt: string
}

export interface TimelineEvent {
  ID: string
  Kind: string
  Summary: string
  Details: Record<string, string>
  OccurredAt: string
}

export interface PublicFeedbackView {
  feedback: Feedback
  timeline: TimelineEvent[]
  replies: Array<{ ID: string; Content: string; CreatedAt: string }>
}

export interface ApiEnvelope<T> { data: T }
export interface ApiErrorEnvelope { error: { code: string; message: string; field_errors: Array<{ field: string; message: string }>; request_id: string } }
