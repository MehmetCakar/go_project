package service

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
//	"time"

	gomail "gopkg.in/gomail.v2"
)

type EmailService interface {
	Send(to, subject, htmlBody string) error
}

type emailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
}

func NewEmailService() EmailService {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	fromName := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")) // opsiyonel

	port := 587
	if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
		port = p
	}

	return &emailService{
		host:     host,
		port:     port,
		username: user,
		password: pass,
		from:     from,
		fromName: fromName,
	}
}

func (e *emailService) Send(to, subject, verifyLink string) error {
    m := gomail.NewMessage()
    e.fromName = os.Getenv("SMTP_FROM_NAME")

    // From: (fromName varsa kullan; yoksa sade e.from)
    if e.fromName != "" {
        m.SetHeader("From", fmt.Sprintf("%s <%s>", e.fromName, e.from))
    } else {
        m.SetHeader("From", "Cakarokko <no-reply@cakarokko.com>")
    }

    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)

    // ✅ Spam dostu, profesyonel HTML gövde
    htmlBody := fmt.Sprintf(`
    <html>
      <body style="font-family: Arial, sans-serif; color: #333; background-color:#f9f9f9; padding: 20px;">
        <h2 style="color:#4CAF50;">Welcome to Cakarokko 🎉</h2>
        <p>Thank you for registering with <strong>Cakarokko</strong>!</p>
        <p>To complete your registration, please verify your email by clicking the button below:</p>

        <p style="margin: 30px 0;">
          <a href="%s" style="display:inline-block;background:#4CAF50;color:white;padding:14px 24px;text-decoration:none;border-radius:5px;">
            ✅ Verify My Email
          </a>
        </p>

        <p>If you didn’t request this email, you can safely ignore it.</p>
        <hr>
        <p style="font-size:12px; color:#888;">
          📩 This email was sent automatically by Cakarokko. Please do not reply.<br>
          🌐 Visit us: <a href="https://cakarokko.com">https://cakarokko.com</a>
        </p>
      </body>
    </html>
    `, verifyLink)

    // HTML gövde
    m.SetBody("text/html", htmlBody)

// Plain-text alternatif: link YOK
//	plain := "Merhaba!\n\nAşağıdaki 6 haneli kodu 15 dakika içinde sitedeki doğrulama kutusuna gir:\n\n" +
  //      	 "KOD: " + code + "\n\n" +
    //  	 	  "Bu işlemi sen başlatmadıysan bu e-postayı yok sayabilirsin."
//	m.AddAlternative("text/plain", plain)

    // ✅ Plain text alternatif (Gmail ve Outlook spam skoru için önemli!)
//    m.AddAlternative("text/plain", "Merhaba! E-postanı doğrulamak için bu bağlantıya tıkla: "+verifyLink)

    d := gomail.NewDialer(e.host, e.port, e.username, e.password)

    // STARTTLS sonrası SNI/hostname doğrulaması için ServerName’i zorla
    d.TLSConfig = &tls.Config{
        ServerName:         e.host,
        MinVersion:         tls.VersionTLS12,
        InsecureSkipVerify: false,
    }

    return d.DialAndSend(m)
}
