package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

// whatsappPhoneNumberID returns the business phone number id used as the
// "from" for outbound messages. It reads WHATSAPP_PHONE_NUMBER_ID, falling
// back to the id embedded in WHATSAPP_API_URL if present.
func whatsappPhoneNumberID() string {
	if id := strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID")); id != "" {
		return id
	}
	apiURL := strings.TrimSpace(os.Getenv("WHATSAPP_API_URL"))
	// WHATSAPP_API_URL typically looks like:
	// https://graph.facebook.com/v25.0/<phone_number_id>/messages
	segments := strings.Split(strings.Trim(apiURL, "/"), "/")
	if len(segments) >= 2 {
		return segments[len(segments)-2]
	}
	return ""
}

// ListCustomerConversation returns every WhatsApp message exchanged with a
// given customer phone number (inbound and outbound, including templates),
// oldest first, so the UI can render a chat thread.
func ListCustomerConversation(c *gin.Context) {
	phone := strings.TrimSpace(c.Param("phone"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number is required"})
		return
	}

	var messages []models.WhatsAppMessage
	if err := db.DB.
		Where("from = ? OR to = ?", phone, phone).
		Order("received_at ASC").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"phone":    phone,
		"messages": messages,
	})
}

type sendMessageRequest struct {
	To   string `json:"to" binding:"required"`
	Text string `json:"text" binding:"required"`
}

// SendWhatsAppMessage sends a free-form text message to a customer via the
// WhatsApp Business Cloud API and records it as an outbound message.
func SendWhatsAppMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to and text are required"})
		return
	}
	req.To = strings.TrimSpace(req.To)
	req.Text = strings.TrimSpace(req.Text)
	if req.To == "" || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to and text are required"})
		return
	}

	apiURL := strings.TrimSpace(os.Getenv("WHATSAPP_API_URL"))
	accessToken := strings.TrimSpace(os.Getenv("WHATSAPP_ACCESS_TOKEN"))
	from := whatsappPhoneNumberID()

	if apiURL == "" || accessToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WhatsApp API not configured"})
		return
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                req.To,
		"type":              "text",
		"text":              map[string]string{"body": req.Text},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach WhatsApp API: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var messageID string
	var apiErr string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(respBody, &result); err == nil && len(result.Messages) > 0 {
			messageID = result.Messages[0].ID
		}
	} else {
		apiErr = strings.TrimSpace(string(respBody))
		if apiErr == "" {
			apiErr = fmt.Sprintf("WhatsApp API returned %d", resp.StatusCode)
		}
	}

	textBody := req.Text
	now := time.Now()
	waMsg := models.WhatsAppMessage{
		MessageID:  messageID,
		From:       from,
		To:         req.To,
		Type:       models.WhatsAppMessageTypeText,
		Direction:  models.WhatsAppMessageDirectionOutbound,
		TextBody:   &textBody,
		Status:     "sent",
		ReceivedAt: now,
	}

	if apiErr != "" {
		waMsg.Status = "failed"
		waMsg.ErrorMessage = &apiErr
	}

	if err := db.DB.Create(&waMsg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, waMsg)
}
