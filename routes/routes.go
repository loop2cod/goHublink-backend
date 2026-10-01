package routes

import (
	"github.com/gin-gonic/gin"

	"gohublink/backend/auth"
	"gohublink/backend/handlers"
)

func Setup(r *gin.Engine) {
	r.GET("/qr/:spot_id", handlers.QRRedirect)
	r.GET("/api/network/count", handlers.NetworkCount)
	r.GET("/api/network/count/stream", handlers.NetworkCountStream)
	r.POST("/api/auth/login", handlers.Login)
	r.POST("/api/auth/refresh", handlers.RefreshToken)
	r.POST("/api/auth/logout", handlers.Logout)
	r.POST("/api/auth/claim", handlers.ClaimScanToken)
	r.POST("/api/auth/verify", handlers.VerifyUserToken)

	r.GET("/webhook/whatsapp", handlers.VerifyWebhook)
	r.POST("/webhook/whatsapp", handlers.ReceiveWhatsAppWebhook)

	protected := r.Group("/api", auth.Middleware())
	{
		protected.POST("/spots", handlers.CreateSpot)
		protected.GET("/spots", handlers.ListSpots)
		protected.GET("/spots/:id", handlers.GetSpot)

		protected.GET("/cities", handlers.ListCities)
		protected.POST("/cities", handlers.CreateCity)
		protected.GET("/cities/:id", handlers.GetCity)
		protected.PUT("/cities/:id", handlers.UpdateCity)
		protected.DELETE("/cities/:id", handlers.DeleteCity)
		protected.POST("/cities/:id/divisions", handlers.CreateDivision)
		protected.PUT("/cities/divisions/:division_id", handlers.UpdateDivision)
		protected.DELETE("/cities/divisions/:division_id", handlers.DeleteDivision)
		protected.GET("/scans", handlers.ListScans)
		protected.GET("/scans/stats", handlers.ScanStats)
		protected.GET("/scans/by-token/:token", handlers.GetScanByToken)
		protected.GET("/scans/:id", handlers.GetScan)

		protected.GET("/customers", handlers.ListCustomers)
		protected.GET("/customers/:phone", handlers.GetCustomer)

		protected.GET("/whatsapp/messages", handlers.ListWhatsAppMessages)
		protected.GET("/whatsapp/messages/:id", handlers.GetWhatsAppMessage)
		protected.GET("/whatsapp/conversation/:phone", handlers.ListCustomerConversation)
		protected.POST("/whatsapp/send", handlers.SendWhatsAppMessage)
	}

	userProtected := r.Group("/api", auth.UserMiddleware())
	{
		userProtected.GET("/auth/me", handlers.GetCurrentUser)
		userProtected.GET("/wallet", handlers.GetWallet)
		userProtected.GET("/wallet/transactions", handlers.GetWalletTransactions)
		userProtected.GET("/offers", handlers.ListOffers)
		userProtected.GET("/offers/claimed", handlers.ListClaimedOffers)
		userProtected.GET("/offers/:id", handlers.GetOffer)
		userProtected.POST("/offers/:id/claim", handlers.ClaimOffer)
	}
}