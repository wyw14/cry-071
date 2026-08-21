package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-071/internal/domain"
)

const ActorKey = "actor"

func Actor() gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := domain.Actor{
			ID:          strings.TrimSpace(c.GetHeader("X-Actor-ID")),
			DisplayName: strings.TrimSpace(c.GetHeader("X-Actor-Name")),
			Role:        domain.Role(strings.TrimSpace(c.GetHeader("X-Actor-Role"))),
			AreaIDs:     splitAreas(c.GetHeader("X-Actor-Areas")),
		}
		if actor.ID != "" && actor.Role != "" {
			c.Set(ActorKey, actor)
		}
		c.Next()
	}
}

func CurrentActor(c *gin.Context) (domain.Actor, bool) {
	value, exists := c.Get(ActorKey)
	if !exists {
		return domain.Actor{}, false
	}
	actor, ok := value.(domain.Actor)
	return actor, ok
}

func RequireRoles(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, exists := CurrentActor(c)
		if !exists {
			abortAuth(c, 401, "authentication_required", "需要提供操作者身份")
			return
		}
		for _, role := range roles {
			if actor.Role == role {
				c.Next()
				return
			}
		}
		abortAuth(c, 403, "permission_denied", "当前角色没有执行该操作的权限")
	}
}

func splitAreas(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func abortAuth(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "field_errors": []any{}, "request_id": GetRequestID(c)}})
}
