package mail

import "context"

// Sender delivers transactional email.
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, to, subject, bodyHTML, bodyText string) error
}

// NoopSender is used when SMTP is not configured.
type NoopSender struct{}

func (NoopSender) Enabled() bool { return false }

func (NoopSender) Send(context.Context, string, string, string, string) error {
	return nil
}
