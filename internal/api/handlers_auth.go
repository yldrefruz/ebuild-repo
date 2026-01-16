package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"ebuild/internal/auth"
	"ebuild/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func hashTokenRaw(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func RegisterHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		pw, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		res, err := db.Exec(`INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)`, req.Username, req.Email, string(pw), models.RoleMaintainer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func LoginHandler(db *sqlx.DB, signingKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var user models.User
		err := db.Get(&user, `SELECT id, username, email, password_hash, role FROM users WHERE username = ?`, req.Username)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		accessTok, err := auth.NewToken(signingKey, user.ID, []string{"read", "maintain"}, time.Minute*30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
			return
		}
		refreshTok, err := auth.NewToken(signingKey, user.ID, []string{"refresh"}, time.Hour*24*30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
			return
		}
		hash := hashTokenRaw(refreshTok)
		_, err = db.Exec(`INSERT INTO tokens (owner_user_id, token_hash, is_generated, scopes, created_at) VALUES (?, ?, 0, ?, datetime('now'))`, user.ID, hash, "refresh")
		if err != nil {
			// non-fatal
		}
		_, _ = db.Exec(`INSERT INTO token_audit (action, token_hash, owner_user_id, actor_user_id, parent_token_hash, meta) VALUES (?, ?, ?, ?, ?, ?)`, "refresh_created", hash, user.ID, user.ID, nil, nil)
		secure := os.Getenv("EBUILD_COOKIE_SECURE") == "1"
		domain := os.Getenv("EBUILD_COOKIE_DOMAIN")
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate csrf"})
			return
		}
		csrf := hex.EncodeToString(b)
		// SameSite: default Strict unless EBUILD_COOKIE_SAMESITE="Lax"
		samesite := http.SameSiteStrictMode
		if os.Getenv("EBUILD_COOKIE_SAMESITE") == "Lax" {
			samesite = http.SameSiteLaxMode
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_refresh", Value: refreshTok, Path: "/", HttpOnly: true, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: 60 * 60 * 24 * 30})
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_csrf", Value: csrf, Path: "/", HttpOnly: false, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: 60 * 60 * 24 * 30})
		c.JSON(http.StatusOK, gin.H{"token": accessTok, "username": user.Username, "csrf": csrf})
	}
}
