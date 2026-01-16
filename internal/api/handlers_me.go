package api

import (
	"net/http"

	"ebuild/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func MeHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		var user struct {
			ID       int64  `db:"id" json:"id"`
			Username string `db:"username" json:"username"`
			Role     string `db:"role" json:"role"`
		}
		err := db.Get(&user, `SELECT id, username, role FROM users WHERE id = ? LIMIT 1`, claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
	}
}
