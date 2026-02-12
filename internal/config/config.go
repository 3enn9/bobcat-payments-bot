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
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  .env файл не найден, переменные окружения не загружены")
	}

	return &Config{
		Port:     os.Getenv("DB_PORT"),
		Root:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASS"),
		Dbname:   os.Getenv("DB_NAME"),
		Host:     os.Getenv("DB_HOST"),
		Token:    os.Getenv("BOT_TOKEN"),
	}
}
