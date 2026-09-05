package mail

import (
	"fmt"
	"io"
	"log"
	"net/textproto"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type IMAPConfig struct {
	Host     string
	Port     string
	User     string
	Pass     string
	Mailbox  string
	From     string
	StartUID uint32
}

func (c IMAPConfig) Enabled() bool {
	return strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.User) != "" && c.Pass != ""
}

func (c IMAPConfig) addr() string {
	port := strings.TrimSpace(c.Port)
	if port == "" {
		port = "993"
	}
	return c.Host + ":" + port
}

func (c IMAPConfig) folder() string {
	if strings.TrimSpace(c.Mailbox) == "" {
		return "INBOX"
	}
	return c.Mailbox
}

type Attachment struct {
	Filename string
	Bytes    []byte
}

type Message struct {
	UID         uint32
	From        string
	Attachments []Attachment
}

func FetchNewPDF(cfg IMAPConfig, lastUID uint32) ([]Message, uint32, error) {
	c, err := dialAndLogin(cfg)
	if err != nil {
		return nil, lastUID, err
	}
	defer c.Logout()

	mbox, err := c.Select(cfg.folder(), true)
	if err != nil {
		return nil, lastUID, fmt.Errorf("imap select: %w", err)
	}

	fromUID := lastUID + 1
	if lastUID == 0 {
		if cfg.StartUID > 0 {
			fromUID = cfg.StartUID + 1
		} else if mbox.UidNext > 1 {
			log.Printf("accountant imap: курсор пустой, пропускаю старые письма, uid=%d", mbox.UidNext-1)
			return nil, mbox.UidNext - 1, nil
		} else {
			return nil, 0, nil
		}
	}

	criteria := imap.NewSearchCriteria()
	seq := new(imap.SeqSet)
	seq.AddRange(fromUID, 0)
	criteria.Uid = seq

	uids, err := c.UidSearch(criteria)
	if err != nil {
		return nil, lastUID, fmt.Errorf("imap search: %w", err)
	}

	var newUIDs []uint32
	for _, uid := range uids {
		if uid > lastUID {
			newUIDs = append(newUIDs, uid)
		}
	}
	if len(newUIDs) == 0 {
		return nil, lastUID, nil
	}

	return fetchMessagesByUID(c, cfg, newUIDs, lastUID)
}

// FetchPDFSince ищет письма с PDF от бухгалтера начиная с даты (включительно).
// UID знать не нужно — IMAP SINCE. Возвращает сообщения и максимальный UID в ящике.
func FetchPDFSince(cfg IMAPConfig, since time.Time) ([]Message, uint32, error) {
	c, err := dialAndLogin(cfg)
	if err != nil {
		return nil, 0, err
	}
	defer c.Logout()

	mbox, err := c.Select(cfg.folder(), true)
	if err != nil {
		return nil, 0, fmt.Errorf("imap select: %w", err)
	}
	maxUID := uint32(0)
	if mbox.UidNext > 1 {
		maxUID = mbox.UidNext - 1
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.UTC)
	if from := strings.TrimSpace(cfg.From); from != "" {
		criteria.Header = textproto.MIMEHeader{}
		criteria.Header.Add("From", from)
	}

	uids, err := c.UidSearch(criteria)
	if err != nil {
		return nil, 0, fmt.Errorf("imap search since: %w", err)
	}
	if len(uids) == 0 {
		return nil, maxUID, nil
	}

	messages, fetchMax, err := fetchMessagesByUID(c, cfg, uids, 0)
	if err != nil {
		return nil, 0, err
	}
	if fetchMax > maxUID {
		maxUID = fetchMax
	}
	return messages, maxUID, nil
}

func dialAndLogin(cfg IMAPConfig) (*imapclient.Client, error) {
	c, err := imapclient.DialTLS(cfg.addr(), nil)
	if err != nil {
		return nil, fmt.Errorf("imap dial: %w", err)
	}
	if err := c.Login(cfg.User, cfg.Pass); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	return c, nil
}

func fetchMessagesByUID(c *imapclient.Client, cfg IMAPConfig, uids []uint32, minUID uint32) ([]Message, uint32, error) {
	wantedFrom := strings.ToLower(strings.TrimSpace(cfg.From))
	var result []Message
	maxUID := minUID

	const batchSize = 25
	for start := 0; start < len(uids); start += batchSize {
		end := start + batchSize
		if end > len(uids) {
			end = len(uids)
		}
		batch := uids[start:end]

		fetchSet := new(imap.SeqSet)
		fetchSet.AddNum(batch...)
		section := &imap.BodySectionName{Peek: true}
		items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}
		ch := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() {
			done <- c.UidFetch(fetchSet, items, ch)
		}()

		for msg := range ch {
			if msg == nil {
				continue
			}
			if msg.Uid > maxUID {
				maxUID = msg.Uid
			}
			if minUID > 0 && msg.Uid <= minUID {
				continue
			}
			from := envelopeFrom(msg.Envelope)
			if wantedFrom != "" && strings.ToLower(from) != wantedFrom {
				continue
			}
			body := msg.GetBody(section)
			if body == nil {
				continue
			}
			attachments, err := extractPDFAttachments(body)
			if err != nil || len(attachments) == 0 {
				continue
			}
			result = append(result, Message{
				UID:         msg.Uid,
				From:        from,
				Attachments: attachments,
			})
		}
		if err := <-done; err != nil {
			return nil, minUID, fmt.Errorf("imap fetch: %w", err)
		}
	}
	return result, maxUID, nil
}

func envelopeFrom(env *imap.Envelope) string {
	if env == nil || len(env.From) == 0 {
		return ""
	}
	addr := env.From[0]
	if addr.MailboxName == "" {
		return ""
	}
	if addr.HostName == "" {
		return addr.MailboxName
	}
	return addr.MailboxName + "@" + addr.HostName
}

func extractPDFAttachments(r io.Reader) ([]Attachment, error) {
	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, err
	}
	defer mr.Close()

	var out []Attachment
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, nil
		}
		filename := partFilename(p)
		if filename == "" || !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
			continue
		}
		raw, err := io.ReadAll(p.Body)
		if err != nil || len(raw) == 0 {
			continue
		}
		out = append(out, Attachment{Filename: filename, Bytes: raw})
	}
	return out, nil
}

func partFilename(p *mail.Part) string {
	switch h := p.Header.(type) {
	case *mail.AttachmentHeader:
		name, err := h.Filename()
		if err == nil {
			return strings.Trim(name, `"'`)
		}
	case *mail.InlineHeader:
		ct, _, _ := h.ContentType()
		if strings.Contains(strings.ToLower(ct), "pdf") {
			if disp, params, err := h.ContentDisposition(); err == nil && disp != "" {
				if name := strings.Trim(params["filename"], `"'`); name != "" {
					return name
				}
			}
		}
	}
	return ""
}
