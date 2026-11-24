package shared

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/wneessen/go-mail"
)

type EmailSender interface {
	SendVerificationEmail(toEmail, fullName, verificationLink string) error
	SendResetPasswordEmail(toEmail, fullName, resetLink string) error
	SendResendVerificationEmail(toEmail, fullName, verificationLink string) error
}

type mailer struct {
	host string
	port int
	from string
	cb   *gobreaker.CircuitBreaker[any]
}

// NewMailer creates an EmailSender powered by go-mail and protected by a circuit breaker.
func NewMailer(host string, port int, from string) EmailSender {
	if port <= 0 {
		port = 1025
	}
	cbSettings := gobreaker.Settings{
		Name:        "SMTP-CircuitBreaker",
		MaxRequests: 3,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
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

func (m *mailer) send(toMail, subject, htmlBody string) error {
	_, err := m.cb.Execute(func() (any, error) {
		msg := mail.NewMsg()
		if err := msg.From(m.from); err != nil {
			return nil, fmt.Errorf("invalid from address: %w", err)
		}
		if err := msg.To(toMail); err != nil {
			return nil, fmt.Errorf("invalid to address: %w", err)
		}
		msg.Subject(subject)
		msg.SetBodyString(mail.TypeTextHTML, htmlBody)

		client, err := mail.NewClient(m.host,
			mail.WithPort(m.port),
			mail.WithTLSPolicy(mail.TLSOpportunistic),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}

		if err := client.DialAndSend(msg); err != nil {
			return nil, fmt.Errorf("failed to send email via SMTP: %w", err)
		}

		return nil, nil
	})
	return err
}

func (m *mailer) SendVerificationEmail(toMail, fullName, verificationLink string) error {
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

	return m.send(toMail, subject, body)
}

func (m *mailer) SendResetPasswordEmail(toMail, fullName, resetLink string) error {
	subject := "Instruksi Reset Kata Sandi Akun Anda"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #1e293b; max-width: 600px; margin: 0 auto; padding: 20px;">
	<h2 style="color: #0f172a;">Halo, %s!</h2>
	<p>Kami menerima permintaan untuk mereset kata sandi akun Anda. Silakan klik tombol di bawah ini untuk membuat kata sandi baru:</p>
	<p style="margin: 24px 0;">
		<a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">
			Reset Kata Sandi
		</a>
	</p>
	<p style="font-size: 13px; color: #64748b;">
		Tautan ini hanya berlaku selama <strong>15 menit</strong>. Jika Anda tidak pernah meminta reset kata sandi, Anda dapat mengabaikan email ini dengan aman.
	</p>
	<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
	<p style="font-size: 13px; color: #64748b;">
		Jika tombol di atas tidak berfungsi, salin dan buka tautan berikut di browser Anda:<br>
		<a href="%s" style="color: #2563eb; word-break: break-all;">%s</a>
	</p>
</body>
</html>`, fullName, resetLink, resetLink, resetLink)

	return m.send(toMail, subject, body)
}

func (m *mailer) SendResendVerificationEmail(toMail, fullName, verificationLink string) error {
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

	return m.send(toMail, subject, body)
}
