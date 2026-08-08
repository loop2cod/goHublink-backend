package routes

import (
	"github.com/gin-gonic/gin"

	"gohublink/backend/handlers"
)

func SpotRoutes(protected *gin.RouterGroup) {
	spots := protected.Group("/spots")
	{
		spots.POST("", handlers.CreateSpot)
		spots.GET("", handlers.ListSpots)
		spots.GET("/:id", handlers.GetSpot)
	}
}