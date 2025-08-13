package router

import (
	"bdo_calc_go/internal/handler"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	UserHandler *handler.UserHandler
}

func Register(r *gin.Engine, d Dependencies) {
	// 공용 라우트
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// v1 그룹
	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", d.UserHandler.Create)
			users.GET("/:id", d.UserHandler.GetByID)
			users.GET("", d.UserHandler.List)
		}
	}
}
