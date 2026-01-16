package api

import (
	"ebuild/internal/auth"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func VoteHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		pkgID := c.Param("id")
		var req struct {
			Value int `json:"value" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, err := db.Exec(`INSERT INTO votes (user_id, package_id, value, created_at) VALUES (?, ?, ?, datetime('now')) ON CONFLICT(user_id, package_id) DO UPDATE SET value = excluded.value, created_at = datetime('now')`, claims.UserID, pkgID, req.Value)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
