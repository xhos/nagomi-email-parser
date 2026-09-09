package rbc

import (
	"regexp"
	"strings"

	"nagomi-email-parser/internal/domain"
	"nagomi-email-parser/internal/parser"
)

func init() { parser.Register(&credit{}) }

type credit struct{}

func (p *credit) Match(m parser.EmailMeta) bool {
	return strings.Contains(m.Subject, "You received a credit.") &&
		strings.Contains(m.Text, "RBC Royal Bank")
}

func (p *credit) Parse(m parser.EmailMeta) (*domain.Transaction, error) {
	patterns := map[string]*regexp.Regexp{
		"account": regexp.MustCompile(`(\*{12}\d+|\*+\d+)`),
		"amount":  regexp.MustCompile(`\$(\d+\.\d{2})`),
		"txdate":  regexp.MustCompile(`([A-Za-z]+ \d{1,2}, \d{4})`),
		"desc":    regexp.MustCompile(`from ([A-Z][A-Z' ]+[A-Z])`),
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
		strings.TrimSpace(fields["desc"]),
	)
}
