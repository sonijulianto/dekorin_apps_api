package models

import "time"

// AgendaChatRoom merepresentasikan ruang diskusi otomatis untuk sebuah LeadAgendaID yang memiliki kolaborasi aktif.
type AgendaChatRoom struct {
	ID                 string     `gorm:"primaryKey;type:varchar(50)" json:"id"`
	LeadAgendaID       string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"lead_agenda_id"`
	LastMessageAt      *time.Time `gorm:"index" json:"last_message_at,omitempty"`
	LastMessagePreview string     `gorm:"type:varchar(255);default:''" json:"last_message_preview"`
	LastSenderName     string     `gorm:"type:varchar(150);default:''" json:"last_sender_name"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// AgendaChatMessage menyimpan riwayat pesan (teks, foto, dokumen, dan system log) di dalam room diskusi kolaborasi.
type AgendaChatMessage struct {
	ID                  string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	RoomID              string    `gorm:"type:varchar(50);index;not null" json:"room_id"`
	SenderUserID        string    `gorm:"type:varchar(50);index;default:''" json:"sender_user_id"`
	SenderCompanyID     string    `gorm:"type:varchar(50);index;default:''" json:"sender_company_id"`
	SenderCompanyName   string    `gorm:"type:varchar(150);default:''" json:"sender_company_name"`
	SenderCategory      string    `gorm:"type:varchar(60);default:''" json:"sender_category"`
	SenderName          string    `gorm:"type:varchar(150);default:''" json:"sender_name"`
	SenderRole          string    `gorm:"type:varchar(30);default:''" json:"sender_role"`
	MessageType         string    `gorm:"type:varchar(20);default:'text';index" json:"message_type"` // text | image | file | system
	Content             string    `gorm:"type:text;default:''" json:"content"`
	AttachmentURL       string    `gorm:"type:text;default:''" json:"attachment_url"`
	AttachmentName      string    `gorm:"type:varchar(255);default:''" json:"attachment_name"`
	AttachmentSize      int64     `gorm:"default:0" json:"attachment_size"`
	ReplyToID           string    `gorm:"type:varchar(50);index;default:''" json:"reply_to_id"`
	ReplySenderName     string    `gorm:"type:varchar(150);default:''" json:"reply_sender_name"`
	ReplyPreview        string    `gorm:"type:varchar(255);default:''" json:"reply_preview"`
	MentionedCompanyIDs string    `gorm:"type:text;default:''" json:"mentioned_company_ids"`
	CreatedAt           time.Time `gorm:"index" json:"created_at"`
}

// AgendaChatRead mencatat waktu baca terakhir per user pada suatu room untuk menghitung unread badge.
type AgendaChatRead struct {
	ID         string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	RoomID     string    `gorm:"type:varchar(50);uniqueIndex:idx_chat_read_room_user;not null" json:"room_id"`
	UserID     string    `gorm:"type:varchar(50);uniqueIndex:idx_chat_read_room_user;not null" json:"user_id"`
	LastReadAt time.Time `json:"last_read_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AgendaChatMemberAssignment mencatat anggota tim (crew/staff) yang di-assign oleh Owner/Admin ke room diskusi agenda.
type AgendaChatMemberAssignment struct {
	ID               string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	RoomID           string    `gorm:"type:varchar(50);uniqueIndex:idx_chat_assign_room_user;not null" json:"room_id"`
	CompanyID        string    `gorm:"type:varchar(50);index;not null" json:"company_id"`
	UserID           string    `gorm:"type:varchar(50);uniqueIndex:idx_chat_assign_room_user;not null" json:"user_id"`
	AssignedByUserID string    `gorm:"type:varchar(50);default:''" json:"assigned_by_user_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// SendChatMessageRequest payload untuk mengirim pesan teks/foto/dokumen via REST API.
type SendChatMessageRequest struct {
	MessageType         string   `json:"message_type"` // text | image | file
	Content             string   `json:"content"`
	AttachmentURL       string   `json:"attachment_url"`
	AttachmentName      string   `json:"attachment_name"`
	AttachmentSize      int64    `json:"attachment_size"`
	ReplyToID           string   `json:"reply_to_id"`
	MentionedCompanyIDs []string `json:"mentioned_company_ids"`
}

// AssignChatCrewRequest payload untuk mengatur daftar crew/staff perusahaan yang di-assign ke room diskusi.
type AssignChatCrewRequest struct {
	UserIDs []string `json:"user_ids"`
}
