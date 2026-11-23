package shared

import (
	"context"
	"encoding/json/v2"
	"log/slog"

	"github.com/hibiken/asynq"
)

type TaskHandler struct {
	mailer EmailSender
	logger *slog.Logger
}

func NewTaskHandler(mailer EmailSender, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		mailer: mailer,
		logger: logger,
	}
}

// Register mendaftarkan seluruh task handler Asynq ke ServeMux
func (h *TaskHandler) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeSendVerificationEmail, h.HandleSendVerificationEmail)
	mux.HandleFunc(TypeSendResetPasswordEmail, h.HandleSendResetPasswordEmail)
	mux.HandleFunc(TypeSendResendVerificationEmail, h.HandleSendResendVerificationEmail)
}

func (h *TaskHandler) HandleSendVerificationEmail(ctx context.Context, t *asynq.Task) error {
	var payload VerificationEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	h.logger.Info("Processing verification email task", "to", payload.ToEmail)

	return h.mailer.SendVerificationEmail(payload.ToEmail, payload.FullName, payload.VerificationLink)
}

func (h *TaskHandler) HandleSendResetPasswordEmail(ctx context.Context, t *asynq.Task) error {
	var payload ResetPasswordEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	h.logger.Info("Processing reset password email task", "to", payload.ToEmail)

	return h.mailer.SendResetPasswordEmail(payload.ToEmail, payload.FullName, payload.ResetLink)
}

func (h *TaskHandler) HandleSendResendVerificationEmail(ctx context.Context, t *asynq.Task) error {
	var payload ResendVerificationEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	h.logger.Info("Processing resend email verification task", "to", payload.ToEmail)

	return h.mailer.SendResendVerificationEmail(payload.ToEmail, payload.FullName, payload.VerificationLink)
}
