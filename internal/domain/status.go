package domain

type FeedbackStatus string

const (
	StatusPendingAcceptance FeedbackStatus = "pending_acceptance"
	StatusNeedsInformation  FeedbackStatus = "needs_information"
	StatusInProgress        FeedbackStatus = "in_progress"
	StatusPendingConfirm    FeedbackStatus = "pending_confirmation"
	StatusClosed            FeedbackStatus = "closed"
	StatusRejected          FeedbackStatus = "rejected"
)

type transitionRule struct {
	From     FeedbackStatus
	To       FeedbackStatus
	Recovery bool
}

var transitionRules = []transitionRule{
	{From: StatusPendingAcceptance, To: StatusNeedsInformation},
	{From: StatusPendingAcceptance, To: StatusInProgress},
	{From: StatusPendingAcceptance, To: StatusRejected},
	{From: StatusNeedsInformation, To: StatusPendingAcceptance},
	{From: StatusNeedsInformation, To: StatusInProgress},
	{From: StatusNeedsInformation, To: StatusRejected},
	{From: StatusInProgress, To: StatusNeedsInformation},
	{From: StatusInProgress, To: StatusPendingConfirm},
	{From: StatusInProgress, To: StatusRejected},
	{From: StatusPendingConfirm, To: StatusInProgress},
	{From: StatusPendingConfirm, To: StatusClosed},
	{From: StatusClosed, To: StatusInProgress, Recovery: true},
	{From: StatusRejected, To: StatusPendingAcceptance, Recovery: true},
	{From: StatusRejected, To: StatusInProgress, Recovery: true},
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
	if !from.Valid() || !to.Valid() || from == to {
		return false
	}
	for _, rule := range transitionRules {
		if rule.From == from && rule.To == to {
			return true
		}
	}
	return false
}

func AllowedTransitions(status FeedbackStatus) []FeedbackStatus {
	if !status.Valid() {
		return nil
	}
	result := make([]FeedbackStatus, 0, 3)
	for _, rule := range transitionRules {
		if rule.From != status {
			continue
		}
		result = append(result, rule.To)
	}
	return result
}

func IsRecoveryTransition(from, to FeedbackStatus) bool {
	if !from.Valid() || !to.Valid() {
		return false
	}
	for _, rule := range transitionRules {
		if rule.From == from && rule.To == to {
			return rule.Recovery
		}
	}
	return false
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
