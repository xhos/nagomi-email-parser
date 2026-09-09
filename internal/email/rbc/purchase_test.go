package rbc

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestPurchaseParser(t *testing.T) {
	assertTransaction(
		t,
		&purchase{},
		filepath.Join("testdata", "you-made-a-purchase.decoded.eml"),
		"test-purchase-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "1.77",
			Date:        time.Date(2025, time.September, 15, 8, 18, 20, 0, time.FixedZone("", -6*60*60)),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "TIM HORTONS #0000",
		},
	)
}

func TestPurchaseParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&purchase{},
		filepath.Join("testdata", "fw-you-made-a-purchase.decoded.eml"),
		"test-purchase-fwd-id",
		expectedTransactionDetails{
			Account:     "************1001",
			Amount:      "39.50",
			Date:        time.Date(2025, time.September, 13, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "SOME NO FRILLS 0000",
		},
	)
}

func TestPurchaseParserNoAccount(t *testing.T) {
	assertTransaction(
		t,
		&purchase{},
		filepath.Join("testdata", "you-made-a-purchase-no-account.decoded.eml"),
		"test-purchase-no-account-id",
		expectedTransactionDetails{
			Account:     "",
			Amount:      "1.77",
			Date:        time.Date(2025, time.September, 10, 8, 34, 35, 0, time.FixedZone("", -6*60*60)),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "TIM HORTONS #0000",
		},
	)
}

func TestPurchaseParserNoAccountForwarded(t *testing.T) {
	assertTransaction(
		t,
		&purchase{},
		filepath.Join("testdata", "fw-you-made-a-purchase-no-account.decoded.eml"),
		"test-purchase-fwd-no-account-id",
		expectedTransactionDetails{
			Account:     "",
			Amount:      "90.39",
			Date:        time.Date(2025, time.September, 3, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "AMZN Mktp CA",
		},
	)
}
