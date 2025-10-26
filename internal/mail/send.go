package mail

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendMail sends a basic email via Gmail SMTP.
func SendMail(to, subject, body string) error {
	from := os.Getenv("GMAIL_USERNAME")
	pass := os.Getenv("GMAIL_PASSWORD")

	if from == "" || pass == "" {
		return fmt.Errorf("GMAIL_USERNAME or GMAIL_PASSWORD not set")
	}

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Gmail SMTP setup
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, pass, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	return nil
}
