package jobqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Job struct {
	IdempotencyKey string `json:"idempotency_key"`
	AppointmentID  string `json:"appointment_id"`
	DoctorID       string `json:"doctor_id"`
	OccurredAt     string `json:"occurred_at"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
	Attempt        int    `json:"attempt"`
}

type JobLog struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	JobID   string `json:"job_id"`
	Attempt int    `json:"attempt"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type JobQueue interface {
	Enqueue(job Job)
	Stop()
}

type workerPool struct {
	rdb         *redis.Client
	jobCh       chan Job
	gatewayURL  string
	workerCount int
	wg          sync.WaitGroup
}

func NewJobQueue(rdb *redis.Client, gatewayURL string, poolSize int) JobQueue {
	if poolSize <= 0 {
		poolSize = 3
	}
	wp := &workerPool{
		rdb:         rdb,
		jobCh:       make(chan Job, 100),
		gatewayURL:  gatewayURL,
		workerCount: poolSize,
	}
	wp.start()
	return wp
}

func (wp *workerPool) start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *workerPool) Enqueue(job Job) {
	// Initial enqueue doesn't count as an attempt yet in terms of processing
	// but we log it as enqueued with attempt 0 or 1?
	// Section 6.4: attempt integer (1-based)
	wp.logJob(job, "enqueued", "info", "")
	wp.jobCh <- job
}

func (wp *workerPool) Stop() {
	close(wp.jobCh)
	wp.wg.Wait()
}

func (wp *workerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobCh {
		wp.processJob(job)
	}
}

func (wp *workerPool) processJob(job Job) {
	job.Attempt++
	wp.logJob(job, "processing", "info", "")

	// Check idempotency in Redis
	if wp.rdb != nil {
		val, err := wp.rdb.Get(context.Background(), "job:"+job.IdempotencyKey).Result()
		if err == nil && val == "done" {
			// Section 9: Log at info level and drop the job silently.
			// Re-logging at info level as a "duplicate" drop
			wp.logJob(job, "dropped_duplicate", "info", "duplicate idempotency key")
			return
		}
	}

	// Call gateway
	err := wp.callGateway(job)
	if err == nil {
		if wp.rdb != nil {
			wp.rdb.Set(context.Background(), "job:"+job.IdempotencyKey, "done", 24*time.Hour)
		}
		wp.logJob(job, "success", "info", "")
		return
	}

	// Retry logic
	if job.Attempt < 3 {
		wp.logJob(job, "retry", "warn", err.Error())
		backoff := time.Duration(1<<uint(job.Attempt-1)) * time.Second
		time.Sleep(backoff)
		wp.jobCh <- job
		return
	}

	// Dead letter
	wp.logJob(job, "dead_letter", "error", err.Error())
}

func (wp *workerPool) callGateway(job Job) error {
	payload, _ := json.Marshal(map[string]string{
		"idempotency_key": job.IdempotencyKey,
		"channel":         job.Channel,
		"recipient":       job.Recipient,
		"message":         job.Message,
	})

	resp, err := http.Post(wp.gatewayURL+"/notify", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("gateway returned %d", resp.StatusCode)
}

func (wp *workerPool) logJob(job Job, status, level, errMsg string) {
	entry := JobLog{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		JobID:   job.IdempotencyKey,
		Attempt: job.Attempt,
		Status:  status,
		Error:   errMsg,
	}
	data, _ := json.Marshal(entry)
	if status == "dead_letter" {
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Println(string(data))
	}
}

func GenerateIdempotencyKey(eventType, id, occurredAt string) string {
	h := sha256.New()
	h.Write([]byte(eventType + id + occurredAt))
	return hex.EncodeToString(h.Sum(nil))
}
