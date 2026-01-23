package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Email     string    `json:"email,omitempty" binding:"required,email"`
	Password  string    `json:"-" binding:"required,min=8,max=72"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}
