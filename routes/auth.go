package routes

import (
	"github.com/gin-gonic/gin"

	"gohublink/backend/handlers"
)

func AuthRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", handlers.Login)
}