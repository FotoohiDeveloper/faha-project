package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UserStatus string
type UserRank string

const (
	StatusSuspended       UserStatus = "SUSPENDED"        
	StatusPendingApproval UserStatus = "PENDING_APPROVAL" 
	StatusActive          UserStatus = "ACTIVE"           
	StatusBlocked         UserStatus = "BLOCKED"          
)

const (
	RankDirectorGeneral UserRank = "DIRECTOR_GENERAL" // مدیر کل
	RankCommander       UserRank = "COMMANDER"        // فرمانده
	RankAirDefenseOp    UserRank = "AIR_DEFENSE_OP"   // اپراتور پدافندی
	RankManpadsOp       UserRank = "MANPADS_OP"       // اپراتور دوش‌پرتاب
	RankMithaqOp        UserRank = "MITHAQ_OP"        // اپراتور سامانه میثاق
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username     string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"` 
	Rank         UserRank   `gorm:"type:varchar(50);not null" json:"rank"`
	Status       UserStatus `gorm:"type:varchar(50);default:'SUSPENDED';not null" json:"status"`

	Permissions pq.Int64Array `gorm:"type:bigint[]" json:"permissions"`

	ZoneID *uuid.UUID `gorm:"type:uuid" json:"zone_id"`
	Zone   *Zone      `gorm:"foreignKey:ZoneID" json:"zone,omitempty"`

	CreatedByID *uuid.UUID `gorm:"type:uuid" json:"created_by_id"`
	CreatedBy   *User      `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`

	Credentials     []WebAuthnCred `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"-"`
	PasswordHistory []PassHistory  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PassHistory struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uuid.UUID `gorm:"type:uuid;index;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time
}

type WebAuthnCred struct {
	ID              uint           `gorm:"primaryKey"`
	UserID          uuid.UUID      `gorm:"type:uuid;index;not null"`
	CredentialID    []byte         `gorm:"type:bytea;uniqueIndex;not null"` 
	PublicKey       []byte         `gorm:"type:bytea;not null"`             
	AttestationType string         `gorm:"type:varchar(100)"`
	Transport       pq.StringArray `gorm:"type:text[]"`
	SignCount       uint32         `gorm:"type:bigint;default:0"`
	CreatedAt       time.Time
}