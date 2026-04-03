package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	texttmpl "text/template"
	"time"

	"github.com/rs/zerolog"

	"backend/internal/config"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type Service struct {
	cfg         config.SMTPConfig
	logger      zerolog.Logger
	subjects    *texttmpl.Template
	textBodies  *texttmpl.Template
	htmlBodies  *template.Template
	fromHeader  string
	fromAddress string
	enabled     bool
}

func New(cfg config.SMTPConfig, logger zerolog.Logger) (*Service, error) {
	svc := &Service{
		cfg:     cfg,
		logger:  logger.With().Str("component", "smtp_mailer").Logger(),
		enabled: cfg.Enabled,
	}
	if !cfg.Enabled {
		return svc, nil
	}

	fromAddr := strings.TrimSpace(cfg.FromAddress)
	parsedFrom, err := mail.ParseAddress(fromAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_FROM_ADDRESS: %w", err)
	}
	if strings.TrimSpace(parsedFrom.Address) == "" {
		return nil, fmt.Errorf("invalid SMTP_FROM_ADDRESS: empty address")
	}
	svc.fromAddress = parsedFrom.Address
	if strings.TrimSpace(cfg.FromName) != "" {
		svc.fromHeader = (&mail.Address{Name: cfg.FromName, Address: parsedFrom.Address}).String()
	} else {
		svc.fromHeader = parsedFrom.String()
	}

	funcs := texttmpl.FuncMap{
		"join": strings.Join,
	}
	subjects, err := texttmpl.New("subjects").Funcs(funcs).ParseFS(templateFS, "templates/*.subject.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse email subject templates failed: %w", err)
	}
	textBodies, err := texttmpl.New("text").Funcs(funcs).ParseFS(templateFS, "templates/*.text.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse email text templates failed: %w", err)
	}
	htmlBodies, err := template.New("html").ParseFS(templateFS, "templates/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse email html templates failed: %w", err)
	}
	svc.subjects = subjects
	svc.textBodies = textBodies
	svc.htmlBodies = htmlBodies
	return svc, nil
}

func (s *Service) Enabled() bool {
	return s != nil && s.enabled
}

func (s *Service) SendTemplate(ctx context.Context, templateName string, to []string, data any) error {
	if !s.Enabled() {
		return nil
	}
	recipients, err := normalizeRecipients(to)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}

	subject, textBody, htmlBody, err := s.render(templateName, data)
	if err != nil {
		return err
	}
	msg, err := s.buildMessage(recipients, subject, textBody, htmlBody)
	if err != nil {
		return err
	}
	return s.send(ctx, recipients, msg)
}

func (s *Service) render(templateName string, data any) (string, string, string, error) {
	subject, err := executeTextTemplate(s.subjects, templateName+".subject.tmpl", data)
	if err != nil {
		return "", "", "", fmt.Errorf("render email subject failed: %w", err)
	}
	textBody, err := executeTextTemplate(s.textBodies, templateName+".text.tmpl", data)
	if err != nil {
		return "", "", "", fmt.Errorf("render email text body failed: %w", err)
	}
	htmlBody, err := executeHTMLTemplate(s.htmlBodies, templateName+".html.tmpl", data)
	if err != nil {
		return "", "", "", fmt.Errorf("render email html body failed: %w", err)
	}
	return strings.TrimSpace(subject), textBody, htmlBody, nil
}

func executeTextTemplate(tpl *texttmpl.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func executeHTMLTemplate(tpl *template.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func normalizeRecipients(to []string) ([]string, error) {
	if len(to) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(to))
	out := make([]string, 0, len(to))
	for _, raw := range to {
		addr := strings.TrimSpace(raw)
		if addr == "" {
			continue
		}
		parsed, err := mail.ParseAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid recipient address %q: %w", raw, err)
		}
		normalized := strings.ToLower(strings.TrimSpace(parsed.Address))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, parsed.Address)
	}
	return out, nil
}

func (s *Service) buildMessage(to []string, subject, textBody, htmlBody string) ([]byte, error) {
	var mixed bytes.Buffer
	writer := multipart.NewWriter(&mixed)
	messageID := buildMessageID(s.fromAddress)
	dateHeader := time.Now().Format(time.RFC1123Z)

	var headers bytes.Buffer
	headers.WriteString("From: " + s.fromHeader + "\r\n")
	headers.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	headers.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	headers.WriteString("Date: " + dateHeader + "\r\n")
	headers.WriteString("Message-ID: " + messageID + "\r\n")
	headers.WriteString("Content-Language: vi\r\n")
	headers.WriteString("X-Mailer: backend\r\n")
	headers.WriteString("Auto-Submitted: auto-generated\r\n")
	headers.WriteString("MIME-Version: 1.0\r\n")
	headers.WriteString("Content-Type: multipart/alternative; boundary=" + writer.Boundary() + "\r\n")
	headers.WriteString("\r\n")

	if err := writeMIMEPart(writer, "text/plain; charset=UTF-8", textBody); err != nil {
		return nil, err
	}
	if err := writeMIMEPart(writer, "text/html; charset=UTF-8", htmlBody); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	headers.Write(mixed.Bytes())
	return headers.Bytes(), nil
}

func buildMessageID(fromAddress string) string {
	domain := "localhost"
	if at := strings.LastIndex(strings.TrimSpace(fromAddress), "@"); at >= 0 && at+1 < len(strings.TrimSpace(fromAddress)) {
		domain = strings.TrimSpace(fromAddress[at+1:])
	}
	return fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), os.Getpid(), domain)
}

func writeMIMEPart(writer *multipart.Writer, contentType, body string) error {
	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Type", contentType)
	partHeaders.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(part)
	if _, err := qp.Write([]byte(body)); err != nil {
		_ = qp.Close()
		return err
	}
	return qp.Close()
}

func (s *Service) send(ctx context.Context, to []string, msg []byte) error {
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	conn, client, err := s.dialSMTP(ctx, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	if username := strings.TrimSpace(s.cfg.Username); username != "" {
		auth := smtp.PlainAuth("", username, s.cfg.Password, s.cfg.Host)
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth failed: %w", err)
			}
		}
	}

	if err := client.Mail(s.fromAddress); err != nil {
		return fmt.Errorf("smtp MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp RCPT TO failed for %s: %w", recipient, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA failed: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write message failed: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp finalize message failed: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit failed: %w", err)
	}
	return nil
}

func (s *Service) dialSMTP(ctx context.Context, addr string) (net.Conn, *smtp.Client, error) {
	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	tlsConfig := &tls.Config{
		ServerName:         s.cfg.Host,
		InsecureSkipVerify: s.cfg.InsecureSkipVerify,
	}

	switch s.cfg.TLSMode {
	case "direct":
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("smtp direct TLS dial failed: %w", err)
		}
		if err := applyConnDeadline(ctx, conn, s.cfg.Timeout); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, nil, fmt.Errorf("create smtp client failed: %w", err)
		}
		return conn, client, nil
	default:
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf("smtp dial failed: %w", err)
		}
		if err := applyConnDeadline(ctx, conn, s.cfg.Timeout); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, nil, fmt.Errorf("create smtp client failed: %w", err)
		}
		if s.cfg.TLSMode == "starttls" {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				client.Close()
				_ = conn.Close()
				return nil, nil, fmt.Errorf("smtp server does not support STARTTLS")
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()
				_ = conn.Close()
				return nil, nil, fmt.Errorf("smtp STARTTLS failed: %w", err)
			}
		}
		return conn, client, nil
	}
}

func applyConnDeadline(ctx context.Context, conn net.Conn, fallback time.Duration) error {
	if conn == nil {
		return nil
	}
	if deadline, ok := ctx.Deadline(); ok {
		return conn.SetDeadline(deadline)
	}
	if fallback > 0 {
		return conn.SetDeadline(time.Now().Add(fallback))
	}
	return nil
}
