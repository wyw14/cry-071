package domain

import "slices"

type FeedbackStatus string

const (
	StatusPendingAcceptance FeedbackStatus = "pending_acceptance"
	StatusNeedsInformation  FeedbackStatus = "needs_information"
	StatusInProgress        FeedbackStatus = "in_progress"
	StatusPendingConfirm    FeedbackStatus = "pending_confirmation"
	StatusClosed            FeedbackStatus = "closed"
	StatusRejected          FeedbackStatus = "rejected"
)

var statusTransitions = map[FeedbackStatus][]FeedbackStatus{
	StatusPendingAcceptance: {StatusNeedsInformation, StatusInProgress, StatusRejected},
	StatusNeedsInformation:  {StatusPendingAcceptance, StatusInProgress, StatusRejected},
	StatusInProgress:        {StatusNeedsInformation, StatusPendingConfirm, StatusRejected},
	StatusPendingConfirm:    {StatusInProgress, StatusClosed},
	StatusClosed:            {StatusInProgress},
	StatusRejected:          {StatusPendingAcceptance},
}

func (s FeedbackStatus) Valid() bool {
	switch s {
	case StatusPendingAcceptance, StatusNeedsInformation, StatusInProgress,
		StatusPendingConfirm, StatusClosed, StatusRejected:
		return true
	default:
		return false
	}
}

func (s FeedbackStatus) Terminal() bool {
	return s == StatusClosed || s == StatusRejected
}

func (s FeedbackStatus) PublicLabel() string {
	switch s {
	case StatusPendingAcceptance:
		return "待受理"
	case StatusNeedsInformation:
		return "待补充"
	case StatusInProgress:
		return "处理中"
	case StatusPendingConfirm:
		return "待确认"
	case StatusClosed:
		return "已关闭"
	case StatusRejected:
		return "已驳回"
	default:
		return "未知状态"
	}
}

func CanTransition(from, to FeedbackStatus) bool {
	return slices.Contains(statusTransitions[from], to)
}

func AllowedTransitions(status FeedbackStatus) []FeedbackStatus {
	return slices.Clone(statusTransitions[status])
}

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

func (p Priority) SLAHours() int {
	switch p {
	case PriorityCritical:
		return 4
	case PriorityHigh:
		return 12
	case PriorityNormal:
		return 48
	case PriorityLow:
		return 96
	default:
		return 48
	}
}
