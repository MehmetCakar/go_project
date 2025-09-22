package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type EmailService interface {
	Send(to, subject, body string) error
}

type smtpEmail struct{}

func NewEmailService() EmailService { return &smtpEmail{} }

func (s *smtpEmail) Send(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")           // smtp-relay.brevo.com
	port := os.Getenv("SMTP_PORT")           // 587
	user := os.Getenv("SMTP_USER")           // 9748...@smtp-brevo.com
	pass := os.Getenv("SMTP_PASS")           // ********
	from := os.Getenv("SMTP_FROM")           // Ecom GO <no-reply@cakarokko.com>

	addr := host + ":" + port

	// RFC 5322 compliant headers
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	sb.WriteString("\r\n")
	sb.WriteString(body)
	msg := []byte(sb.String())

	// AUTH
	auth := smtp.PlainAuth("", user, pass, host)

	// STARTTLS (587)
	tlsconfig := &tls.Config{ServerName: host}
	// smtp.SendMail kendi STARTTLS’ı çağırır; ama CA/tls sorunlarında bu fallback işe yarar.
	_ = tlsconfig // sadece örnek; genelde SendMail yeter

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
