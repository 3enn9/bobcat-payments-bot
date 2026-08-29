package clock

import (
	"log"
	"time"
)

var location *time.Location

func Init(tz string) {
	if tz == "" {
		location = time.Local
		log.Printf("clock: using system timezone (%s)", time.Now().Format("-07:00"))
		return
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("clock: timezone %q invalid, using system timezone: %v", tz, err)
		location = time.Local
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
