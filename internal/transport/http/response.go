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
	presentation := classifyError(err)
	middleware.ReportError(c, err, presentation.status, presentation.code)
	body := ErrorBody{
		Code:        presentation.code,
		Message:     presentation.message,
		FieldErrors: presentation.fields,
		RequestID:   middleware.GetRequestID(c),
	}
	c.JSON(presentation.status, gin.H{"error": body})
}

type errorPresentation struct {
	status  int
	code    string
	message string
	fields  []FieldError
}

func classifyError(err error) errorPresentation {
	presentation := errorPresentation{
		status:  http.StatusInternalServerError,
		code:    "internal_error",
		message: "服务暂时不可用",
		fields:  []FieldError{},
	}
	// errors.Is unwraps nested domain errors, so wrapped sentinels such as
	// ConflictError (-> ErrConflict) and StateError (-> ErrInvalidTransition)
	// map to the same stable code as their sentinel instead of falling through
	// to the generic 500 response.
	switch {
	case errors.Is(err, domain.ErrNotFound):
		presentation.status = http.StatusNotFound
		presentation.code = "not_found"
		presentation.message = "请求的资源不存在"
	case errors.Is(err, domain.ErrForbidden):
		presentation.status = http.StatusForbidden
		presentation.code = "permission_denied"
		presentation.message = "当前身份没有操作权限"
	case errors.Is(err, domain.ErrTokenInvalid):
		presentation.status = http.StatusUnauthorized
		presentation.code = "query_token_invalid"
		presentation.message = "查询令牌无效或已过期"
	case errors.Is(err, domain.ErrVersionConflict):
		presentation.status = http.StatusConflict
		presentation.code = "version_conflict"
		presentation.message = "数据已被其他操作更新，请刷新后重试"
	case errors.Is(err, domain.ErrConflict):
		presentation.status = http.StatusConflict
		presentation.code = "conflict"
		presentation.message = "当前操作与已有数据冲突"
	case errors.Is(err, domain.ErrInvalidTransition):
		presentation.status = http.StatusUnprocessableEntity
		presentation.code = "invalid_transition"
		presentation.message = "当前状态不允许该操作"
	case errors.Is(err, domain.ErrInvalidArgument):
		presentation.status = http.StatusUnprocessableEntity
		presentation.code = "invalid_argument"
		presentation.message = "请求参数不正确"
	}

	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		presentation.status = http.StatusUnprocessableEntity
		presentation.code = "validation_failed"
		presentation.message = "请求内容未通过校验"
		presentation.fields = append(presentation.fields, FieldError{
			Field:   validationErr.Field,
			Message: validationErr.Message,
		})
	}
	var validatorErrs validator.ValidationErrors
	if errors.As(err, &validatorErrs) {
		presentation.status = http.StatusUnprocessableEntity
		presentation.code = "validation_failed"
		presentation.message = "请求内容未通过校验"
		for _, item := range validatorErrs {
			presentation.fields = append(presentation.fields, FieldError{
				Field:   item.Field(),
				Message: validationMessage(item),
			})
		}
	}
	return presentation
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
