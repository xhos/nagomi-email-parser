package wise

import (
	"path/filepath"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
)

func TestWiseSpendParserForwarded(t *testing.T) {
	assertTransaction(
		t,
		&spend{},
		filepath.Join("testdata", "fw-wise-spend.decoded.eml"),
		"test-wise-spend-fwd-id",
		expectedTransactionDetails{
			Account:     "",
			Amount:      "15.00",
			Date:        time.Date(2026, time.February, 24, 0, 0, 0, 0, time.UTC),
			Currency:    "CAD",
			Direction:   domain.Out,
			Description: "Independent City Market",
		},
	)
}
