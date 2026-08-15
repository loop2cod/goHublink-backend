package models

import (
	"time"

	"gorm.io/datatypes"
)

type WhatsAppMessageType string

const (
	WhatsAppMessageTypeText       WhatsAppMessageType = "text"
	WhatsAppMessageTypeImage      WhatsAppMessageType = "image"
	WhatsAppMessageTypeAudio      WhatsAppMessageType = "audio"
	WhatsAppMessageTypeVideo      WhatsAppMessageType = "video"
	WhatsAppMessageTypeDocument   WhatsAppMessageType = "document"
	WhatsAppMessageTypeSticker    WhatsAppMessageType = "sticker"
	WhatsAppMessageTypeLocation   WhatsAppMessageType = "location"
	WhatsAppMessageTypeContacts   WhatsAppMessageType = "contacts"
	WhatsAppMessageTypeInteractive WhatsAppMessageType = "interactive"
	WhatsAppMessageTypeReaction   WhatsAppMessageType = "reaction"
	WhatsAppMessageTypeSystem     WhatsAppMessageType = "system"
	WhatsAppMessageTypeUnknown    WhatsAppMessageType = "unknown"
)

type WhatsAppMessageDirection string

const (
	WhatsAppMessageDirectionInbound  WhatsAppMessageDirection = "inbound"
	WhatsAppMessageDirectionOutbound WhatsAppMessageDirection = "outbound"
)

type WhatsAppMessage struct {
	ID              uint64                  `gorm:"primaryKey" json:"id"`
	MessageID       string                  `gorm:"uniqueIndex:idx_wa_msg_id;size:100;not null" json:"message_id"`
	From            string                  `gorm:"index:idx_wa_from;size:20;not null" json:"from"`
	To              string                  `gorm:"size:20" json:"to"`
	Type            WhatsAppMessageType     `gorm:"size:30;not null" json:"type"`
	Direction       WhatsAppMessageDirection `gorm:"size:10;not null;default:'inbound'" json:"direction"`
	TextBody        *string                 `gorm:"type:text" json:"text_body,omitempty"`
	MediaID         *string                 `gorm:"size:255" json:"media_id,omitempty"`
	MediaURL        *string                 `gorm:"size:500" json:"media_url,omitempty"`
	MediaMimeType   *string                 `gorm:"size:100" json:"media_mime_type,omitempty"`
	MediaCaption    *string                 `gorm:"type:text" json:"media_caption,omitempty"`
	MediaFilename   *string                 `gorm:"size:255" json:"media_filename,omitempty"`
	MediaSHA256     *string                 `gorm:"size:64" json:"media_sha256,omitempty"`
	MediaSize       *int64                  `json:"media_size,omitempty"`
	LocationLat     *float64                `json:"location_lat,omitempty"`
	LocationLon     *float64                `json:"location_lon,omitempty"`
	LocationName    *string                 `gorm:"size:255" json:"location_name,omitempty"`
	LocationAddress *string                 `gorm:"size:500" json:"location_address,omitempty"`
	ContactData     datatypes.JSON          `gorm:"type:jsonb" json:"contact_data,omitempty"`
	InteractiveData datatypes.JSON          `gorm:"type:jsonb" json:"interactive_data,omitempty"`
	ReactionData    datatypes.JSON          `gorm:"type:jsonb" json:"reaction_data,omitempty"`
	ContextData     datatypes.JSON          `gorm:"type:jsonb" json:"context_data,omitempty"`
	RawPayload      datatypes.JSON          `gorm:"type:jsonb;not null" json:"raw_payload"`
	Status          string                  `gorm:"size:20;default:'received'" json:"status"`
	ErrorMessage    *string                 `gorm:"type:text" json:"error_message,omitempty"`
	ReceivedAt      time.Time               `gorm:"not null;default:CURRENT_TIMESTAMP" json:"received_at"`
	ProcessedAt     *time.Time              `json:"processed_at,omitempty"`
}

type WhatsAppWebhookEvent struct {
	ID        uint64      `gorm:"primaryKey" json:"id"`
	EventType string      `gorm:"size:50;not null" json:"event_type"`
	MessageID *string     `gorm:"size:100" json:"message_id,omitempty"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	Processed bool        `gorm:"default:false" json:"processed"`
	Error     *string     `gorm:"type:text" json:"error,omitempty"`
	CreatedAt time.Time   `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}