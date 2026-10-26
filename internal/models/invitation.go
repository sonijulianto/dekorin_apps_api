package models

import "time"

// Invitation mewakili data website undangan pernikahan & acara digital mandiri
type Invitation struct {
	ID             string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID      string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	Slug           string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"` // URL slug, misal: bagas-zulia
	Title          string    `gorm:"type:varchar(200);not null" json:"title"`           // misal: The Wedding of Bagas & Zulia
	Template       string    `gorm:"type:varchar(50);default:'green-leaves'" json:"template"` // green-leaves, creamy-latte, floral, luxury-grey, khitan-green
	EventType      string    `gorm:"type:varchar(50);default:'wedding'" json:"event_type"` // wedding, khitan
	CoverImage     string    `gorm:"type:text" json:"cover_image"`
	Quotes         string    `gorm:"type:text" json:"quotes"` // Kutipan ayat suci / mutiara

	// Data Mempelai Pria (atau Anak Khitan)
	GroomName       string `gorm:"type:varchar(150);not null" json:"groom_name"`
	GroomNickname   string `gorm:"type:varchar(50)" json:"groom_nickname"`
	GroomChildOrder string `gorm:"type:varchar(100)" json:"groom_child_order"` // Misal: Putra pertama dari
	GroomParents    string `gorm:"type:varchar(255)" json:"groom_parents"`
	GroomInstagram  string `gorm:"type:varchar(100)" json:"groom_instagram"`
	GroomPhoto      string `gorm:"type:text" json:"groom_photo"`

	// Data Mempelai Wanita
	BrideName       string `gorm:"type:varchar(150)" json:"bride_name"`
	BrideNickname   string `gorm:"type:varchar(50)" json:"bride_nickname"`
	BrideChildOrder string `gorm:"type:varchar(100)" json:"bride_child_order"` // Misal: Putri kedua dari
	BrideParents    string `gorm:"type:varchar(255)" json:"bride_parents"`
	BrideInstagram  string `gorm:"type:varchar(100)" json:"bride_instagram"`
	BridePhoto      string `gorm:"type:text" json:"bride_photo"`

	// Data Khusus Khitanan (jika EventType == khitan)
	ChildName     string `gorm:"type:varchar(150)" json:"child_name"`
	ChildNickname string `gorm:"type:varchar(50)" json:"child_nickname"`
	ChildOrder    string `gorm:"type:varchar(100)" json:"child_order"`

	// Jadwal & Lokasi Akad / Acara Inti
	AkadDate    string `gorm:"type:varchar(50)" json:"akad_date"` // format: YYYY-MM-DD atau teks
	AkadTime    string `gorm:"type:varchar(50)" json:"akad_time"` // misal: 08:00 - 10:00 WIB
	AkadVenue   string `gorm:"type:varchar(150)" json:"akad_venue"`
	AkadAddress string `gorm:"type:text" json:"akad_address"`
	AkadMapsURL string `gorm:"type:text" json:"akad_maps_url"`

	// Jadwal & Lokasi Resepsi / Tasyakuran
	ResepsiDate    string `gorm:"type:varchar(50)" json:"resepsi_date"`
	ResepsiTime    string `gorm:"type:varchar(50)" json:"resepsi_time"` // misal: 11:00 - 14:00 WIB
	ResepsiVenue   string `gorm:"type:varchar(150)" json:"resepsi_venue"`
	ResepsiAddress string `gorm:"type:text" json:"resepsi_address"`
	ResepsiMapsURL string `gorm:"type:text" json:"resepsi_maps_url"`

	// Amplop Digital & Rekening
	BankName1    string `gorm:"type:varchar(50)" json:"bank_name_1"`
	BankAccount1 string `gorm:"type:varchar(50)" json:"bank_account_1"`
	BankHolder1  string `gorm:"type:varchar(150)" json:"bank_holder_1"`

	BankName2    string `gorm:"type:varchar(50)" json:"bank_name_2"`
	BankAccount2 string `gorm:"type:varchar(50)" json:"bank_account_2"`
	BankHolder2  string `gorm:"type:varchar(150)" json:"bank_holder_2"`

	GiftAddress    string `gorm:"type:text" json:"gift_address"` // Alamat kirim kado fisik
	WaConfirmation string `gorm:"type:varchar(50)" json:"wa_confirmation"` // Nomor WA konfirmasi hadiah/kehadiran

	// Media Pendukung & Cerita
	AudioURL       string `gorm:"type:text" json:"audio_url"`       // Lagu latar mp3
	GalleryImages  string `gorm:"type:text" json:"gallery_images"`  // URL foto galeri dipisah koma
	YoutubeURL     string `gorm:"type:text" json:"youtube_url"`      // Video YouTube prewedding/momen
	LoveStory      string `gorm:"type:text" json:"love_story"`       // Rangkuman kisah cinta / milestone
	TurutMengundang string `gorm:"type:text" json:"turut_mengundang"` // Daftar keluarga besar yang mengundang

	// Foto Spesifik Sesuai Desain Template Asli
	Sec1Photo    string `gorm:"type:text" json:"sec1_photo"`    // sec1.jpeg (Foto background pembuka / quote)
	AkadPhoto    string `gorm:"type:text" json:"akad_photo"`    // akad.jpeg (Foto banner kartu Akad Nikah)
	ResepsiPhoto string `gorm:"type:text" json:"resepsi_photo"` // reception.jpeg (Foto banner kartu Resepsi)
	EventPhoto   string `gorm:"type:text" json:"event_photo"`   // event.jpeg (Foto kartu countdown event)
	ClosingPhoto string `gorm:"type:text" json:"closing_photo"` // closing.jpeg (Foto penutup / thank you)

	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Wishes []InvitationWish `gorm:"foreignKey:InvitationID" json:"wishes,omitempty"`
}

// InvitationWish mewakili ucapan doa & konfirmasi kehadiran tamu (Buku Tamu / RSVP)
type InvitationWish struct {
	ID           string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	InvitationID string    `gorm:"type:varchar(50);index;not null" json:"invitation_id"`
	GuestName    string    `gorm:"type:varchar(150);not null" json:"guest_name"`
	Attendance   string    `gorm:"type:varchar(50);default:'hadir'" json:"attendance"` // hadir, tidak_hadir, ragu
	GuestCount   int       `gorm:"default:1" json:"guest_count"`
	Message      string    `gorm:"type:text;not null" json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateOrUpdateInvitationRequest payload pembuatan / perubahan undangan
type CreateOrUpdateInvitationRequest struct {
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	Template        string `json:"template"` // green-leaves, creamy-latte, floral, luxury-grey, khitan-green
	EventType       string `json:"event_type"` // wedding, khitan
	CoverImage      string `json:"cover_image"`
	Quotes          string `json:"quotes"`
	GroomName       string `json:"groom_name"`
	GroomNickname   string `json:"groom_nickname"`
	GroomChildOrder string `json:"groom_child_order"`
	GroomParents    string `json:"groom_parents"`
	GroomInstagram  string `json:"groom_instagram"`
	GroomPhoto      string `json:"groom_photo"`
	BrideName       string `json:"bride_name"`
	BrideNickname   string `json:"bride_nickname"`
	BrideChildOrder string `json:"bride_child_order"`
	BrideParents    string `json:"bride_parents"`
	BrideInstagram  string `json:"bride_instagram"`
	BridePhoto      string `json:"bride_photo"`
	ChildName       string `json:"child_name"`
	ChildNickname   string `json:"child_nickname"`
	ChildOrder      string `json:"child_order"`
	AkadDate        string `json:"akad_date"`
	AkadTime        string `json:"akad_time"`
	AkadVenue       string `json:"akad_venue"`
	AkadAddress     string `json:"akad_address"`
	AkadMapsURL     string `json:"akad_maps_url"`
	ResepsiDate     string `json:"resepsi_date"`
	ResepsiTime     string `json:"resepsi_time"`
	ResepsiVenue    string `json:"resepsi_venue"`
	ResepsiAddress  string `json:"resepsi_address"`
	ResepsiMapsURL  string `json:"resepsi_maps_url"`
	BankName1       string `json:"bank_name_1"`
	BankAccount1    string `json:"bank_account_1"`
	BankHolder1     string `json:"bank_holder_1"`
	BankName2       string `json:"bank_name_2"`
	BankAccount2    string `json:"bank_account_2"`
	BankHolder2     string `json:"bank_holder_2"`
	GiftAddress     string `json:"gift_address"`
	WaConfirmation  string `json:"wa_confirmation"`
	AudioURL        string `json:"audio_url"`
	GalleryImages   string `json:"gallery_images"`
	YoutubeURL      string `json:"youtube_url"`
	LoveStory       string `json:"love_story"`
	TurutMengundang string `json:"turut_mengundang"`
	Sec1Photo       string `json:"sec1_photo"`
	AkadPhoto       string `json:"akad_photo"`
	ResepsiPhoto    string `json:"resepsi_photo"`
	EventPhoto      string `json:"event_photo"`
	ClosingPhoto    string `json:"closing_photo"`
	IsActive        *bool  `json:"is_active"`
}

// SubmitWishRequest payload pengiriman ucapan / RSVP tamu
type SubmitWishRequest struct {
	GuestName  string `json:"guest_name"`
	Attendance string `json:"attendance"`
	GuestCount int    `json:"guest_count"`
	Message    string `json:"message"`
}
