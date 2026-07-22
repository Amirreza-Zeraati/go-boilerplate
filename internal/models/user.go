package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base holds fields every table shares: a UUID primary key, created/updated
// timestamps, and a soft-delete marker. Embed it in your models.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate generates a UUID if one wasn't set explicitly.
func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User is the example domain model. Replace/extend with your real entities.
// Note: never store plaintext passwords — PasswordHash holds a bcrypt/argon2
// hash, which the (later) auth layer will populate.
type User struct {
	Base
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         string `gorm:"not null;default:user" json:"role"`
}

// TableName pins the table name so struct renames don't silently change it.
func (User) TableName() string { return "users" }
