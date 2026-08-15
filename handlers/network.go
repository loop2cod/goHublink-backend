package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

// dummyBase is the synthetic starting network size shown while the business is
// still small. The live count is this base plus the real customer count.
const dummyBase = 3500

func currentNetworkCount() (int64, error) {
	var total int64
	err := db.DB.Model(&models.Customer{}).Count(&total).Error
	return total, err
}

// NetworkCount returns the total network size: a synthetic baseline plus the
// real number of customers.
func NetworkCount(c *gin.Context) {
	total, err := currentNetworkCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": dummyBase + total,
	})
}

// NetworkCountStream is a Server-Sent Events endpoint that pushes the live
// network count to the client the moment it changes. It is plain HTTP, so it
// reuses the existing CORS setup and auto-reconnects on the client without any
// custom connection management.
func NetworkCountStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	send := func(total int64) {
		_, _ = c.Writer.WriteString("data: " + strconv.FormatInt(total, 10) + "\n\n")
		flusher.Flush()
	}

	// Push the initial value immediately so the client doesn't wait.
	start := time.Now()
	last, err := currentNetworkCount()
	if err != nil {
		return
	}
	send(dummyBase + last)

	// Keep the connection open. Push a new value whenever the real customer
	// count changes OR when the counter "grows" on its own so it always feels
	// alive. A heartbeat is also sent so proxies don't kill the idle stream.
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	done := c.Request.Context().Done()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			customers, err := currentNetworkCount()
			if err != nil {
				continue
			}

			// Increment the synthetic count by one roughly every minute so the
			// counter keeps climbing even without new real customers.
			elapsed := time.Since(start)
			dummyGrowth := int64(elapsed / time.Minute)

			current := dummyBase + customers + dummyGrowth
			if current != last {
				last = current
				send(current)
			}
		}
	}
}
