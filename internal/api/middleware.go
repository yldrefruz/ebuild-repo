package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"ebuild/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type ctxKey string

const (
	CtxClaims   ctxKey = "claims"
	CtxRawToken ctxKey = "raw_token"
)

func AuthMiddleware(db *sqlx.DB, signingKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if authz == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokenRaw := parts[1]
		claims, err := auth.ParseToken(signingKey, tokenRaw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		hash := sha256.Sum256([]byte(tokenRaw))
		hashS := hex.EncodeToString(hash[:])
		var revokedAt sql.NullString
		err = db.Get(&revokedAt, `SELECT revoked_at FROM tokens WHERE token_hash = ? LIMIT 1`, hashS)
		if err == nil && revokedAt.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			return
		}
		c.Set(string(CtxClaims), claims)
		c.Set(string(CtxRawToken), tokenRaw)
		c.Next()
	}
}

func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		for _, s := range claims.Scopes {
			if s == scope {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
	}
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m != "POST" && m != "PUT" && m != "DELETE" && m != "PATCH" {
			c.Next()
			return
		}
		p := c.Request.URL.Path
		if p == "/login" || p == "/register" || p == "/refresh" || p == "/tokens/revoke" || strings.HasPrefix(p, "/static/") || p == "/health" {
			c.Next()
			return
		}
		if c.GetHeader("Authorization") != "" {
			c.Next()
			return
		}
		hdr := c.GetHeader("X-CSRF-Token")
		ck, err := c.Request.Cookie("ebuild_csrf")
		if err != nil || ck == nil || hdr == "" || ck.Value != hdr {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
			return
		}
		c.Next()
	}
}
