package email

import (
	"net/smtp"
	"os"
)

const (
	from     = "officegames.rowmur@gmail.com"
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

var password = ""

func SendEmail(to []string, subject, body string) error {
	if password == "" {
		password = os.Getenv("EMAIL_PASSWORD")
	}

	message := []byte("Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		body,
	)

	auth := smtp.PlainAuth("", from, password, smtpHost)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
}
