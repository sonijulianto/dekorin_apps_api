package models

import "time"

// AppVersion menyimpan data versi aplikasi untuk update otomatis (in-app update APK)
type AppVersion struct {
	ID             string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Platform       string    `gorm:"type:varchar(20);default:'android';index" json:"platform"` // android / ios
	VersionName    string    `gorm:"type:varchar(20);not null" json:"version_name"`           // contoh: "1.0.1"
	VersionCode    int       `gorm:"not null" json:"version_code"`                             // contoh: 2 (build number)
	MinVersionCode int       `gorm:"default:1" json:"min_version_code"`                        // versi minimum sebelum wajib force update
	DownloadURL    string    `gorm:"type:varchar(500);not null" json:"download_url"`           // URL download APK
	ReleaseNotes   string    `gorm:"type:text" json:"release_notes"`                           // Catatan rilis pembaruan
	ForceUpdate    bool      `gorm:"default:false" json:"force_update"`                        // Wajib update atau opsional
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AppVersionResponse DTO untuk dikirim ke mobile app
type AppVersionResponse struct {
	Platform       string `json:"platform"`
	VersionName    string `json:"version_name"`
	VersionCode    int    `json:"version_code"`
	MinVersionCode int    `json:"min_version_code"`
	DownloadURL    string `json:"download_url"`
	ReleaseNotes   string `json:"release_notes"`
	ForceUpdate    bool   `json:"force_update"`
}
