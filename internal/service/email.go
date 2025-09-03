package service

import (
	"net/smtp"
	"os"
)

type EmailService interface{ Send(to, subject, body string) error }
type smtpEmail struct{}

func NewEmailService() EmailService { return &smtpEmail{} }

func (s *smtpEmail) Send(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	from := os.Getenv("SMTP_FROM")
	addr := host + ":" + port

	msg := "From: "+from+"\r\n" +
		"To: "+to+"\r\n" +
		"Subject: "+subject+"\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" +
		body

	// MailHog için auth yok
	return smtp.SendMail(addr, nil, from, []string{to}, []byte(msg))
}
