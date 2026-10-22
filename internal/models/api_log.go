package models

import "time"

// ApiLog mewakili rekaman log request & response API
type ApiLog struct {
	ID           string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID    string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	Method       string    `gorm:"type:varchar(10);index" json:"method"`
	Status       string    `gorm:"type:varchar(20);index" json:"status"` // SUCCESS, FAILED
	StatusCode   int       `gorm:"type:int" json:"status_code"`
	URL          string    `gorm:"type:varchar(255);index" json:"url"`
	IPAddr       string    `gorm:"type:varchar(50)" json:"ip_addr"`
	Username     string    `gorm:"type:varchar(100);index" json:"username"`
	LogTime      time.Time `gorm:"index" json:"log_time"`
	ReqData      string    `gorm:"type:text" json:"req_data"`
	ResponseData string    `gorm:"type:text" json:"response_data"`
	Remark       string    `gorm:"type:varchar(255)" json:"remark"`
	ApkVersion   string    `gorm:"type:varchar(50)" json:"apk_version"`
	DeviceModel  string    `gorm:"type:varchar(100)" json:"device_model"`
	BuildDate    *string   `gorm:"type:varchar(50)" json:"build_date"`
	Name         string    `gorm:"type:varchar(100)" json:"name"`
	BranchCode   string    `gorm:"type:varchar(50)" json:"branch_code"`
	BranchName   string    `gorm:"type:varchar(100)" json:"branch_name"`
	CreatedAt    time.Time `json:"created_at"`
}
