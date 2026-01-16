package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"ebuild/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func AddArtifactHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgID := c.Param("id")
		ver := c.Param("ver")
		ci, exists := c.Get(string(CtxClaims))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims := ci.(*auth.Claims)
		var req struct {
			BlobURL  string `json:"blob_url" binding:"required"`
			Filename string `json:"filename"`
			Size     int64  `json:"size_bytes"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var versionID int64
		err := db.Get(&versionID, `SELECT id FROM package_versions WHERE package_id = ? AND version = ?`, pkgID, ver)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		pkgIDint, _ := strconv.ParseInt(pkgID, 10, 64)
		ok, err := isMaintainerOrAdmin(db, claims.UserID, pkgIDint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a maintainer"})
			return
		}
		res, err := db.Exec(`INSERT INTO artifacts (package_version_id, blob_url, filename, size_bytes, created_at) VALUES (?, ?, ?, ?, datetime('now'))`, versionID, req.BlobURL, req.Filename, req.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func ListArtifactsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgID := c.Param("id")
		ver := c.Param("ver")
		var versionID int64
		err := db.Get(&versionID, `SELECT id FROM package_versions WHERE package_id = ? AND version = ?`, pkgID, ver)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var arts []map[string]interface{}
		rows, err := db.Query(`SELECT id, blob_url, filename, size_bytes, created_at FROM artifacts WHERE package_version_id = ?`, versionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]interface{}, len(cols))
			valPtrs := make([]interface{}, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			m := map[string]interface{}{}
			for i, col := range cols {
				v := vals[i]
				if b, ok := v.([]byte); ok {
					m[col] = string(b)
				} else {
					m[col] = v
				}
			}
			arts = append(arts, m)
		}
		c.JSON(http.StatusOK, gin.H{"artifacts": arts})
	}
}

func DownloadArtifactHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		aID := c.Param("artifact_id")
		var art struct {
			ID               int64  `db:"id"`
			PackageVersionID int64  `db:"package_version_id"`
			BlobURL          string `db:"blob_url"`
			PackageID        int64  `db:"package_id"`
		}
		err := db.Get(&art, `SELECT a.id, a.package_version_id, a.blob_url, pv.package_id FROM artifacts a JOIN package_versions pv ON a.package_version_id = pv.id WHERE a.id = ?`, aID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
			return
		}
		_, _ = db.Exec(`UPDATE package_versions SET download_count = COALESCE(download_count,0) + 1 WHERE id = ?`, art.PackageVersionID)
		_, _ = db.Exec(`UPDATE packages SET download_count = COALESCE(download_count,0) + 1 WHERE id = ?`, art.PackageID)
		c.Redirect(http.StatusFound, art.BlobURL)
	}
}
