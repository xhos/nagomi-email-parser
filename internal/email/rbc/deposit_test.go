package rbc

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestDepositParser(t *testing.T) {
	assertTransaction(
		t,
		&deposit{},
		filepath.Join("testdata", "deposit-notice.decoded.eml"),
		"test-deposit-id",
		expectedTransactionDetails{
			Account:     "Savings",
			Amount:      "2.65",
			Date:        time.Date(2025, time.September, 4, 1, 20, 54, 0, time.FixedZone("", -6*60*60)),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "RBC Deposit",
		},
	)
}

func TestDepositParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&deposit{},
		filepath.Join("testdata", "fw-deposit-notice.decoded.eml"),
		"test-deposit-fwd-id",
		expectedTransactionDetails{
			Account:     "Savings",
			Amount:      "1183.98",
			Date:        time.Date(2025, time.August, 28, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.In,
			Description: "RBC Deposit",
		},
	)
}
