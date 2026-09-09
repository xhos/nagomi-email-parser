package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/mhale/smtpd"
)

type Handler interface {
	ProcessEmail(userID string, from string, to []string, data []byte) error
}

type Server struct {
	addr        string
	domain      string
	handler     Handler
	log         *log.Logger
	smtpServer  *smtpd.Server
	tlsCert     string
	tlsKey      string
	tlsRequired bool
}

func NewServer(addr, domain string, handler Handler, logger *log.Logger) *Server {
	return &Server{
		addr:    addr,
		domain:  domain,
		handler: handler,
		log:     logger.WithPrefix("smtp"),
	}
}

func (s *Server) WithTLS(certFile, keyFile string, required bool) *Server {
	s.tlsCert = certFile
	s.tlsKey = keyFile
	s.tlsRequired = required
	return s
}

func (s *Server) Start(ctx context.Context) error {
	s.smtpServer = &smtpd.Server{
		Addr:     s.addr,
		Handler:  s.mailHandler,
		Appname:  "nagomi-email-parser",
		Hostname: s.domain,
	}

	// configure TLS if certificates are provided
	if s.tlsCert != "" && s.tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(s.tlsCert, s.tlsKey)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		s.smtpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   s.domain,
		}
		s.smtpServer.TLSRequired = s.tlsRequired

		if s.tlsRequired {
			s.log.Info("starting smtp server with TLS (required)", "addr", s.addr, "domain", s.domain)
		} else {
			s.log.Info("starting smtp server with TLS (opportunistic)", "addr", s.addr, "domain", s.domain)
		}
	} else {
		s.log.Info("starting smtp server without TLS", "addr", s.addr, "domain", s.domain)
	}

	go func() {
		<-ctx.Done()
		s.log.Info("stopping smtp server")
		if err := s.smtpServer.Close(); err != nil {
			s.log.Error("error stopping smtp server", "err", err)
		}
	}()

	return s.smtpServer.ListenAndServe()
}

var uuidPattern = regexp.MustCompile(`^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})@`)

func (s *Server) mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	s.log.Info("received email", "from", from, "to", to, "size", len(data))

	// extract uuid from first recipient
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}

	recipient := strings.ToLower(to[0])

	matches := uuidPattern.FindStringSubmatch(recipient)
	if len(matches) < 2 {
		s.log.Warn("invalid recipient format", "to", recipient, "from", from)
		return fmt.Errorf("invalid recipient format, expected <uuid>@domain")
	}

	userID := matches[1]

	return s.handler.ProcessEmail(userID, from, to, data)
}
