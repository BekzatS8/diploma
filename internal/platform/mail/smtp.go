package mail

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
	// ImplicitTLS: port 465 (SMTPS). Otherwise STARTTLS is used when advertised (typical for 587).
	ImplicitTLS bool
}

type smtpSender struct {
	cfg SMTPConfig
}

// NewSMTP creates an SMTP mail sender. Host must be non-empty.
func NewSMTP(cfg SMTPConfig) Sender {
	return &smtpSender{cfg: cfg}
}

func (s *smtpSender) Enabled() bool {
	return strings.TrimSpace(s.cfg.Host) != "" && strings.TrimSpace(s.cfg.From) != ""
}

func (s *smtpSender) Send(ctx context.Context, to, subject, bodyHTML, bodyText string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	from := strings.TrimSpace(s.cfg.From)
	fromHeader := from
	if name := strings.TrimSpace(s.cfg.FromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", encodeRFC2047(name), from)
	}

	msg := buildMIMEMessage(fromHeader, to, subject, bodyHTML, bodyText)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.ImplicitTLS {
		return s.sendImplicitTLS(addr, from, to, msg)
	}
	return s.sendSTARTTLS(addr, from, to, msg)
}

func (s *smtpSender) sendImplicitTLS(addr, from, to string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := s.authenticate(client); err != nil {
		return err
	}
	return s.submit(client, from, to, msg)
}

func (s *smtpSender) sendSTARTTLS(addr, from, to string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if err := s.authenticate(client); err != nil {
		return err
	}
	return s.submit(client, from, to, msg)
}

func (s *smtpSender) authenticate(client *smtp.Client) error {
	if strings.TrimSpace(s.cfg.User) == "" {
		return nil
	}
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	if ok, _ := client.Extension("AUTH"); !ok {
		return fmt.Errorf("smtp: AUTH not supported")
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return nil
}

func (s *smtpSender) submit(client *smtp.Client, from, to string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	return client.Quit()
}

func buildMIMEMessage(from, to, subject, bodyHTML, bodyText string) []byte {
	if bodyText == "" {
		bodyText = stripHTMLTags(bodyHTML)
	}
	boundary := "buhpro-mail-boundary"

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + encodeRFC2047(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	b.WriteString("\r\n")

	writePart := func(contentType, body string) {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(body)
		b.WriteString("\r\n")
	}

	writePart("text/plain", bodyText)
	writePart("text/html", bodyHTML)
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func encodeRFC2047(s string) string {
	if s == "" {
		return ""
	}
	// ASCII-only subjects need no encoding.
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

func stripHTMLTags(html string) string {
	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n\n")
	text := replacer.Replace(html)
	var out strings.Builder
	inTag := false
	for _, r := range text {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}
