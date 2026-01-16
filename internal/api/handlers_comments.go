package api

import (
	"ebuild/internal/auth"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func CreateCommentHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		pkgID := c.Param("id")
		var req struct {
			Body           string  `json:"body" binding:"required"`
			PackageVersion *string `json:"package_version_id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var pvID interface{}
		if req.PackageVersion != nil {
			pvID = req.PackageVersion
		} else {
			pvID = nil
		}
		_, err := db.Exec(`INSERT INTO comments (user_id, package_id, package_version_id, body, created_at) VALUES (?, ?, ?, ?, datetime('now'))`, claims.UserID, pkgID, pvID, req.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "ok"})
	}
}

func ListCommentsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgID := c.Param("id")
		var comments []map[string]interface{}
		if err := db.Select(&comments, `SELECT id, user_id, package_version_id, body, created_at FROM comments WHERE package_id = ? ORDER BY created_at DESC`, pkgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"comments": comments})
	}
}
