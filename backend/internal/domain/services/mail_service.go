package services

import "context"

type MailTemplateSender interface {
	Enabled() bool
	SendTemplate(ctx context.Context, templateName string, to []string, data any) error
}
