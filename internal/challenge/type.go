package challenge

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeOTP       = "otp"
	TypeMagicLink = "magic_link"
)

type Challenge struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	UserID        *uuid.UUID
	Type          string
	Target        string
	SecretHash    string
	ExpiresAt     time.Time
	ConsumedAt    *time.Time
	AttemptCount  int
	CreatedAt     time.Time
}
