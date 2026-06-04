package db

import (
	"PaymentsBot/internal/config"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

func NewConnectionDB(c *config.Config) (*Database, error) {

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", c.Root, c.Password, c.Host, c.Port, c.Dbname)
	var err error
	var db *sql.DB

	for i := 0; i < 20; i++ {
		db, err = sql.Open("mysql", dataSourceName)
		log.Printf("connection: %s", dataSourceName)
		if err != nil {
			log.Printf("❌ Failed to open DB connection (try %d/20): %v", i+1, err)
		} else if pingErr := db.Ping(); pingErr == nil {
			log.Println("✅ Connected to DB")
			return &Database{DB: db}, nil
		} else {
			log.Printf("⚠️ Waiting for DB to be ready (try %d/20)...", i+1)
			db.Close() // важно закрыть неудачное соединение
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to DB after 10 attempts: %w", err)
}
