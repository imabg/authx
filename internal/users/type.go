package users

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type User struct {
	ID                uuid.UUID  `json:"id"`
	ApplicationID     uuid.UUID  `json:"application_id"`
	Email             string     `json:"email"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	SignupMethod      string     `json:"signup_method,omitempty"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at,omitempty"`
	EmailVerified     bool       `json:"email_verified"`
	Status            string     `json:"status"`
	CreationTimestamp time.Time  `json:"creation_timestamp"`
	UpdatedTimestamp  time.Time  `json:"updated_timestamp"`
}

func (u *User) Active() bool {
	return u != nil && u.Status == StatusActive
}

func (u *User) MarkVerified() {
	if u.EmailVerifiedAt != nil {
		u.EmailVerified = true
	}
}

type UserDTO struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}
