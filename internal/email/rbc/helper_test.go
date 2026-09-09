package rbc

import (
	"fmt"
	"os"
	"testing"
	"time"

	"nagomi-email-parser/internal/domain"
	"nagomi-email-parser/internal/email"
	"nagomi-email-parser/internal/parser"
)

type expectedTransactionDetails struct {
	Account     string
	Amount      string
	Date        time.Time
	Currency    string
	Direction   domain.Direction
	Description string
}

func assertTransaction(
	t *testing.T,
	p parser.Parser,
	fixturePath string,
	emailID string,
	expected expectedTransactionDetails,
) {
	t.Helper()

	rawBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixturePath, err)
	}

	msg, content, err := email.ParseMessage(rawBytes)
	if err != nil {
		t.Fatalf("failed to parse email message from %s: %v", fixturePath, err)
	}

	meta, err := parser.ToEmailMeta(emailID, msg, content)
	if err != nil {
		t.Fatalf("toEmailMeta failed for %s: %v", fixturePath, err)
	}

	if !p.Match(meta) {
		t.Fatalf("Match(meta) was false for %s; subject=%q", fixturePath, meta.Subject)
	}

	tx, err := p.Parse(meta)
	if err != nil {
		t.Fatalf("Parse(meta) returned error for %s: %v", fixturePath, err)
	}
	if tx == nil {
		t.Fatalf("Parse(meta) returned nil Transaction for %s but expected a real one", fixturePath)
	}

	if tx.TxAccount != expected.Account {
		t.Errorf("Account = %q; want %q (fixture: %s)", tx.TxAccount, expected.Account, fixturePath)
	}
	if fmt.Sprintf("%.2f", tx.TxAmount) != expected.Amount {
		t.Errorf("Amount = %q; want %q (fixture: %s)", fmt.Sprintf("%.2f", tx.TxAmount), expected.Amount, fixturePath)
	}
	if !tx.TxDate.Equal(expected.Date) {
		t.Errorf("TxnDate = %v; want %v (fixture: %s)", tx.TxDate, expected.Date, fixturePath)
	}
	if tx.TxDirection != expected.Direction {
		t.Errorf("Direction = %v; want %v (fixture: %s)", tx.TxDirection, expected.Direction, fixturePath)
	}
	if tx.TxDesc != expected.Description {
		t.Errorf("Merchant = %q; want %q (fixture: %s)", tx.TxDesc, expected.Description, fixturePath)
	}
}
