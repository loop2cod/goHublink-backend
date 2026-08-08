package routes

import (
	"github.com/gin-gonic/gin"

	"gohublink/backend/auth"
	"gohublink/backend/handlers"
)

func Setup(r *gin.Engine) {
	r.GET("/qr/:spot_id", handlers.QRRedirect)
	r.POST("/api/auth/login", handlers.Login)
	r.POST("/api/auth/refresh", handlers.RefreshToken)
	r.POST("/api/auth/logout", handlers.Logout)

	protected := r.Group("/api", auth.Middleware())
	{
		protected.POST("/spots", handlers.CreateSpot)
		protected.GET("/spots", handlers.ListSpots)
		protected.GET("/spots/:id", handlers.GetSpot)
		protected.GET("/scans", handlers.ListScans)
		protected.GET("/scans/:id", handlers.GetScan)
	}
}