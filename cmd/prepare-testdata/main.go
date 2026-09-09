// processes .eml files in testdata directories, extracts content and anonymizes sensitive data
// no arguments needed, it finds all testdata dirs under ./internal/email

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"nagomi-email-parser/internal/email"

	"github.com/charmbracelet/log"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Level:           log.InfoLevel,
		Prefix:          "prepare-testdata",
	})

	testDataDirs, err := findTestDataDirs("./internal/email")
	if err != nil {
		logger.Fatal("Failed to find testdata directories", "error", err)
	}

	totalProcessed := 0
	for _, dir := range testDataDirs {
		processed, err := processTestDataDir(logger, dir)
		if err != nil {
			logger.Error("Failed to process directory", "dir", dir, "error", err)
			continue
		}
		totalProcessed += processed
	}

	logger.Info("Processed", "total", totalProcessed)
}

func findTestDataDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "testdata" {
			dirs = append(dirs, path)
		}
		return nil
	})

	return dirs, err
}

func processTestDataDir(logger *log.Logger, dir string) (int, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		filePath := filepath.Join(dir, filename)

		isAlreadyDecoded := strings.HasSuffix(filename, ".decoded.eml")
		isRawEmailFile := strings.HasSuffix(filename, ".eml") && !isAlreadyDecoded

		if isRawEmailFile {
			if err := processEmailFile(filePath); err != nil {
				logger.Error("Failed to process raw email", "file", filePath, "error", err)
				continue
			}
			processed++
			continue
		}

		if isAlreadyDecoded {
			if err := anonymizeDecodedFile(filePath); err != nil {
				logger.Error("Failed to anonymize decoded file", "file", filePath, "error", err)
				continue
			}
			processed++
		}
	}

	return processed, nil
}

func processEmailFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	msg, content, err := email.ParseMessage(data)
	if err != nil {
		return fmt.Errorf("failed to parse email - cannot extract subject: %w", err)
	}

	subject := msg.Header.Get("Subject")
	if subject == "" {
		subject = "no-subject"
	}

	decodedPath := createDecodedPath(filePath, subject)

	decodedEmail := fmt.Sprintf(`Subject: %s
From: %s
To: %s
Date: %s

%s`,
		subject,
		anonymizeEmail(msg.Header.Get("From")),
		anonymizeEmail(msg.Header.Get("To")),
		msg.Header.Get("Date"),
		anonymizeContent(content))

	return os.WriteFile(decodedPath, []byte(decodedEmail), 0644)
}

func anonymizeDecodedFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)

	content = anonymizeContent(content)

	return os.WriteFile(filePath, []byte(content), 0644)
}

func createDecodedPath(originalPath, subject string) string {
	dir := filepath.Dir(originalPath)
	cleanSubject := sanitizeSubject(subject)

	baseFilename := cleanSubject + ".decoded.eml"
	decodedPath := filepath.Join(dir, baseFilename)

	if _, err := os.Stat(decodedPath); os.IsNotExist(err) {
		return decodedPath
	}

	counter := 1
	for {
		numberedFilename := fmt.Sprintf("%s-%d.decoded.eml", cleanSubject, counter)
		numberedPath := filepath.Join(dir, numberedFilename)

		if _, err := os.Stat(numberedPath); os.IsNotExist(err) {
			return numberedPath
		}

		counter++
	}
}

func sanitizeSubject(subject string) string {
	clean := strings.ToLower(subject)
	clean = strings.ReplaceAll(clean, " ", "-")

	var result strings.Builder
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	clean = result.String()

	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}
	clean = strings.Trim(clean, "-")

	if clean == "" {
		clean = "no-subject"
	}

	return clean
}

func anonymizeEmail(header string) string {
	if header == "" {
		return ""
	}

	nameEmailPattern := regexp.MustCompile(`^(.+?)\s*<([^>]+)>$`)
	emailOnlyPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	if nameEmailPattern.MatchString(header) {
		return "Example <email@example.com>"
	}

	if emailOnlyPattern.MatchString(strings.TrimSpace(header)) {
		return "email@example.com"
	}

	return emailPattern.ReplaceAllString(header, "email@example.com")
}

func anonymizeContent(content string) string {
	httpPattern := regexp.MustCompile(`https?://[^\s"'<>)\]]+`)
	content = httpPattern.ReplaceAllString(content, "https://example.com")

	fromPattern := regexp.MustCompile(`From:\s*(.+?)\s*<([^>]+)>`)
	content = fromPattern.ReplaceAllString(content, "From: Example <email@example.com>")

	toQuotedPattern := regexp.MustCompile(`To:\s*"[^"]*"\s*<[^>]+>`)
	content = toQuotedPattern.ReplaceAllString(content, "To: Example <email@example.com>")

	toPattern := regexp.MustCompile(`To:\s*([^<\s]+)\s*<([^>]+)>`)
	content = toPattern.ReplaceAllString(content, "To: Example <email@example.com>")

	return content
}
