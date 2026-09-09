package smtp

import (
	"fmt"
	"nagomi-email-parser/internal/api"
	"nagomi-email-parser/internal/domain"
	"nagomi-email-parser/internal/email"
	_ "nagomi-email-parser/internal/email/all"
	pb "nagomi-email-parser/internal/gen/nagomi/v1"
	"nagomi-email-parser/internal/parser"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type EmailHandler struct {
	API           *api.Client
	Log           *log.Logger
	UnsafeSaveEML bool
}

func NewEmailHandler(apiClient *api.Client, log *log.Logger, unsafeSaveEML bool) *EmailHandler {
	return &EmailHandler{
		API:           apiClient,
		Log:           log.WithPrefix("handler"),
		UnsafeSaveEML: unsafeSaveEML,
	}
}

func (h *EmailHandler) ProcessEmail(userUUID, from string, to []string, data []byte) error {
	h.Log.Info("processing email", "user_uuid", userUUID, "from", from)

	if h.UnsafeSaveEML {
		if err := h.saveEmailToFile(userUUID, from, data); err != nil {
			h.Log.Warn("failed to save debug email file", "err", err)
		}
	}

	// resolve user id
	user, err := h.API.GetUser(userUUID)
	if err != nil {
		h.Log.Error("user not found", "user_uuid", userUUID, "err", err)
		return nil // accept email to avoid retries
	}
	userID := user.Id
	h.Log.Info("found user", "user_id", userID)

	msg, decoded, err := email.ParseMessage(data)
	if err != nil {
		h.Log.Error("failed to parse email message", "user_uuid", userUUID, "from", from, "err", err)
		return nil
	}

	meta, err := parser.ToEmailMeta(fmt.Sprintf("%s-%d", userUUID, len(data)), msg, decoded)
	if err != nil {
		h.Log.Error("failed to parse email metadata", "user_uuid", userUUID, "from", from, "err", err)
		return nil
	}

	prsr := parser.Find(meta)
	if prsr == nil {
		h.Log.Warn("no parser matched for email", "user_uuid", userUUID, "from", from, "subject", meta.Subject)
		return nil
	}

	txn, err := prsr.Parse(meta)
	if err != nil {
		h.Log.Error("parser failed to extract transaction", "user_uuid", userUUID, "from", from, "subject", meta.Subject, "err", err)
		return nil
	}
	if txn == nil {
		return nil
	}

	h.Log.Debug("parsed transaction",
		"user_uuid", userUUID,
		"email_id", txn.EmailID,
		"tx_date", txn.TxDate,
		"bank", txn.TxBank,
		"account", txn.TxAccount,
		"amount", txn.TxAmount,
		"currency", txn.TxCurrency,
		"direction", txn.TxDirection,
		"description", txn.TxDesc,
	)

	accounts, err := h.API.GetAccounts(userID)
	if err != nil {
		h.Log.Error("failed to fetch accounts", "user_uuid", userUUID, "err", err)
		return nil
	}

	accountMap := make(map[string]int, len(accounts))
	for _, acc := range accounts {
		if acc.Name == "" {
			continue
		}
		bankPrefix := strings.ToLower(acc.Bank)
		accountMap[fmt.Sprintf("%s-%s", bankPrefix, acc.Name)] = int(acc.Id)
		for _, alias := range acc.Aliases {
			accountMap[fmt.Sprintf("%s-%s", bankPrefix, alias)] = int(acc.Id)
		}
	}

	if err := h.resolveAccount(userUUID, txn, accountMap, user); err != nil {
		h.Log.Error("failed to resolve account", "user_uuid", userUUID, "from", from, "err", err)
		return nil
	}

	if err := h.API.CreateTransaction(userID, txn); err != nil {
		h.Log.Error("failed to create transaction", "user_uuid", userUUID, "from", from, "err", err)
		return nil
	}

	h.Log.Info("transaction created successfully", "user_uuid", userUUID, "from", from, "bank", txn.TxBank, "amount", txn.TxAmount, "currency", txn.TxCurrency)

	return nil
}

func (h *EmailHandler) saveEmailToFile(userUUID, from string, data []byte) error {
	const debugDir = "debug_emails"
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	msg, decoded, err := email.ParseMessage(data)
	if err != nil {
		return fmt.Errorf("failed to parse email for debug file: %w", err)
	}

	subject := msg.Header.Get("Subject")
	sanitizedSubject := sanitizeFilename(subject)
	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("%s_%s_%s_%s", userUUID, timestamp, sanitizedSubject, strings.ReplaceAll(from, "@", "_at_"))

	content := fmt.Sprintf("Subject: %s\nFrom: %s\nTo: %s\nDate: %s\n\n%s",
		subject,
		msg.Header.Get("From"),
		msg.Header.Get("To"),
		msg.Header.Get("Date"),
		decoded,
	)
	filePath := filepath.Join(debugDir, baseName+".decoded.eml")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write debug email file: %w", err)
	}

	h.Log.Info("saved debug email file", "path", filePath, "size", len(content), "subject", subject)
	return nil
}

func (h *EmailHandler) resolveAccount(userUUID string, txn *domain.Transaction, accountMap map[string]int, user *pb.User) error {
	noAccountParsed := txn.TxAccount == ""

	if noAccountParsed {
		h.Log.Warn("no account parsed from email; skipping transaction", "user_uuid", userUUID, "bank", txn.TxBank)
		return nil
	}

	cleanAccount := strings.TrimLeft(txn.TxAccount, "*")
	accountKey := fmt.Sprintf("%s-%s", strings.ToLower(txn.TxBank), cleanAccount)

	if existingAccountID, exists := accountMap[accountKey]; exists {
		txn.AccountID = existingAccountID
		return nil
	}

	account, err := h.API.CreateAccount(userUUID, cleanAccount, txn.TxBank, txn.TxCurrency)
	if err != nil {
		return fmt.Errorf("failed to create account for %s-%s: %w", txn.TxBank, cleanAccount, err)
	}

	txn.AccountID = int(account.Id)
	return nil
}

func sanitizeFilename(subject string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	sanitized := subject
	for _, char := range invalid {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	const maxFilenameLength = 50
	if len(sanitized) > maxFilenameLength {
		sanitized = sanitized[:maxFilenameLength]
	}

	sanitized = strings.TrimRight(sanitized, "_")

	if sanitized == "" {
		return "no-subject"
	}

	return sanitized
}
