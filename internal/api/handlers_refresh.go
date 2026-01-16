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

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RefreshHandler(db *sqlx.DB, signingKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("ebuild_refresh")
		if err != nil || cookie.Value == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh cookie"})
			return
		}
		refreshTok := cookie.Value
		hdr := c.GetHeader("X-CSRF-Token")
		ck, _ := c.Request.Cookie("ebuild_csrf")
		if ck == nil || hdr == "" || ck.Value != hdr {
			c.JSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
			return
		}
		claims, err := auth.ParseToken(signingKey, refreshTok)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		h := sha256.Sum256([]byte(refreshTok))
		hS := hex.EncodeToString(h[:])
		var revoked sql.NullString
		err = db.Get(&revoked, `SELECT revoked_at FROM tokens WHERE token_hash = ? LIMIT 1`, hS)
		if err != nil || revoked.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token revoked or not found"})
			return
		}
		accessTok, err := auth.NewToken(signingKey, claims.UserID, []string{"read", "maintain"}, time.Minute*30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
			return
		}
		newRefresh, err := auth.NewToken(signingKey, claims.UserID, []string{"refresh"}, time.Hour*24*30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
			return
		}
		hOld := sha256.Sum256([]byte(refreshTok))
		oldHash := hex.EncodeToString(hOld[:])
		hNew := sha256.Sum256([]byte(newRefresh))
		newHash := hex.EncodeToString(hNew[:])
		_, _ = db.Exec(`UPDATE tokens SET revoked_at = datetime('now') WHERE token_hash = ?`, oldHash)
		_, err = db.Exec(`INSERT INTO tokens (owner_user_id, token_hash, is_generated, scopes, parent_token_hash, created_at) VALUES (?, ?, 0, ?, ?, datetime('now'))`, claims.UserID, newHash, "refresh", oldHash)
		if err != nil {
			// non-fatal
		}
		_, _ = db.Exec(`INSERT INTO token_audit (action, token_hash, owner_user_id, actor_user_id, parent_token_hash) VALUES (?, ?, ?, ?, ?)`, "refresh_revoked", oldHash, claims.UserID, claims.UserID, nil)
		_, _ = db.Exec(`INSERT INTO token_audit (action, token_hash, owner_user_id, actor_user_id, parent_token_hash) VALUES (?, ?, ?, ?, ?)`, "refresh_created", newHash, claims.UserID, claims.UserID, oldHash)
		secure := os.Getenv("EBUILD_COOKIE_SECURE") == "1"
		domain := os.Getenv("EBUILD_COOKIE_DOMAIN")
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate csrf"})
			return
		}
		csrf := hex.EncodeToString(b)
		samesite := http.SameSiteStrictMode
		if os.Getenv("EBUILD_COOKIE_SAMESITE") == "Lax" {
			samesite = http.SameSiteLaxMode
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_refresh", Value: newRefresh, Path: "/", HttpOnly: true, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: 60 * 60 * 24 * 30})
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_csrf", Value: csrf, Path: "/", HttpOnly: false, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: 60 * 60 * 24 * 30})
		var username string
		_ = db.Get(&username, `SELECT username FROM users WHERE id = ? LIMIT 1`, claims.UserID)
		c.JSON(http.StatusOK, gin.H{"token": accessTok, "username": username, "expires_at": time.Now().Add(time.Minute * 30).UTC(), "csrf": csrf})
	}
}
