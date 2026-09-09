package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"nagomi-email-parser/internal/api"
	"nagomi-email-parser/internal/config"
	"nagomi-email-parser/internal/grpc"
	"nagomi-email-parser/internal/smtp"
	"nagomi-email-parser/internal/version"

	"github.com/charmbracelet/log"
)

func main() {
	cfg := config.Load()

	// ----- logger -----------------

	logWriter := io.Writer(os.Stdout)
	logFormatter := log.TextFormatter

	if cfg.LogFormat != "text" {
		logFormatter = log.JSONFormatter
	}

	logger := log.NewWithOptions(logWriter, log.Options{
		ReportTimestamp: true,
		Prefix:          "email-parser",
		Level:           cfg.LogLevel,
		Formatter:       logFormatter,
	})

	logger.Info("starting email-parser", "version", version.FullVersion())

	logger.Debug("debug is enabled")

	if cfg.UnsafeSaveEML {
		logger.Warn("UNSAFE_SAVE_EML is enabled, emails will be saved to disk")
	}

	// ----- api client -------------
	apiClient, err := api.NewClient(cfg.NagomiCoreURL, "", cfg.APIKey)
	if err != nil {
		logger.Fatal("api client init", "err", err)
	}
	defer func() {
		if err := apiClient.Close(); err != nil {
			logger.Error("failed to close gRPC connection", "err", err)
		}
	}()

	// ----- connectivity check -----
	logger.Info("checking nagomi-core connectivity", "url", cfg.NagomiCoreURL)
	if err := apiClient.Ping(); err != nil {
		logger.Fatal("nagomi-core not reachable", "err", err)
	}
	logger.Info("nagomi-core connectivity confirmed")

	// ----- services ---------------
	handler := smtp.NewEmailHandler(apiClient, logger, cfg.UnsafeSaveEML)
	smtpServer := smtp.NewServer(cfg.SMTPAddress, cfg.Domain, handler, logger)
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		smtpServer = smtpServer.WithTLS(cfg.TLSCert, cfg.TLSKey, cfg.TLSRequired)
	}

	grpcHealthSrv, err := grpc.NewHealthServer(cfg.GRPCAddress)
	if err != nil {
		logger.Fatal("grpc health server init", "err", err)
	}

	// ----- servers ----------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Info("grpc health server starting", "address", cfg.GRPCAddress)
		if err := grpcHealthSrv.Start(); err != nil {
			logger.Error("grpc health server error", "err", err)
		}
	}()

	go func() {
		if err := smtpServer.Start(ctx); err != nil {
			logger.Fatal("smtp server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down. bye!")

	cancel()
	grpcHealthSrv.Stop()
}
