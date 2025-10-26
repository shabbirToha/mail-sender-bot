package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
)

func buildMessage(from, to, subject, body string, attachments map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n",
		from, to, subject, boundary)
	buf.WriteString(headers)

	// body
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", "text/plain; charset=utf-8")
	partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	w, _ := writer.CreatePart(partHeader)
	qp := quotedprintable.NewWriter(w)
	_, _ = qp.Write([]byte(body))
	qp.Close()

	// attachments
	for name, data := range attachments {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "application/octet-stream")
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
		h.Set("Content-Transfer-Encoding", "base64")
		part, _ := writer.CreatePart(h)
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(encoded, data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			part.Write(encoded[i:end])
			part.Write([]byte("\r\n"))
		}
	}

	writer.Close()
	return buf.Bytes(), nil
}

func sendMail(smtpHost, smtpPort, username, password string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", username, password, smtpHost)
	return smtp.SendMail(addr, auth, username, to, msg)
}
