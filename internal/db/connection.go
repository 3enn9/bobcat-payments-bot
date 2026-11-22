package db

import (
	"PaymentsBot/internal/config"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/url"
	"strings"
	"time"
)

func NewConnectionDB(c *config.Config) (*Database, error) {
	locationName := strings.TrimSpace(c.Timezone)
	if locationName == "" {
		locationName = "Europe/Samara"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return nil, fmt.Errorf("load database timezone %q: %w", locationName, err)
	}
	_, offset := time.Now().In(location).Zone()
	zoneSign := "+"
	if offset < 0 {
		zoneSign = "-"
		offset = -offset
	}
	serverTimeZone := fmt.Sprintf("'%s%02d:%02d'", zoneSign, offset/3600, offset%3600/60)

	query := url.Values{}
	query.Set("parseTime", "true")
	query.Set("timeout", "3s")
	query.Set("loc", locationName)
	query.Set("time_zone", serverTimeZone)
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", c.Root, c.Password, c.Host, c.Port, c.Dbname, query.Encode())
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
