package router

import (
	"go-star/api"
	"go-star/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func CommentRouter(r *gin.RouterGroup) {
	app := api.App.CommentApi
	r.POST("articles/:id/comments", middleware.AuthMiddleware, middleware.RateLimitByUser("comment", 5, time.Minute), app.CommentCreateView)
	r.GET("articles/:id/comments", middleware.AuthMiddleware, app.CommentListView)
	r.DELETE("comments/:id", middleware.AuthMiddleware, app.CommentRemoveView)
}
