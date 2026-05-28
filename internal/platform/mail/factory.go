package mail

import (
	"strings"

	"buhpro/internal/config"
)

// NewFromConfig returns SMTP sender when configured, otherwise NoopSender.
func NewFromConfig(cfg config.MailConfig) Sender {
	host := strings.TrimSpace(cfg.Host)
	from := strings.TrimSpace(cfg.From)
	if host == "" || from == "" {
		return NoopSender{}
	}

	port := cfg.Port
	if port <= 0 {
		port = 587
	}

	implicitTLS := cfg.ImplicitTLS
	if !cfg.ImplicitTLS && !cfg.StartTLS && port == 465 {
		implicitTLS = true
	}

	return NewSMTP(SMTPConfig{
		Host:        host,
		Port:        port,
		User:        strings.TrimSpace(cfg.User),
		Password:    cfg.Password,
		From:        from,
		FromName:    strings.TrimSpace(cfg.FromName),
		ImplicitTLS: implicitTLS,
	})
}
