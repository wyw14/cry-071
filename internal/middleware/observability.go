package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type errorReport struct {
	err    error
	status int
	code   string
}

const errorReportKey = "error_report"

// ReportError records the classified error so the request log can surface the
// underlying cause (e.g. an idempotency-key conflict) alongside the request_id,
// which is essential for correlating failed requests during troubleshooting.
func ReportError(c *gin.Context, err error, status int, code string) {
	if err == nil {
		return
	}
	c.Set(errorReportKey, errorReport{err: err, status: status, code: code})
}

func ErrorReport(c *gin.Context) (error, string, int, bool) {
	value, ok := c.Get(errorReportKey)
	if !ok {
		return nil, "", 0, false
	}
	report, ok := value.(errorReport)
	if !ok {
		return nil, "", 0, false
	}
	return report.err, report.code, report.status, true
}

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		fields := []zap.Field{
			zap.String("request_id", GetRequestID(c)), zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()), zap.Int("status", c.Writer.Status()),
			zap.Int64("bytes", int64(c.Writer.Size())), zap.Duration("duration", time.Since(started)),
		}
		if err, code, status, ok := ErrorReport(c); ok {
			fields = append(fields, zap.Int("error_status", status), zap.String("error_code", code),
				zap.String("error", err.Error()))
			if status >= http.StatusInternalServerError {
				logger.Error("http_request_error", fields...)
			} else {
				logger.Info("http_request_error", fields...)
			}
			return
		}
		logger.Info("http_request", fields...)
	}
}

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("http_panic", zap.Any("panic", recovered), zap.ByteString("stack", debug.Stack()), zap.String("request_id", GetRequestID(c)))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{
					"code": "internal_error", "message": "服务暂时不可用", "field_errors": []any{}, "request_id": GetRequestID(c),
				}})
			}
		}()
		c.Next()
	}
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
