package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nagomi-email-parser/internal/domain"
)

// ExtractFields applies each regex to the email body and returns the single capture group for each key
// The "account" field is optional - if not found, it will be set to empty string
func ExtractFields(emailBody string, patterns map[string]*regexp.Regexp) (map[string]string, error) {
	out := make(map[string]string, len(patterns))
	for key, re := range patterns {
		m := re.FindStringSubmatch(emailBody)
		if len(m) < 2 {
			if key == "account" {
				out[key] = ""
				continue
			}
			return nil, fmt.Errorf("field %q not found in text", key)
		}
		out[key] = m[1]
	}

	return out, nil
}

// ParseEmailDate normalizes a string into time.Time
func ParseEmailDate(raw string) (time.Time, error) {
	return time.Parse("January 2, 2006", raw)
}

// BuildTransaction assembles a domain.Transaction
func BuildTransaction(
	m EmailMeta,
	fields map[string]string,
	bank string,
	currency string,
	dir domain.Direction,
	desc string,
) (*domain.Transaction, error) {

	recv := time.Now()
	if m.Date != "" {
		if parsed, err := time.Parse(time.RFC3339, m.Date); err == nil {
			recv = parsed
		} else if parsed, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", m.Date); err == nil {
			recv = parsed
		}
	}

	bodyDate, err := ParseEmailDate(fields["txdate"])
	if err != nil {
		return nil, err
	}

	// decide which timestamp to keep
	var final time.Time
	if recv.Year() == bodyDate.Year() && recv.YearDay() == bodyDate.YearDay() {
		final = recv
	} else {
		final = bodyDate
	}

	rawAmt := strings.ReplaceAll(fields["amount"], ",", "")
	amt, err := strconv.ParseFloat(rawAmt, 64)
	if err != nil {
		return nil, err
	}

	return &domain.Transaction{
		EmailID:         m.ID,
		TxDate:          final,
		TxBank:          bank,
		TxAccount:       fields["account"],
		TxAmount:        amt,
		TxCurrency:      currency,
		TxDirection:     dir,
		TxDesc:          desc,
		Category:        "",
		UserNotes:       "",
		ForeignAmount:   nil,
		ForeignCurrency: nil,
		ExchangeRate:    nil,
	}, nil
}
