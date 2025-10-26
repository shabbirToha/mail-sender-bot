package bot

import (
	"fmt"
	"net/smtp"
	"time"
)

func ScheduleEmail(to, subject, body, username, password, smtpHost, smtpPort, schedule string) error {
	layout := "2006-01-02 15:04"
	sendTime, err := time.Parse(layout, schedule)
	if err != nil {
		return fmt.Errorf("invalid time format, use YYYY-MM-DD HH:MM")
	}

	go func() {
		duration := time.Until(sendTime)
		if duration > 0 {
			time.Sleep(duration)
		}

		auth := smtp.PlainAuth("", username, password, smtpHost)
		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

		msg := []byte(fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
			username, to, subject, body,
		))

		err := smtp.SendMail(addr, auth, username, []string{to}, msg)
		if err != nil {
			fmt.Println("❌ Failed to send scheduled mail:", err)
			return
		}

		fmt.Println("✅ Scheduled email sent successfully:", to)
	}()

	return nil
}
