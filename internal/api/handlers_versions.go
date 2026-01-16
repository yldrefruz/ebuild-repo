package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func ListVersionsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgID := c.Param("id")
		var versions []map[string]interface{}
		rows, err := db.Query(`SELECT id, version, metadata, released_by, released_at, is_deprecated FROM package_versions WHERE package_id = ? ORDER BY released_at DESC`, pkgID)
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
			versions = append(versions, m)
		}
		c.JSON(http.StatusOK, gin.H{"versions": versions})
	}
}

func GetVersionHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgID := c.Param("id")
		ver := c.Param("ver")
		rows, err := db.Query(`SELECT id, package_id, version, metadata, released_by, released_at, is_deprecated FROM package_versions WHERE package_id = ? AND version = ? LIMIT 1`, pkgID, ver)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		if !rows.Next() {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
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
		c.JSON(http.StatusOK, gin.H{"version": m})
	}
}
