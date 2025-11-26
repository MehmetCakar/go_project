package service

import (
	"crypto/tls"
	"fmt"

	gomail "gopkg.in/gomail.v2"
)

type EmailSender interface {
	Send(to, subject, htmlBody string) error
}

type SMTPSender struct {
	host string
	port int
	user string
	pass string
	from string
	name string
}

func NewSMTPSender(host string, port int, user, pass, from, name string) *SMTPSender {
	return &SMTPSender{host: host, port: port, user: user, pass: pass, from: from, name: name}
}

func (s *SMTPSender) Send(to, subject, htmlBody string) error {
	if s.host == "" || s.user == "" || s.pass == "" || s.from == "" {
		return fmt.Errorf("smtp not configured: host/user/pass/from required")
	}
	m := gomail.NewMessage()
	if s.name != "" {
		m.SetHeader("From", m.FormatAddress(s.from, s.name))
	} else {
		m.SetHeader("From", s.from)
	}
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(s.host, s.port, s.user, s.pass)
	d.TLSConfig = &tls.Config{ServerName: s.host} // STARTTLS

	return d.DialAndSend(m)
}
