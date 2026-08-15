package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

func currentNetworkCount() (int64, error) {
	var total int64
	err := db.DB.Model(&models.Customer{}).Count(&total).Error
	return total, err
}

// NetworkCount returns the total number of people who have joined the
// network (i.e. the number of recorded scans).
func NetworkCount(c *gin.Context) {
	total, err := currentNetworkCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": total,
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
	last, err := currentNetworkCount()
	if err != nil {
		return
	}
	send(last)

	// Keep the connection open; poll the DB and push on every change. Also
	// send a periodic heartbeat so proxies don't close the idle connection.
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	done := c.Request.Context().Done()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			current, err := currentNetworkCount()
			if err != nil {
				continue
			}
			if current != last {
				last = current
				send(current)
			}
		}
	}
}
