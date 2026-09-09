package rbc

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestPaymentParser(t *testing.T) {
	assertTransaction(
		t,
		&payment{},
		filepath.Join("testdata", "payment-made.decoded.eml"),
		"test-payment-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "415.54",
			Date:        time.Date(2025, time.September, 12, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "RBC Payment",
		},
	)
}

func TestPaymentParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&payment{},
		filepath.Join("testdata", "fw-payment-made.decoded.eml"),
		"test-payment-fwd-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "500.00",
			Date:        time.Date(2025, time.September, 5, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "RBC Payment",
		},
	)
}
