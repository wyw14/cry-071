package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/middleware"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"field_errors"`
	RequestID   string       `json:"request_id"`
}

func respond(c *gin.Context, status int, data any) { c.JSON(status, gin.H{"data": data}) }

func respondError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "服务暂时不可用"
	fields := []FieldError{}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "not_found", "请求的资源不存在"
	case errors.Is(err, domain.ErrForbidden):
		status, code, message = 403, "permission_denied", "当前身份没有操作权限"
	case errors.Is(err, domain.ErrTokenInvalid):
		status, code, message = 401, "query_token_invalid", "查询令牌无效或已过期"
	case errors.Is(err, domain.ErrVersionConflict):
		status, code, message = 409, "version_conflict", "数据已被其他操作更新，请刷新后重试"
	case errors.Is(err, domain.ErrConflict):
		status, code, message = 409, "conflict", "当前操作与已有数据冲突"
	case errors.Is(err, domain.ErrInvalidTransition):
		status, code, message = 422, "invalid_transition", "当前状态不允许该操作"
	case errors.Is(err, domain.ErrInvalidArgument):
		status, code, message = 422, "invalid_argument", "请求参数不正确"
	default:
		var validation domain.ValidationError
		if errors.As(err, &validation) {
			status, code, message = 422, "validation_failed", "请求内容未通过校验"
			fields = append(fields, FieldError{Field: validation.Field, Message: validation.Message})
		}
		var validatorErrors validator.ValidationErrors
		if errors.As(err, &validatorErrors) {
			status, code, message = 422, "validation_failed", "请求内容未通过校验"
			for _, item := range validatorErrors {
				fields = append(fields, FieldError{Field: item.Field(), Message: validationMessage(item)})
			}
		}
	}
	c.JSON(status, gin.H{"error": ErrorBody{Code: code, Message: message, FieldErrors: fields, RequestID: middleware.GetRequestID(c)}})
}

func validationMessage(field validator.FieldError) string {
	switch field.Tag() {
	case "required":
		return "此字段不能为空"
	case "min":
		return "内容长度不足"
	case "max":
		return "内容长度超过限制"
	default:
		return "字段格式不正确"
	}
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		respondError(c, err)
		return false
	}
	return true
}
