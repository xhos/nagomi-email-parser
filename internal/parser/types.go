package parser

import (
	"nagomi-email-parser/internal/domain"
	"net/mail"
)

// EmailMeta holds the minimal fields needed to pick and parse a message
type EmailMeta struct {
	ID      string
	Subject string
	Text    string
	Date    string // RFC3339 from Mailpit
}

// Parser defines a bank‐specific parser
type Parser interface {
	Match(meta EmailMeta) bool
	Parse(meta EmailMeta) (*domain.Transaction, error)
}

func ToEmailMeta(id string, msg *mail.Message, decodedContent string) (EmailMeta, error) {
	subject := msg.Header.Get("Subject")

	var dateStr string
	if parsedDate, err := msg.Header.Date(); err == nil {
		dateStr = parsedDate.Format("2006-01-02T15:04:05Z07:00") // RFC3339
	} else {
		dateStr = msg.Header.Get("Date")
	}

	return EmailMeta{
		ID:      id,
		Subject: subject,
		Text:    decodedContent,
		Date:    dateStr,
	}, nil
}
