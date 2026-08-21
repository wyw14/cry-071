package domain

import "fmt"

type FaultCode uint8

const (
	FaultMissing FaultCode = iota + 1
	FaultCollision
	FaultPermission
	FaultTransition
	FaultInput
	FaultQueryCredential
	FaultStaleVersion
)

type Fault struct {
	Code   FaultCode
	Public string
}

func (f *Fault) Error() string { return f.Public }

func (f *Fault) Is(target error) bool {
	other, ok := target.(*Fault)
	return ok && f.Code == other.Code
}

var (
	ErrNotFound          = &Fault{Code: FaultMissing, Public: "resource not found"}
	ErrConflict          = &Fault{Code: FaultCollision, Public: "resource conflict"}
	ErrForbidden         = &Fault{Code: FaultPermission, Public: "operation forbidden"}
	ErrInvalidTransition = &Fault{Code: FaultTransition, Public: "invalid feedback transition"}
	ErrInvalidArgument   = &Fault{Code: FaultInput, Public: "invalid argument"}
	ErrTokenInvalid      = &Fault{Code: FaultQueryCredential, Public: "query token invalid"}
	ErrVersionConflict   = &Fault{Code: FaultStaleVersion, Public: "version conflict"}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type StateError struct {
	From FeedbackStatus
	To   FeedbackStatus
}

func (e StateError) Error() string {
	return fmt.Sprintf("cannot move feedback from %s to %s", e.From, e.To)
}

func (StateError) Unwrap() error { return ErrInvalidTransition }

type ConflictError struct {
	Resource string
	Key      string
	Cause    error
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("%s %q conflicts with current state", e.Resource, e.Key)
}

func (e ConflictError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrConflict
}
