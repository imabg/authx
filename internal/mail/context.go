package mail

import "context"

type ctxKey struct{}

// WithApplicationID attaches the sending application id for mailer logs.
func WithApplicationID(ctx context.Context, applicationID string) context.Context {
	if applicationID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, applicationID)
}

func applicationIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
