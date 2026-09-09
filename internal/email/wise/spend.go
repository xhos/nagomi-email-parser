package wise

import (
	"regexp"
	"strings"

	"nagomi-email-parser/internal/domain"
	"nagomi-email-parser/internal/parser"
)

func init() { parser.Register(&spend{}) }

type spend struct{}

var reOrdinal = regexp.MustCompile(`(\d+)(?:st|nd|rd|th)`)

func (s *spend) Match(m parser.EmailMeta) bool {
	return strings.Contains(m.Subject, "spent at") &&
		strings.Contains(m.Text, "Wise")
}

func (s *spend) Parse(m parser.EmailMeta) (*domain.Transaction, error) {
	// strip ordinal suffixes so e.g. "24th" becomes "24", enabling date matching
	normalized := reOrdinal.ReplaceAllString(m.Text, "$1")

	fields, err := parser.ExtractFields(normalized, map[string]*regexp.Regexp{
		"amount":   regexp.MustCompile(`(?i)You spent (\d+(?:\.\d+)?)`),
		"currency": regexp.MustCompile(`(?i)You spent \d+(?:\.\d+)? ([A-Z]{3}) at`),
		"txdate":   regexp.MustCompile(`([A-Za-z]+ \d{1,2}, \d{4})`),
		"desc":     regexp.MustCompile(`(?i)You spent \d+(?:\.\d+)? [A-Z]{3} at ([^.]+)\.`),
	})
	if err != nil {
		return nil, err
	}

	return parser.BuildTransaction(
		m,
		fields,
		"wise",
		fields["currency"],
		domain.Out,
		strings.TrimSpace(fields["desc"]),
	)
}
