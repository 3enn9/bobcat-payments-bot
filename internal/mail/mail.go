package mail

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Service struct {
	Host string
	Port string
	User string
	Pass string
	From string

	IMAPHost string
	IMAPPort string
	IMAPUser string
	IMAPPass string
}

type SentCopyError struct {
	Err error
}

func (e *SentCopyError) Error() string {
	return fmt.Sprintf("письмо отправлено, но копию не удалось сохранить в папку «Отправленные»: %v", e.Err)
}

func (e *SentCopyError) Unwrap() error { return e.Err }

func NewService(host, port, user, pass, from, imapHost, imapPort, imapUser, imapPass string) *Service {
	return &Service{
		Host:     strings.TrimSpace(host),
		Port:     strings.TrimSpace(port),
		User:     strings.TrimSpace(user),
		Pass:     pass,
		From:     strings.TrimSpace(from),
		IMAPHost: strings.TrimSpace(imapHost),
		IMAPPort: strings.TrimSpace(imapPort),
		IMAPUser: strings.TrimSpace(imapUser),
		IMAPPass: imapPass,
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
	body.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	body.WriteString("Счёт во вложении.\r\n\r\n")

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: application/pdf\r\n")
	body.WriteString("Content-Transfer-Encoding: base64\r\n")
	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": fileName})
	body.WriteString("Content-Disposition: " + contentDisposition + "\r\n\r\n")
	body.WriteString(wrapBase64(pdf))
	body.WriteString("\r\n--" + boundary + "--\r\n")

	auth := smtp.PlainAuth("", s.User, s.Pass, s.Host)
	var sendErr error
	if port == "465" {
		sendErr = sendTLS(addr, s.From, []string{to}, body.Bytes(), auth)
	} else {
		sendErr = smtp.SendMail(addr, auth, s.From, []string{to}, body.Bytes())
	}
	if sendErr != nil {
		return sendErr
	}
	if err := s.appendSentCopy(body.Bytes()); err != nil {
		return &SentCopyError{Err: err}
	}
	return nil
}

func (s *Service) appendSentCopy(message []byte) error {
	if s.IMAPHost == "" || s.IMAPUser == "" {
		return fmt.Errorf("IMAP не настроен")
	}
	port := s.IMAPPort
	if port == "" {
		port = "993"
	}
	addr := net.JoinHostPort(s.IMAPHost, port)
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 15 * time.Second},
		"tcp",
		addr,
		&tls.Config{ServerName: s.IMAPHost, MinVersion: tls.VersionTLS12},
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReader(conn)

	greeting, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToUpper(greeting), "* OK") &&
		!strings.HasPrefix(strings.ToUpper(greeting), "* PREAUTH") {
		return fmt.Errorf("неожиданный ответ IMAP: %s", strings.TrimSpace(greeting))
	}

	user, err := quoteIMAP(s.IMAPUser)
	if err != nil {
		return err
	}
	pass, err := quoteIMAP(s.IMAPPass)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(conn, "A001 LOGIN %s %s\r\n", user, pass); err != nil {
		return err
	}
	if err := readIMAPTaggedResponse(reader, "A001"); err != nil {
		return err
	}

	mailbox, err := quoteIMAP("Sent")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(conn, "A002 APPEND %s (\\Seen) {%d}\r\n", mailbox, len(message)); err != nil {
		return err
	}
	continuation, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.TrimSpace(continuation), "+") {
		return fmt.Errorf("IMAP APPEND отклонён: %s", strings.TrimSpace(continuation))
	}
	if err := writeIMAPAll(conn, message); err != nil {
		return err
	}
	if err := writeIMAPAll(conn, []byte("\r\n")); err != nil {
		return err
	}
	return readIMAPTaggedResponse(reader, "A002")
}

func quoteIMAP(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("недопустимый символ в IMAP-параметре")
	}
	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	return `"` + quoted + `"`, nil
}

func readIMAPTaggedResponse(reader *bufio.Reader, tag string) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, tag+" ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[1], "OK") {
			return fmt.Errorf("IMAP команда отклонена: %s", line)
		}
		return nil
	}
}

func writeIMAPAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		written, err := conn.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return fmt.Errorf("соединение IMAP закрыто")
		}
		data = data[written:]
	}
	return nil
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
	if email == "" || len(email) > 255 || strings.ContainsAny(email, "\r\n\t ") {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || at >= len(email)-1 {
		return false
	}
	dot := strings.LastIndex(email[at:], ".")
	return dot > 1
}
