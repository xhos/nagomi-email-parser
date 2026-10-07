package smtp

import (
	"errors"
	"fmt"
	"mime"
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
	"unicode/utf8"

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
	h.Log.Info("found user", "user_id", user.Id)

	report := h.process(userUUID, user, from, data)
	report.UserId = user.Id
	if err := h.API.ReportEmail(report); err != nil {
		h.Log.Warn("failed to report email", "user_uuid", userUUID, "err", err)
	}

	// always accept: a sender retrying won't change the outcome
	return nil
}

// process imports the email and describes what happened for ReportEmail
func (h *EmailHandler) process(userUUID string, user *pb.User, from string, data []byte) *pb.ReportEmailRequest {
	report := &pb.ReportEmailRequest{From: truncate(from, maxFromLen)}
	failed := func(reason string) *pb.ReportEmailRequest {
		report.Outcome = pb.EmailOutcome_EMAIL_OUTCOME_FAILED
		reason = truncate(reason, maxErrorLen)
		report.Error = &reason
		return report
	}

	msg, decoded, err := email.ParseMessage(data)
	if err != nil {
		h.Log.Error("failed to parse email message", "user_uuid", userUUID, "from", from, "err", err)
		return failed("couldn't read the email")
	}

	if headerFrom := msg.Header.Get("From"); headerFrom != "" {
		report.From = truncate(decodeHeader(headerFrom), maxFromLen)
	}
	report.Subject = truncate(decodeHeader(msg.Header.Get("Subject")), maxSubjectLen)
	body := truncate(decoded, maxBodyLen)
	report.Body = &body

	meta, err := parser.ToEmailMeta(fmt.Sprintf("%s-%d", userUUID, len(data)), msg, decoded)
	if err != nil {
		h.Log.Error("failed to parse email metadata", "user_uuid", userUUID, "from", from, "err", err)
		return failed("couldn't read the email")
	}

	prsr := parser.Find(meta)
	if prsr == nil {
		h.Log.Warn("no parser matched for email", "user_uuid", userUUID, "from", from, "subject", meta.Subject)
		report.Outcome = pb.EmailOutcome_EMAIL_OUTCOME_UNRECOGNIZED
		return report
	}

	txn, err := prsr.Parse(meta)
	if err != nil {
		h.Log.Error("parser failed to extract transaction", "user_uuid", userUUID, "from", from, "subject", meta.Subject, "err", err)
		return failed("couldn't read the transaction: " + err.Error())
	}
	if txn == nil {
		report.Outcome = pb.EmailOutcome_EMAIL_OUTCOME_UNRECOGNIZED
		return report
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

	accounts, err := h.API.GetAccounts(user.Id)
	if err != nil {
		h.Log.Error("failed to fetch accounts", "user_uuid", userUUID, "err", err)
		return failed("couldn't load accounts")
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
		if errors.Is(err, errNoAccount) {
			return failed("no account number in the email")
		}
		return failed("couldn't create the account")
	}

	txID, err := h.API.CreateTransaction(user.Id, txn)
	if err != nil {
		h.Log.Error("failed to create transaction", "user_uuid", userUUID, "from", from, "err", err)
		return failed("couldn't create the transaction")
	}

	h.Log.Info("transaction created successfully", "user_uuid", userUUID, "from", from, "bank", txn.TxBank, "amount", txn.TxAmount, "currency", txn.TxCurrency)

	report.Outcome = pb.EmailOutcome_EMAIL_OUTCOME_IMPORTED
	report.Body = nil
	if txID != 0 {
		report.TransactionId = &txID
	}
	return report
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

var errNoAccount = errors.New("no account parsed from email")

func (h *EmailHandler) resolveAccount(userUUID string, txn *domain.Transaction, accountMap map[string]int, user *pb.User) error {
	noAccountParsed := txn.TxAccount == ""

	if noAccountParsed {
		return errNoAccount
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

// limits mirror ReportEmailRequest's validation, which counts characters
const (
	maxFromLen    = 500
	maxSubjectLen = 1000
	maxErrorLen   = 1000
	maxBodyLen    = 32768
)

func truncate(s string, maxChars int) string {
	// postgres text rejects NUL bytes and invalid UTF-8
	s = strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "\uFFFD")
	if utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	return string([]rune(s)[:maxChars])
}

// decodeHeader turns RFC 2047 encoded words into text, keeping the raw value on failure
func decodeHeader(v string) string {
	decoded, err := new(mime.WordDecoder).DecodeHeader(v)
	if err != nil {
		return v
	}
	return decoded
}
