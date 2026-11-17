package scheduler

import (
	"PaymentsBot/internal/clock"
	"log"
	"time"
)

const retryInterval = time.Hour

func runTaskWithRetries(task func() error) {
	for {
		if err := task(); err == nil {
			return
		} else {
			log.Printf("daily scheduler task error: %v; retrying in %s", err, retryInterval)
		}
		time.Sleep(retryInterval)
	}
}

func SendDailyScheduler(task func() error) {
	go func() {
		now := clock.Now()

		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())

		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		delay := time.Until(nextRun)
		log.Printf("scheduler first run at: %s", nextRun)

		runTaskWithRetries(task)
		log.Printf("delay: %v", delay)
		time.Sleep(delay)

		runTaskWithRetries(task)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			runTaskWithRetries(task)
		}
	}()
}
