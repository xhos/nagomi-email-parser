package rbc

import (
	"regexp"
	"strings"

	"nagomi-email-parser/internal/domain"
	"nagomi-email-parser/internal/parser"
)

func init() { parser.Register(&deposit{}) }

type deposit struct{}

func (d *deposit) Match(m parser.EmailMeta) bool {
	return strings.Contains(m.Subject, "Deposit Notice") &&
		strings.Contains(m.Text, "RBC Royal Bank")
}

func (d *deposit) Parse(m parser.EmailMeta) (*domain.Transaction, error) {
	patterns := map[string]*regexp.Regexp{
		"account": regexp.MustCompile(`bank account ([A-Za-z]+)`),
		"amount":  regexp.MustCompile(`\$([0-9,]+\.\d{2})`),
		"txdate":  regexp.MustCompile(`([A-Za-z]+ \d{1,2}, \d{4})`),
	}
	fields, err := parser.ExtractFields(m.Text, patterns)
	if err != nil {
		return nil, err
	}

	return parser.BuildTransaction(
		m,
		fields,
		"rbc",
		"CAD",
		domain.In,
		"RBC Deposit",
	)
}
