package handler

import "github.com/gin-gonic/gin"

// WhoAmI验证用户是谁，并返回用户iD
func (h *UserHandler) WhoAmI(c *gin.Context) {
	Success(c, gin.H{"user_id": c.GetInt64("userID")})
}
