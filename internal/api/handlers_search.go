package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func SearchHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		category := c.Query("category")
		bucket := c.Query("bucket")
		// default to most_downloaded on main page
		sort := c.DefaultQuery("sort", "most_downloaded")
		var rows []map[string]interface{}
		limit := 50
		// Simple FTS5 search if q provided
		// see https://sqlite.org/fts5.html for details.
		if q != "" {
			// use parameterized query and the FTS alias 'f'
			base := "SELECT p.id, p.name, p.description, p.download_count FROM packages p JOIN packages_fts f ON p.id = f.rowid WHERE f MATCH ?"
			switch sort {
			case "most_downloaded":
				base += " ORDER BY p.download_count DESC"
			case "random":
				base += " ORDER BY RANDOM()"
			default:
				base += " ORDER BY p.created_at DESC"
			}
			base += " LIMIT ?"
			log.Printf("search fts query=%s args=[%s %d]", base, q, limit)
			rs, err := db.Query(base, q, limit)
			if err != nil {
				// if SQLite built without FTS5, fallback to LIKE search
				if strings.Contains(err.Error(), "fts5") || strings.Contains(err.Error(), "no such module") {
					likeQ := "%" + q + "%"
					alt := "SELECT p.id, p.name, p.description, p.download_count FROM packages p WHERE p.name LIKE ? OR p.description LIKE ?"
					switch sort {
					case "most_downloaded":
						alt += " ORDER BY p.download_count DESC"
					case "random":
						alt += " ORDER BY RANDOM()"
					default:
						alt += " ORDER BY p.created_at DESC"
					}
					alt += " LIMIT ?"
					log.Printf("fts unavailable, falling back to LIKE query=%s args=[%s %s %d]", alt, likeQ, likeQ, limit)
					rs, err = db.Query(alt, likeQ, likeQ, limit)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			defer rs.Close()
			cols, _ := rs.Columns()
			for rs.Next() {
				vals := make([]interface{}, len(cols))
				valPtrs := make([]interface{}, len(cols))
				for i := range vals {
					valPtrs[i] = &vals[i]
				}
				if err := rs.Scan(valPtrs...); err != nil {
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
				rows = append(rows, m)
			}
		} else {
			base := "SELECT id, name, description, download_count FROM packages"
			var clauses []string
			var args []interface{}
			if category != "" {
				clauses = append(clauses, "id IN (SELECT package_id FROM package_categories WHERE category_id = ?)")
				args = append(args, category)
			}
			if bucket != "" {
				clauses = append(clauses, "id IN (SELECT package_id FROM package_buckets WHERE bucket_id = ?)")
				args = append(args, bucket)
			}
			if len(clauses) > 0 {
				base = base + " WHERE " + strings.Join(clauses, " AND ")
			}
			switch sort {
			case "most_downloaded":
				base += " ORDER BY download_count DESC"
			case "random":
				base += " ORDER BY RANDOM()"
			default:
				base += " ORDER BY created_at DESC"
			}
			base += " LIMIT ?"
			args = append(args, limit)
			log.Printf("search base query=%s args=%v", base, args)
			rs, err := db.Query(base, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rs.Close()
			cols, _ := rs.Columns()
			for rs.Next() {
				vals := make([]interface{}, len(cols))
				valPtrs := make([]interface{}, len(cols))
				for i := range vals {
					valPtrs[i] = &vals[i]
				}
				if err := rs.Scan(valPtrs...); err != nil {
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
				rows = append(rows, m)
			}
		}
		c.JSON(http.StatusOK, gin.H{"results": rows})
	}
}
