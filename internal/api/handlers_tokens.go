package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"ebuild/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func CreateTokenHandler(db *sqlx.DB, signingKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)

		var req struct {
			Scopes []string `json:"scopes" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ttl := time.Hour * 24 * 365 * 10
		tokenStr, err := auth.NewToken(signingKey, claims.UserID, req.Scopes, ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
			return
		}
		hash := fmtHash(tokenStr)
		_, err = db.Exec(`INSERT INTO tokens (owner_user_id, token_hash, is_generated, scopes, created_at) VALUES (?, ?, 1, ?, datetime('now'))`, claims.UserID, hash, strings.Join(req.Scopes, ","))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, _ = db.Exec(`INSERT INTO token_audit (action, token_hash, owner_user_id, actor_user_id) VALUES (?, ?, ?, ?)`, "token_generated", hash, claims.UserID, claims.UserID)
		c.JSON(http.StatusOK, gin.H{"token": tokenStr})
	}
}

func RevokeTokenHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, hasClaims := c.Get(string(CtxClaims))
		var claims *auth.Claims
		if hasClaims {
			claims = ci.(*auth.Claims)
		}
		var req struct {
			Token string `json:"token"`
		}
		_ = c.BindJSON(&req)
		tokenToRevoke := req.Token
		if tokenToRevoke == "" {
			if ck, err := c.Request.Cookie("ebuild_refresh"); err == nil {
				tokenToRevoke = ck.Value
			}
		}
		if tokenToRevoke == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no token provided"})
			return
		}
		hash := fmtHash(tokenToRevoke)
		var ownerID int64
		err := db.Get(&ownerID, `SELECT owner_user_id FROM tokens WHERE token_hash = ? LIMIT 1`, hash)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		if hasClaims {
			rawTokIfc, _ := c.Get(string(CtxRawToken))
			rawTok, _ := rawTokIfc.(string)
			var callerRole string
			_ = db.Get(&callerRole, `SELECT role FROM users WHERE id = ? LIMIT 1`, claims.UserID)
			if ownerID != claims.UserID && req.Token != rawTok && callerRole != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "not allowed to revoke this token"})
				return
			}
		} else {
			ck, err := c.Request.Cookie("ebuild_refresh")
			if err != nil || ck == nil || ck.Value != tokenToRevoke {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or mismatched refresh cookie"})
				return
			}
			hdr := c.GetHeader("X-CSRF-Token")
			csrfCk, _ := c.Request.Cookie("ebuild_csrf")
			if csrfCk == nil || hdr == "" || csrfCk.Value != hdr {
				c.JSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
				return
			}
		}
		_, err = db.Exec(`UPDATE tokens SET revoked_at = datetime('now') WHERE token_hash = ?`, hash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		actorID := sql.NullInt64{}
		if hasClaims {
			actorID.Int64 = claims.UserID
			actorID.Valid = true
		}
		var actor interface{}
		if actorID.Valid {
			actor = actorID.Int64
		} else {
			actor = nil
		}
		_, _ = db.Exec(`INSERT INTO token_audit (action, token_hash, owner_user_id, actor_user_id) VALUES (?, ?, ?, ?)`, "token_revoked", hash, ownerID, actor)
		secure := os.Getenv("EBUILD_COOKIE_SECURE") == "1"
		domain := os.Getenv("EBUILD_COOKIE_DOMAIN")
		samesite := http.SameSiteStrictMode
		if os.Getenv("EBUILD_COOKIE_SAMESITE") == "Lax" {
			samesite = http.SameSiteLaxMode
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_refresh", Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: -1})
		http.SetCookie(c.Writer, &http.Cookie{Name: "ebuild_csrf", Value: "", Path: "/", HttpOnly: false, Secure: secure, SameSite: samesite, Domain: domain, MaxAge: -1})
		c.JSON(http.StatusOK, gin.H{"status": "revoked"})
	}
}

func fmtHash(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}
