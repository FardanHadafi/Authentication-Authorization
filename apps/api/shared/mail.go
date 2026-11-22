package shared

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
)

type EmailSender interface {
	SendVerificationEmail(toEmail, fullName, verificationLink string) error
}

type mailer struct {
	host string
	port string
	from string
	cb   *gobreaker.CircuitBreaker[any] // Circuit Breaker
}

func NewMailer(host, port, from string) EmailSender {
	// Konfigurasi circuit breaker sederhana & efektif
	cbSettings := gobreaker.Settings{
		Name:        "SMTP-CircuitBreaker",
		MaxRequests: 3,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip (buka sirkuit) jika gagal berturut2 3x
			return counts.ConsecutiveFailures >= 3
		},
	}

	return &mailer{
		host: host,
		port: port,
		from: from,
		cb:   gobreaker.NewCircuitBreaker[any](cbSettings),
	}
}

func (m *mailer) SendVerificationEmail(toMail, fullName, verificationLink string) error {
	// Eksekusi pengiriman di dalam circuit breaker
	_, err := m.cb.Execute(func() (any, error) {
		subject := "Verifikasi Email Akun Anda"
		body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #1e293b; max-width: 600px; margin: 0 auto; padding: 20px;">
	<h2 style="color: #0f172a;">Halo, %s!</h2>
	<p>Terima kasih telah mendaftar. Silakan klik tombol di bawah ini untuk memverifikasi akun Anda:</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">
			Verifikasi Akun Saya
		</a>
	</p>
	<p style="font-size: 13px; color: #64748b;">
		Jika tombol di atas tidak berfungsi, salin dan buka tautan berikut di browser Anda:<br>
		<a href="%s" style="color: #2563eb; word-break: break-all;">%s</a>
	</p>
</body>
</html>`, fullName, verificationLink, verificationLink, verificationLink)

		var message strings.Builder
		fmt.Fprintf(&message, "From: %s\r\n", m.from)
		fmt.Fprintf(&message, "To: %s\r\n", toMail)
		fmt.Fprintf(&message, "Subject: %s\r\n", subject)
		message.WriteString("MIME-Version: 1.0\r\n")
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(body)

		address := fmt.Sprintf("%s:%s", m.host, m.port)
		return nil, smtp.SendMail(address, nil, m.from, []string{toMail}, []byte(message.String()))
	})

	return err
}
