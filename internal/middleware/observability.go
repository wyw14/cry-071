package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("http_request",
			zap.String("request_id", GetRequestID(c)), zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()), zap.Int("status", c.Writer.Status()),
			zap.Int64("bytes", int64(c.Writer.Size())), zap.Duration("duration", time.Since(started)),
		)
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
