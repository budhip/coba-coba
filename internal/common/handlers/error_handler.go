package handlers

import (
	"context"
	"fmt"

	xlog "bitbucket.org/Amartha/go-x/log"
)

// ErrorHandler centralizes error handling logic
type ErrorHandler struct {
	logPrefix string
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logPrefix string) *ErrorHandler {
	return &ErrorHandler{
		logPrefix: logPrefix,
	}
}

// ErrorType defines the type of error for different handling strategies
type ErrorType int

const (
	ErrorTypeSkippable ErrorType = iota // Error that should skip message without DLQ
	ErrorTypeDLQ                        // Error that should go to DLQ
)

// ErrorResult contains error handling result
type ErrorResult struct {
	Err        error
	ShouldSkip bool
}

// HandleError processes error and returns appropriate result
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, errType ErrorType, logFields []xlog.Field) ErrorResult {
	if err == nil {
		return ErrorResult{Err: nil, ShouldSkip: false}
	}

	logFields = append(logFields, xlog.Err(err))

	switch errType {
	case ErrorTypeSkippable:
		xlog.Info(ctx, eh.logPrefix, append(logFields, xlog.String("action", "skipped"))...)
		return ErrorResult{Err: err, ShouldSkip: true}
	case ErrorTypeDLQ:
		xlog.Warn(ctx, eh.logPrefix, append(logFields, xlog.String("action", "sent_to_dlq"))...)
		return ErrorResult{Err: err, ShouldSkip: false}
	default:
		xlog.Warn(ctx, eh.logPrefix, logFields...)
		return ErrorResult{Err: err, ShouldSkip: false}
	}
}

// WrapError wraps error with additional context
func (eh *ErrorHandler) WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// LogAndReturnError logs error and returns it
func (eh *ErrorHandler) LogAndReturnError(ctx context.Context, err error, logFields []xlog.Field) error {
	if err == nil {
		return nil
	}
	xlog.Error(ctx, eh.logPrefix, append(logFields, xlog.Err(err))...)
	return err
}
