package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"ebuild/internal/auth"
	"ebuild/internal/models"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func CreatePackageHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)

		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		res, err := db.Exec(`INSERT INTO packages (name, description, created_by, token_required) VALUES (?, ?, ?, 1)`, req.Name, req.Description, claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func GetPackageHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var pkg models.Package
		err := db.Get(&pkg, `SELECT id, name, description, created_by, token_required, created_at FROM packages WHERE id = ?`, id)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var ver models.PackageVersion
		err = db.Get(&ver, `SELECT id, package_id, version, metadata, released_by, released_at, is_deprecated FROM package_versions WHERE package_id = ? ORDER BY released_at DESC LIMIT 1`, id)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"package": pkg, "latest_version": ver})
	}
}

func isMaintainerOrAdmin(db *sqlx.DB, userID int64, pkgID int64) (bool, error) {
	var role string
	if err := db.Get(&role, `SELECT role FROM users WHERE id = ?`, userID); err != nil {
		return false, err
	}
	if role == string(models.RoleAdmin) {
		return true, nil
	}
	var owner sql.NullInt64
	if err := db.Get(&owner, `SELECT created_by FROM packages WHERE id = ?`, pkgID); err != nil {
		return false, err
	}
	if owner.Valid && owner.Int64 == userID {
		return true, nil
	}
	var exists int
	err := db.Get(&exists, `SELECT 1 FROM package_maintainers WHERE package_id = ? AND user_id = ? LIMIT 1`, pkgID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

func CreateVersionHandler(db *sqlx.DB, signingKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgIDstr := c.Param("id")
		pkgID64, err := strconv.ParseInt(pkgIDstr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package id"})
			return
		}
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		ok, err := isMaintainerOrAdmin(db, claims.UserID, pkgID64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a maintainer"})
			return
		}
		var req struct {
			Version  string `json:"version" binding:"required"`
			Metadata string `json:"metadata"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := semver.NewVersion(req.Version); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semver"})
			return
		}
		res, err := db.Exec(`INSERT INTO package_versions (package_id, version, metadata, released_by, released_at) VALUES (?, ?, ?, ?, ?)`, pkgID64, req.Version, req.Metadata, claims.UserID, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}
