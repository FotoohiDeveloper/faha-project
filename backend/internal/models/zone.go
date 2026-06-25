package models

import "github.com/google/uuid"

type Zone struct {
	ID   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"` // مثلا "استان فارس" یا "شیراز"

	Polygon string `gorm:"type:geometry(Polygon,4326);not null" json:"-"`

	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id"` // برای همپوشانی: والد شیراز میشه استان فارس
	Parent   *Zone      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
}