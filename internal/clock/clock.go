package clock

import (
	"log"
	"time"
)

var location *time.Location

func Init(tz string) {
	if tz == "" {
		tz = "Europe/Samara"
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("clock: timezone %q invalid, using UTC: %v", tz, err)
		location = time.UTC
		return
	}

	location = loc
	log.Printf("clock: using timezone %s", tz)
}

func Now() time.Time {
	if location == nil {
		return time.Now()
	}
	return time.Now().In(location)
}

func Today() string {
	return Now().Format("2006-01-02")
}

func Tomorrow() string {
	return Now().AddDate(0, 0, 1).Format("2006-01-02")
}
