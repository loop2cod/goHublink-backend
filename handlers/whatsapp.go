package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

type WhatsAppWebhookEntry struct {
	ID      string                  `json:"id"`
	Changes []WhatsAppWebhookChange `json:"changes"`
}

type WhatsAppWebhookChange struct {
	Value WhatsAppWebhookValue `json:"value"`
	Field string               `json:"field"`
}

type WhatsAppWebhookValue struct {
	MessagingProduct string                   `json:"messaging_product"`
	Metadata         WhatsAppWebhookMetadata  `json:"metadata"`
	Contacts         []WhatsAppWebhookContact `json:"contacts"`
	Messages         []WhatsAppWebhookMessage `json:"messages"`
	Statuses         []WhatsAppWebhookStatus  `json:"statuses"`
}

type WhatsAppWebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WhatsAppWebhookContact struct {
	Profile WhatsAppWebhookProfile `json:"profile"`
	WaID    string                 `json:"wa_id"`
}

type WhatsAppWebhookProfile struct {
	Name string `json:"name"`
}

type WhatsAppWebhookMessage struct {
	ID          string                       `json:"id"`
	From        string                       `json:"from"`
	To          string                       `json:"to"`
	Timestamp   string                       `json:"timestamp"`
	Type        string                       `json:"type"`
	Text        *WhatsAppWebhookText         `json:"text,omitempty"`
	Image       *WhatsAppWebhookMedia        `json:"image,omitempty"`
	Audio       *WhatsAppWebhookMedia        `json:"audio,omitempty"`
	Video       *WhatsAppWebhookMedia        `json:"video,omitempty"`
	Document    *WhatsAppWebhookMedia        `json:"document,omitempty"`
	Sticker     *WhatsAppWebhookMedia        `json:"sticker,omitempty"`
	Location    *WhatsAppWebhookLocation     `json:"location,omitempty"`
	Contacts    []WhatsAppWebhookContactData `json:"contacts,omitempty"`
	Interactive *WhatsAppWebhookInteractive  `json:"interactive,omitempty"`
	Reaction    *WhatsAppWebhookReaction     `json:"reaction,omitempty"`
	Context     *WhatsAppWebhookContext      `json:"context,omitempty"`
}

type WhatsAppWebhookText struct {
	Body string `json:"body"`
}

type WhatsAppWebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type WhatsAppWebhookLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type WhatsAppWebhookContactData struct {
	Name      WhatsAppWebhookContactName `json:"name"`
	Phones    []WhatsAppWebhookPhone     `json:"phones,omitempty"`
	Emails    []WhatsAppWebhookEmail     `json:"emails,omitempty"`
	Addresses []WhatsAppWebhookAddress   `json:"addresses,omitempty"`
	Urls      []WhatsAppWebhookURL       `json:"urls,omitempty"`
	Birthday  string                     `json:"birthday,omitempty"`
	Org       WhatsAppWebhookOrg         `json:"org,omitempty"`
}

type WhatsAppWebhookContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

type WhatsAppWebhookPhone struct {
	Phone string `json:"phone"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

type WhatsAppWebhookEmail struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"`
}

type WhatsAppWebhookAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

type WhatsAppWebhookURL struct {
	URL  string `json:"url"`
	Type string `json:"type,omitempty"`
}

type WhatsAppWebhookOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type WhatsAppWebhookInteractive struct {
	Type        string                            `json:"type"`
	Header      *WhatsAppWebhookInteractiveHeader `json:"header,omitempty"`
	Body        *WhatsAppWebhookInteractiveBody   `json:"body,omitempty"`
	Footer      *WhatsAppWebhookInteractiveFooter `json:"footer,omitempty"`
	Action      *WhatsAppWebhookInteractiveAction `json:"action,omitempty"`
	ButtonReply *WhatsAppWebhookButtonReply       `json:"button_reply,omitempty"`
	ListReply   *WhatsAppWebhookListReply         `json:"list_reply,omitempty"`
}

type WhatsAppWebhookInteractiveHeader struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	Image    *WhatsAppWebhookMedia `json:"image,omitempty"`
	Video    *WhatsAppWebhookMedia `json:"video,omitempty"`
	Document *WhatsAppWebhookMedia `json:"document,omitempty"`
}

type WhatsAppWebhookInteractiveBody struct {
	Text string `json:"text"`
}

type WhatsAppWebhookInteractiveFooter struct {
	Text string `json:"text"`
}

type WhatsAppWebhookInteractiveAction struct {
	Button   string                         `json:"button,omitempty"`
	Buttons  []WhatsAppWebhookActionButton  `json:"buttons,omitempty"`
	Sections []WhatsAppWebhookActionSection `json:"sections,omitempty"`
	Name     string                         `json:"name,omitempty"`
}

type WhatsAppWebhookActionButton struct {
	Type  string                      `json:"type"`
	Reply *WhatsAppWebhookButtonReply `json:"reply,omitempty"`
}

type WhatsAppWebhookActionSection struct {
	Title string                     `json:"title,omitempty"`
	Rows  []WhatsAppWebhookActionRow `json:"rows"`
}

type WhatsAppWebhookActionRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type WhatsAppWebhookButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type WhatsAppWebhookListReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type WhatsAppWebhookReaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type WhatsAppWebhookContext struct {
	From            string                          `json:"from"`
	ID              string                          `json:"id"`
	ReferredProduct *WhatsAppWebhookReferredProduct `json:"referred_product,omitempty"`
}

type WhatsAppWebhookReferredProduct struct {
	CatalogID         string `json:"catalog_id"`
	ProductRetailerID string `json:"product_retailer_id"`
}

type WhatsAppWebhookStatus struct {
	ID           string                       `json:"id"`
	Status       string                       `json:"status"`
	Timestamp    string                       `json:"timestamp"`
	RecipientID  string                       `json:"recipient_id"`
	Conversation *WhatsAppWebhookConversation `json:"conversation,omitempty"`
	Pricing      *WhatsAppWebhookPricing      `json:"pricing,omitempty"`
	Errors       []WhatsAppWebhookError       `json:"errors,omitempty"`
}

type WhatsAppWebhookConversation struct {
	ID                  string                             `json:"id"`
	Origin              *WhatsAppWebhookConversationOrigin `json:"origin,omitempty"`
	ExpirationTimestamp string                             `json:"expiration_timestamp,omitempty"`
}

type WhatsAppWebhookConversationOrigin struct {
	Type string `json:"type"`
}

type WhatsAppWebhookPricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

type WhatsAppWebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type WhatsAppWebhookPayload struct {
	Object string                 `json:"object"`
	Entry  []WhatsAppWebhookEntry `json:"entry"`
}

func VerifyWebhook(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	verifyToken := getEnv("WHATSAPP_APP_SECRET", "gohublink_verify_token")

	if mode == "" || token == "" {
		log.Printf("WhatsApp webhook verification failed: missing params mode=%q token=%q", mode, token)
		c.String(http.StatusBadRequest, "Missing required parameters")
		return
	}

	if mode == "subscribe" && token == verifyToken {
		log.Printf("WhatsApp webhook verified successfully")
		c.String(http.StatusOK, challenge)
		return
	}

	log.Printf("WhatsApp webhook verification failed: mode=%s, token_match=%v", mode, token == verifyToken)
	c.String(http.StatusForbidden, "Verification failed")
}

func ReceiveWhatsAppWebhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Printf("WhatsApp webhook: failed to read body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if len(body) == 0 {
		log.Println("WhatsApp webhook: empty body, likely health check")
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	if !verifySignature(c, body) {
		log.Printf("WhatsApp webhook: invalid signature")
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
		return
	}

	var payload WhatsAppWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("WhatsApp webhook: invalid JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	log.Printf("WhatsApp webhook received: %d entries", len(payload.Entry))

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			saveWebhookEvent(&payload, change.Value)

			for _, msg := range change.Value.Messages {
				if err := processMessage(&change.Value, &msg); err != nil {
					log.Printf("WhatsApp webhook: error processing message %s: %v", msg.ID, err)
				}
			}

			for _, status := range change.Value.Statuses {
				if err := processStatus(&status); err != nil {
					log.Printf("WhatsApp webhook: error processing status %s: %v", status.ID, err)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func verifySignature(c *gin.Context, body []byte) bool {
	appSecret := getEnv("WHATSAPP_APP_SECRET", "")
	if appSecret == "" {
		log.Println("WARNING: WHATSAPP_APP_SECRET not set, skipping signature verification")
		return true
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		log.Println("WhatsApp webhook: missing X-Hub-Signature-256 header")
		return false
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		log.Printf("WhatsApp webhook: signature mismatch\n  expected: %s\n  received: %s", expected, signature)
		return false
	}

	return true
}

func saveWebhookEvent(payload *WhatsAppWebhookPayload, value WhatsAppWebhookValue) {
	raw, _ := json.Marshal(value)
	event := models.WhatsAppWebhookEvent{
		EventType: "messages",
		Payload:   datatypes.JSON(raw),
		Processed: true,
	}

	for _, msg := range value.Messages {
		event.MessageID = &msg.ID
		break
	}

	if err := db.DB.Create(&event).Error; err != nil {
		log.Printf("Failed to save webhook event: %v", err)
	}
}

func processMessage(value *WhatsAppWebhookValue, msg *WhatsAppWebhookMessage) error {
	waMsg := buildWhatsAppMessage(value, msg)

	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(waMsg).Error; err != nil {
			if !isDuplicateKeyError(err) {
				return err
			}
			log.Printf("Message %s already exists, skipping", msg.ID)
			return nil
		}

		updateCustomerActivity(tx, msg.From, value.Metadata.PhoneNumberID)
		return nil
	})
}

func buildWhatsAppMessage(value *WhatsAppWebhookValue, msg *WhatsAppWebhookMessage) *models.WhatsAppMessage {
	receivedAt, _ := parseTimestamp(msg.Timestamp)
	rawPayload, _ := json.Marshal(msg)

	waMsg := &models.WhatsAppMessage{
		MessageID:  msg.ID,
		From:       msg.From,
		To:         msg.To,
		Type:       parseMessageType(msg.Type),
		Direction:  models.WhatsAppMessageDirectionInbound,
		RawPayload: datatypes.JSON(rawPayload),
		ReceivedAt: receivedAt,
		Status:     "received",
	}

	switch msg.Type {
	case "text":
		if msg.Text != nil {
			waMsg.TextBody = &msg.Text.Body
		}
	case "image":
		if msg.Image != nil {
			waMsg.MediaID = &msg.Image.ID
			waMsg.MediaMimeType = &msg.Image.MimeType
			waMsg.MediaSHA256 = &msg.Image.SHA256
			if msg.Image.Caption != "" {
				waMsg.MediaCaption = &msg.Image.Caption
			}
		}
	case "audio":
		if msg.Audio != nil {
			waMsg.MediaID = &msg.Audio.ID
			waMsg.MediaMimeType = &msg.Audio.MimeType
			waMsg.MediaSHA256 = &msg.Audio.SHA256
		}
	case "video":
		if msg.Video != nil {
			waMsg.MediaID = &msg.Video.ID
			waMsg.MediaMimeType = &msg.Video.MimeType
			waMsg.MediaSHA256 = &msg.Video.SHA256
			if msg.Video.Caption != "" {
				waMsg.MediaCaption = &msg.Video.Caption
			}
		}
	case "document":
		if msg.Document != nil {
			waMsg.MediaID = &msg.Document.ID
			waMsg.MediaMimeType = &msg.Document.MimeType
			waMsg.MediaSHA256 = &msg.Document.SHA256
			if msg.Document.Caption != "" {
				waMsg.MediaCaption = &msg.Document.Caption
			}
			if msg.Document.Filename != "" {
				waMsg.MediaFilename = &msg.Document.Filename
			}
		}
	case "sticker":
		if msg.Sticker != nil {
			waMsg.MediaID = &msg.Sticker.ID
			waMsg.MediaMimeType = &msg.Sticker.MimeType
			waMsg.MediaSHA256 = &msg.Sticker.SHA256
		}
	case "location":
		if msg.Location != nil {
			waMsg.LocationLat = &msg.Location.Latitude
			waMsg.LocationLon = &msg.Location.Longitude
			if msg.Location.Name != "" {
				waMsg.LocationName = &msg.Location.Name
			}
			if msg.Location.Address != "" {
				waMsg.LocationAddress = &msg.Location.Address
			}
		}
	case "contacts":
		if len(msg.Contacts) > 0 {
			contactsData, _ := json.Marshal(msg.Contacts)
			waMsg.ContactData = datatypes.JSON(contactsData)
		}
	case "interactive":
		if msg.Interactive != nil {
			interactiveData, _ := json.Marshal(msg.Interactive)
			waMsg.InteractiveData = datatypes.JSON(interactiveData)
		}
	case "reaction":
		if msg.Reaction != nil {
			reactionData, _ := json.Marshal(msg.Reaction)
			waMsg.ReactionData = datatypes.JSON(reactionData)
		}
	}

	if msg.Context != nil {
		contextData, _ := json.Marshal(msg.Context)
		waMsg.ContextData = datatypes.JSON(contextData)
	}

	return waMsg
}

func parseMessageType(msgType string) models.WhatsAppMessageType {
	switch msgType {
	case "text":
		return models.WhatsAppMessageTypeText
	case "image":
		return models.WhatsAppMessageTypeImage
	case "audio":
		return models.WhatsAppMessageTypeAudio
	case "video":
		return models.WhatsAppMessageTypeVideo
	case "document":
		return models.WhatsAppMessageTypeDocument
	case "sticker":
		return models.WhatsAppMessageTypeSticker
	case "location":
		return models.WhatsAppMessageTypeLocation
	case "contacts":
		return models.WhatsAppMessageTypeContacts
	case "interactive":
		return models.WhatsAppMessageTypeInteractive
	case "reaction":
		return models.WhatsAppMessageTypeReaction
	case "system":
		return models.WhatsAppMessageTypeSystem
	default:
		return models.WhatsAppMessageTypeUnknown
	}
}

func parseTimestamp(ts string) (time.Time, error) {
	if ts == "" {
		return time.Now(), nil
	}
	return time.Parse(time.RFC3339, ts)
}

func processStatus(status *WhatsAppWebhookStatus) error {
	var msg models.WhatsAppMessage
	if err := db.DB.Where("message_id = ?", status.ID).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	msg.Status = status.Status
	now := time.Now()
	msg.ProcessedAt = &now

	return db.DB.Save(&msg).Error
}

func updateCustomerActivity(tx *gorm.DB, phoneNumber, phoneNumberID string) {
	var customer models.Customer
	err := tx.Where("phone_number = ?", phoneNumber).First(&customer).Error
	if err == gorm.ErrRecordNotFound {
		name := ""
		for _, contact := range getContactsFromContext() {
			if contact.WaID == phoneNumber {
				name = contact.Profile.Name
				break
			}
		}
		customer = models.Customer{
			PhoneNumber: phoneNumber,
			Name:        name,
			FirstScanAt: time.Now(),
			LastActive:  time.Now(),
		}
		if err := tx.Create(&customer).Error; err != nil {
			log.Printf("Failed to create customer: %v", err)
		}
		return
	}

	now := time.Now()
	customer.LastActive = now
	if err := tx.Save(&customer).Error; err != nil {
		log.Printf("Failed to update customer activity: %v", err)
	}
}

func getContactsFromContext() []WhatsAppWebhookContact {
	return []WhatsAppWebhookContact{}
}

func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ListWhatsAppMessages(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	msgType := c.Query("type")
	status := c.Query("status")
	limit := parseInt(c.DefaultQuery("limit", "50"))
	offset := parseInt(c.DefaultQuery("offset", "0"))

	query := db.DB.Model(&models.WhatsAppMessage{}).Order("received_at DESC")

	if from != "" {
		query = query.Where("from = ?", from)
	}
	if to != "" {
		query = query.Where("to = ?", to)
	}
	if msgType != "" {
		query = query.Where("type = ?", msgType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var messages []models.WhatsAppMessage
	if err := query.Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   messages,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func GetWhatsAppMessage(c *gin.Context) {
	id := c.Param("id")

	var msg models.WhatsAppMessage
	if err := db.DB.First(&msg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	c.JSON(http.StatusOK, msg)
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	if n <= 0 {
		return 50
	}
	if n > 100 {
		return 100
	}
	return n
}
