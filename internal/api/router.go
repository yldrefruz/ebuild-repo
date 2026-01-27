package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func SetupRouter(db *sqlx.DB, signingKey []byte) *gin.Engine {
	r := gin.Default()
	// serve static test UI
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/static/index.html") })

	r.Use(AuthMiddleware(db, signingKey))
	//r.Use(CSRFMiddleware())

	// health
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	// auth
	r.POST("/register", RegisterHandler(db))
	r.POST("/login", LoginHandler(db, signingKey))
	r.POST("/tokens", CreateTokenHandler(db, signingKey))
	r.POST("/tokens/revoke", RevokeTokenHandler(db))
	r.POST("/refresh", RefreshHandler(db, signingKey))
	r.GET("/me", MeHandler(db))

	// packages
	r.POST("/packages", RequireScope("maintain"), CreatePackageHandler(db))
	r.GET("/packages/:id", GetPackageHandler(db))
	// versions
	r.POST("/packages/:id/versions", RequireScope("maintain"), CreateVersionHandler(db, signingKey))
	r.GET("/packages/:id/versions", ListVersionsHandler(db))
	r.GET("/packages/:id/versions/:ver", GetVersionHandler(db))

	// artifacts
	r.POST("/packages/:id/versions/:ver/artifacts", RequireScope("maintain"), AddArtifactHandler(db))
	r.GET("/packages/:id/versions/:ver/artifacts", ListArtifactsHandler(db))
	r.GET("/artifacts/:artifact_id/download", DownloadArtifactHandler(db))

	// votes
	r.POST("/packages/:id/votes", VoteHandler(db))

	// comments
	r.POST("/packages/:id/comments", CreateCommentHandler(db))
	r.GET("/packages/:id/comments", ListCommentsHandler(db))

	// search
	r.GET("/search", SearchHandler(db))

	return r
}
