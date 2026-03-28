package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
)

// Config holds outbound mail settings read from environment variables.
type Config struct {
	Host string // SMTP_HOST
	Port string // SMTP_PORT
	User string // SMTP_USER
	Pass string // SMTP_PASS
	From string // SMTP_FROM
}

// ConfigFromEnv reads SMTP settings from environment variables.
func ConfigFromEnv() Config {
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return Config{
		Host: os.Getenv("SMTP_HOST"),
		Port: port,
		User: os.Getenv("SMTP_USER"),
		Pass: os.Getenv("SMTP_PASS"),
		From: os.Getenv("SMTP_FROM"),
	}
}

// Ready reports whether all required SMTP fields are set.
func (c Config) Ready() bool {
	return c.Host != "" && c.User != "" && c.Pass != "" && c.From != ""
}

// sendEmail sends a plain-text email to a single recipient.
// It tries implicit TLS first, then falls back to STARTTLS via smtp.SendMail.
func sendEmail(cfg Config, to, subject, body string) error {
	addr := cfg.Host + ":" + cfg.Port
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.From, to, subject, body,
	)

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(cfg.From); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = fmt.Fprint(wc, msg); err != nil {
		return err
	}
	return wc.Close()
}

// SendPasswordReset sends a password-reset email containing resetURL.
func SendPasswordReset(cfg Config, toEmail, resetURL string) error {
	subject := "Reset your Arcline IT password"
	body := fmt.Sprintf(
		"Hello,\r\n\r\n"+
			"A password reset was requested for your Arcline IT account.\r\n\r\n"+
			"Click the link below to set a new password (valid for 1 hour):\r\n\r\n"+
			"%s\r\n\r\n"+
			"If you did not request this, you can safely ignore this email.\r\n\r\n"+
			"-- Arcline IT Support\r\n",
		resetURL,
	)
	return sendEmail(cfg, toEmail, subject, body)
}
