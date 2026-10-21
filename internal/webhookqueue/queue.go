package webhookqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetryDelay = time.Second
	maxRetryDelay     = time.Minute
	doneRetention     = 90 * 24 * time.Hour
)

type Event struct {
	Source     string          `json:"source"`
	ExternalID string          `json:"external_id"`
	Payload    json.RawMessage `json:"payload"`
	ReceivedAt time.Time       `json:"received_at"`
}

type Queue struct {
	pendingDir string
	doneDir    string
	mu         sync.Mutex
}

func New(dir string) (*Queue, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("webhook queue directory is empty")
	}
	q := &Queue{
		pendingDir: filepath.Join(dir, "pending"),
		doneDir:    filepath.Join(dir, "done"),
	}
	for _, path := range []string{q.pendingDir, q.doneDir} {
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, fmt.Errorf("create webhook queue directory: %w", err)
		}
	}
	if err := q.removeStaleDone(); err != nil {
		return nil, err
	}
	return q, nil
}

// Enqueue returns only after the event file and its directory entry are synced.
func (q *Queue) Enqueue(source, externalID string, payload []byte) error {
	externalID = strings.TrimSpace(externalID)
	if source == "" || externalID == "" || len(externalID) > 128 {
		return fmt.Errorf("webhook source and payment ID are required")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("webhook payload is not valid JSON")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	name := eventFilename(source, externalID)
	pendingPath := filepath.Join(q.pendingDir, name)
	donePath := filepath.Join(q.doneDir, name)
	if exists(donePath) || exists(pendingPath) {
		return nil
	}

	event := Event{
		Source:     source,
		ExternalID: externalID,
		Payload:    append(json.RawMessage(nil), payload...),
		ReceivedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode webhook event: %w", err)
	}
	tmp, err := os.CreateTemp(q.pendingDir, ".incoming-*.tmp")
	if err != nil {
		return fmt.Errorf("create webhook event: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure webhook event: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write webhook event: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync webhook event: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close webhook event: %w", err)
	}
	if err := os.Rename(tmpPath, pendingPath); err != nil {
		if exists(pendingPath) || exists(donePath) {
			return nil
		}
		return fmt.Errorf("publish webhook event: %w", err)
	}
	if err := syncDir(q.pendingDir); err != nil {
		return fmt.Errorf("sync pending webhook directory: %w", err)
	}
	return nil
}

// Run processes pending events sequentially. Failed events remain on disk and
// are retried with capped backoff; a crash after DB commit is safe because the
// payment insert is idempotent by (source, external_id).
func (q *Queue) Run(ctx context.Context, process func(Event) error) {
	if process == nil {
		log.Printf("webhook queue stopped: processor is nil")
		return
	}
	if err := q.removeStaleTemps(); err != nil {
		log.Printf("webhook queue cleanup: %v", err)
	}
	ticker := time.NewTicker(defaultRetryDelay)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	attempts := make(map[string]int)
	nextAttempt := make(map[string]time.Time)
	for {
		q.processPending(ctx, process, attempts, nextAttempt)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-cleanupTicker.C:
			if err := q.removeStaleDone(); err != nil {
				log.Printf("webhook queue cleanup: %v", err)
			}
			if err := q.removeStaleTemps(); err != nil {
				log.Printf("webhook queue cleanup: %v", err)
			}
		}
	}
}

func (q *Queue) processPending(ctx context.Context, process func(Event) error, attempts map[string]int, nextAttempt map[string]time.Time) {
	entries, err := os.ReadDir(q.pendingDir)
	if err != nil {
		log.Printf("read pending webhook queue: %v", err)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		if time.Now().Before(nextAttempt[name]) {
			continue
		}

		path := filepath.Join(q.pendingDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			q.recordFailure(name, attempts, nextAttempt, err, nil)
			continue
		}
		var event Event
		if err := json.Unmarshal(data, &event); err != nil || event.Source == "" || event.ExternalID == "" || !json.Valid(event.Payload) {
			if err == nil {
				err = errors.New("invalid queued webhook event")
			}
			q.recordFailure(name, attempts, nextAttempt, err, nil)
			continue
		}
		if err := process(event); err != nil {
			q.recordFailure(name, attempts, nextAttempt, err, &event)
			continue
		}
		if err := q.markDone(name); err != nil {
			q.recordFailure(name, attempts, nextAttempt, err, &event)
			continue
		}
		delete(attempts, name)
		delete(nextAttempt, name)
		log.Printf("incoming payment event processed source=%s external_id=%q", event.Source, event.ExternalID)
	}
}

func (q *Queue) recordFailure(name string, attempts map[string]int, nextAttempt map[string]time.Time, err error, event *Event) {
	attempts[name]++
	delay := defaultRetryDelay * time.Duration(1<<min(attempts[name]-1, 6))
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	nextAttempt[name] = time.Now().Add(delay)
	if event == nil {
		log.Printf("incoming payment queue retry file=%s attempt=%d next_in=%s: %v", name, attempts[name], delay, err)
		return
	}
	log.Printf("incoming payment queue retry source=%s external_id=%q attempt=%d next_in=%s: %v", event.Source, event.ExternalID, attempts[name], delay, err)
}

func (q *Queue) markDone(name string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := os.Rename(filepath.Join(q.pendingDir, name), filepath.Join(q.doneDir, name)); err != nil {
		return fmt.Errorf("archive processed webhook event: %w", err)
	}
	if err := syncDir(q.pendingDir); err != nil {
		return err
	}
	return syncDir(q.doneDir)
}

func (q *Queue) removeStaleTemps() error {
	entries, err := os.ReadDir(q.pendingDir)
	if err != nil {
		return fmt.Errorf("scan webhook queue temp files: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".incoming-") && strings.HasSuffix(entry.Name(), ".tmp") {
			if err := os.Remove(filepath.Join(q.pendingDir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove stale webhook temp: %w", err)
			}
		}
	}
	return nil
}

func (q *Queue) removeStaleDone() error {
	entries, err := os.ReadDir(q.doneDir)
	if err != nil {
		return fmt.Errorf("scan completed webhook queue: %w", err)
	}
	cutoff := time.Now().Add(-doneRetention)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat completed webhook event: %w", err)
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(q.doneDir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove old completed webhook event: %w", err)
			}
		}
	}
	return nil
}

func eventFilename(source, externalID string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + externalID))
	return hex.EncodeToString(sum[:]) + ".json"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
