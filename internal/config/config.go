package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	Port     string
	Root     string
	Password string
	Dbname   string
	Host     string
	Token    string
	MaxToken string
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("⚠️  .env файл не найден, переменные окружения не загружены: %v", err)
	}

	return &Config{
		Port:     os.Getenv("DB_PORT"),
		Root:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASS"),
		Dbname:   os.Getenv("DB_NAME"),
		Host:     os.Getenv("DB_HOST"),
		Token:    os.Getenv("BOT_TOKEN"),
		MaxToken: os.Getenv("MAX_TOKEN"),
		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: os.Getenv("SMTP_PORT"),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
		SMTPFrom: os.Getenv("SMTP_FROM"),
	}
}
