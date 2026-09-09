package rbc

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestWithdrawalParser(t *testing.T) {
	assertTransaction(
		t,
		&withdrawal{},
		filepath.Join("testdata", "withdrawal-warning.decoded.eml"),
		"test-withdrawal-id",
		expectedTransactionDetails{
			Account:     "Savings",
			Amount:      "110.84",
			Date:        time.Date(2025, time.August, 30, 1, 2, 43, 0, time.FixedZone("", -6*60*60)),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "RBC Withdrawal",
		},
	)
}

func TestWithdrawalParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&withdrawal{},
		filepath.Join("testdata", "fw-withdrawal-warning.decoded.eml"),
		"test-withdrawal-fwd-id",
		expectedTransactionDetails{
			Account:     "Daily",
			Amount:      "11.95",
			Date:        time.Date(2025, time.July, 15, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "RBC Withdrawal",
		},
	)
}
