package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Service struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

func NewService(host, port, user, pass, from string) *Service {
	return &Service{
		Host: strings.TrimSpace(host),
		Port: strings.TrimSpace(port),
		User: strings.TrimSpace(user),
		Pass: pass,
		From: strings.TrimSpace(from),
	}
}

func (s *Service) Enabled() bool {
	return s != nil && s.Host != "" && s.From != ""
}

func (s *Service) SendPDF(to, subject, fileName string, pdf []byte) error {
	if !s.Enabled() {
		return fmt.Errorf("SMTP не настроен")
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("пустой email получателя")
	}

	port := s.Port
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(s.Host, port)

	var body bytes.Buffer
	boundary := "paymentsbot-invoice"
	body.WriteString("From: " + s.From + "\r\n")
	body.WriteString("To: " + to + "\r\n")
	body.WriteString("Subject: " + subject + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	body.WriteString("Счёт во вложении.\r\n\r\n")

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: application/pdf\r\n")
	body.WriteString("Content-Transfer-Encoding: base64\r\n")
	body.WriteString("Content-Disposition: attachment; filename=\"" + fileName + "\"\r\n\r\n")
	body.WriteString(wrapBase64(pdf))
	body.WriteString("\r\n--" + boundary + "--\r\n")

	auth := smtp.PlainAuth("", s.User, s.Pass, s.Host)
	if port == "465" {
		return sendTLS(addr, s.From, []string{to}, body.Bytes(), auth)
	}
	return smtp.SendMail(addr, auth, s.From, []string{to}, body.Bytes())
}

func sendTLS(addr, from string, to []string, msg []byte, auth smtp.Auth) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func wrapBase64(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		b.WriteString(encoded[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || at >= len(email)-1 {
		return false
	}
	dot := strings.LastIndex(email[at:], ".")
	return dot > 1
}
