package scheduler

import (
	"log"
	"time"
)

func SendDailyScheduler(task func() error) {
	go func() {
		now := time.Now()

		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())

		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		delay := time.Until(nextRun)
		log.Printf("scheduler first run at: %s", nextRun)

		err := task()
		if err != nil {
			log.Printf("daily scheduler task error: %v", err)
		}
		log.Printf("delay: %v", delay)
		time.Sleep(delay)

		err = task()
		if err != nil {
			log.Printf("daily scheduler task error: %v", err)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			err = task()
			if err != nil {
				log.Printf("daily scheduler task error: %v", err)
			}
		}
	}()
}
