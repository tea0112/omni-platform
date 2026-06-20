package shared

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func NewSMTPEmailSender(cfg SMTPConfig) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
	}
}

func (s *SMTPEmailSender) SendPasswordReset(ctx context.Context, to, token string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	body := fmt.Sprintf("Subject: Password Reset\r\n\r\nYour password reset token: %s\r\n", token)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(body))
}
