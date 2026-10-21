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

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=3s", c.Root, c.Password, c.Host, c.Port, c.Dbname)
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open DB connection: %w", err)
	}

	for i := 0; i < 5; i++ {
		if pingErr := db.Ping(); pingErr == nil {
			log.Println("✅ Connected to DB")
			return &Database{DB: db}, nil
		} else {
			log.Printf("⚠️ Waiting for DB to be ready (try %d/5): %v", i+1, pingErr)
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	log.Printf("⚠️ DB is still unavailable; starting the app so incoming webhooks can be queued")
	return &Database{DB: db}, nil
}
