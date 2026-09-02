package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         string
	Root         string
	Password     string
	Dbname       string
	Host         string
	Token        string
	MaxToken     string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	IMAPHost     string
	IMAPPort     string
	IMAPUser     string
	IMAPPass     string
	IMAPFrom     string
	IMAPMailbox  string
	IMAPStartUID uint32
	Timezone     string
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("⚠️  .env файл не найден, переменные окружения не загружены: %v", err)
	}

	imapUser := firstNonEmpty(os.Getenv("IMAP_USER"), os.Getenv("SMTP_USER"))
	imapPass := firstNonEmpty(os.Getenv("IMAP_PASS"), os.Getenv("SMTP_PASS"))
	imapHost := os.Getenv("IMAP_HOST")
	if imapHost == "" && imapUser != "" {
		imapHost = "imap.yandex.ru"
	}
	imapFrom := os.Getenv("IMAP_FROM")
	if imapFrom == "" && imapUser != "" {
		imapFrom = "ratavina@mail.ru"
	}

	return &Config{
		Port:         os.Getenv("DB_PORT"),
		Root:         os.Getenv("DB_USER"),
		Password:     os.Getenv("DB_PASS"),
		Dbname:       os.Getenv("DB_NAME"),
		Host:         os.Getenv("DB_HOST"),
		Token:        os.Getenv("BOT_TOKEN"),
		MaxToken:     os.Getenv("MAX_TOKEN"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPass:     os.Getenv("SMTP_PASS"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		IMAPHost:     imapHost,
		IMAPPort:     firstNonEmpty(os.Getenv("IMAP_PORT"), "993"),
		IMAPUser:     imapUser,
		IMAPPass:     imapPass,
		IMAPFrom:     imapFrom,
		IMAPMailbox:  firstNonEmpty(os.Getenv("IMAP_MAILBOX"), "INBOX"),
		IMAPStartUID: parseUID(os.Getenv("IMAP_START_UID")),
		Timezone:     os.Getenv("APP_TIMEZONE"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseUID(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(n)
}
