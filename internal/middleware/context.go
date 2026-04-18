package middleware

import "context"

type key string

const (
	KeyUserID  key = "userID"
	KeyOrgID   key = "orgID"
	KeyRole    key = "role"
	KeyEmail   key = "email"
)

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(KeyUserID).(string)
	return v
}

func OrgID(ctx context.Context) string {
	v, _ := ctx.Value(KeyOrgID).(string)
	return v
}

func Role(ctx context.Context) string {
	v, _ := ctx.Value(KeyRole).(string)
	return v
}

func UserEmail(ctx context.Context) string {
	v, _ := ctx.Value(KeyEmail).(string)
	return v
}
