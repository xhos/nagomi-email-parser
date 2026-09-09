package rbc

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestCreditParser(t *testing.T) {
	assertTransaction(
		t,
		&credit{},
		filepath.Join("testdata", "you-received-a-credit.decoded.eml"),
		"test-credit-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "840.72",
			Date:        time.Date(2025, time.June, 10, 17, 42, 0, 0, time.FixedZone("", -6*60*60)),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "SOME MERCHANT",
		},
	)
}

func TestCreditParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&credit{},
		filepath.Join("testdata", "fw-you-received-a-credit.decoded.eml"),
		"test-credit-fwd-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "840.72",
			Date:        time.Date(2025, time.June, 10, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "SOME MERCHANT",
		},
	)
}
