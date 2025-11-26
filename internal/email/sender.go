package email

import (
	"fmt"
	"os"

	gomail "gopkg.in/gomail.v2"
)

type Sender struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func NewFromEnv() (*Sender, error) {
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	port := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	if host == "" || user == "" || pass == "" || from == "" {
		return nil, fmt.Errorf("SMTP env eksik (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM)")
	}
	return &Sender{Host: host, Port: port, User: user, Pass: pass, From: from}, nil
}

func (s *Sender) Send(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(s.Host, s.Port, s.User, s.Pass)
	return d.DialAndSend(m)
}
