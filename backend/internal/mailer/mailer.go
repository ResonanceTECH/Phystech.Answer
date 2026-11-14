// Package mailer — минимальный SMTP-клиент для писем продукта
// (восстановление пароля). В разработке работает с Mailpit.
package mailer

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Mailer struct {
	host string
	port string
	user string
	pass string
	from string
}

func New(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

// Enabled — почта настроена (SMTP_HOST задан). Без него письма не уходят,
// но API восстановления остаётся работоспособным.
func (m *Mailer) Enabled() bool { return m.host != "" }

// envelopeAddr — bare email для SMTP-конверта (MAIL FROM). SMTP_FROM может
// содержать display-name вида «Физтех.Ответ <no-reply@…>», который в
// конверт класть нельзя.
func (m *Mailer) envelopeAddr() string {
	if i := strings.LastIndex(m.from, "<"); i >= 0 {
		if j := strings.Index(m.from[i:], ">"); j > 0 {
			return strings.TrimSpace(m.from[i+1 : i+j])
		}
	}
	return strings.TrimSpace(m.from)
}

// fromHeader — значение заголовка From с RFC 2047-кодированием display-name,
// если в нём есть не-ASCII символы.
func (m *Mailer) fromHeader() string {
	i := strings.LastIndex(m.from, "<")
	if i < 0 {
		return m.from
	}
	name := strings.TrimSpace(m.from[:i])
	addr := strings.TrimSpace(m.from[i:])
	for _, r := range name {
		if r > 127 {
			name = "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(name)) + "?="
			break
		}
	}
	return name + " " + addr
}

// Send отправляет письмо с текстовой и HTML-версиями (multipart/alternative).
func (m *Mailer) Send(to, subject, text, html string) error {
	if !m.Enabled() {
		return fmt.Errorf("mailer: SMTP not configured")
	}
	addr := net.JoinHostPort(m.host, m.port)

	var b strings.Builder
	b.WriteString("From: " + m.fromHeader() + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: =?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?=\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"phystech-mime\"\r\n\r\n")
	b.WriteString("--phystech-mime\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	b.WriteString(text + "\r\n")
	b.WriteString("--phystech-mime\r\n")
	b.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	b.WriteString(html + "\r\n")
	b.WriteString("--phystech-mime--\r\n")

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("mailer: dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mailer: client: %w", err)
	}
	defer client.Close()

	if m.user != "" {
		if err := client.Auth(smtp.PlainAuth("", m.user, m.pass, m.host)); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	if err := client.Mail(m.envelopeAddr()); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write([]byte(b.String())); err != nil {
		return fmt.Errorf("mailer: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close data: %w", err)
	}
	return client.Quit()
}
