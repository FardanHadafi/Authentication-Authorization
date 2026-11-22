package shared

import (
	"encoding/json/v2"
	"time"

	"github.com/hibiken/asynq"
)

// Nama identifier task
const TypeSendVerificationEmail = "email:verification"

// Data yang disimpan di antrean Redis
type VerificationEmailPayload struct {
	ToEmail          string `json:"to_email"`
	FullName         string `json:"full_name"`
	VerificationLink string `json:"verification_link"`
}

// Helper untuk membuat task dengan Timeout, Retry & DLQ
func NewVerificationEmailTask(toEmail, fullName, verificationLink string) (*asynq.Task, error) {
	payload, err := json.Marshal(VerificationEmailPayload{
		ToEmail:          toEmail,
		FullName:         fullName,
		VerificationLink: verificationLink,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(
		TypeSendVerificationEmail,
		payload, asynq.MaxRetry(3), // Retry otomatis coba hingga 3 kali
		asynq.Timeout(10*time.Second), // Timeout maksimal 10 detik per percobaan
		asynq.Retention(24*time.Hour), // DLQ jika gagal 3x simpan ke DLQ archived selama 24 jam
	), nil
}
