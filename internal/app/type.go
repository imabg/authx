package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type Application struct {
	ID                uuid.UUID
	Name              string
	Description       string
	ClientID          string
	ClientSecretHash  string
	Settings          Settings
	Status            string
	UpdatedBy         string
	CreationTimestamp time.Time
	UpdatedTimestamp  time.Time
}

func (a *Application) Active() bool {
	return a != nil && a.Status == StatusActive
}

type ctxKey struct{}

func WithApplication(ctx context.Context, application *Application) context.Context {
	return context.WithValue(ctx, ctxKey{}, application)
}

func FromContext(ctx context.Context) *Application {
	application, _ := ctx.Value(ctxKey{}).(*Application)
	return application
}
