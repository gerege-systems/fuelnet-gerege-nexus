package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"sync"
	"time"
)

type EmailMessage struct {
	To      string
	Subject string
	Body    string
}

type OTPMailer interface {
	SendOTP(ctx context.Context, toEmail, code string) error
}

type SyncOTPMailer struct {
	smtpHost string
	smtpPort string
	from     string
	password string
}

func NewSyncOTPMailer(host, port, from, password string) *SyncOTPMailer {
	return &SyncOTPMailer{
		smtpHost: host,
		smtpPort: port,
		from:     from,
		password: password,
	}
}

func (m *SyncOTPMailer) SendOTP(ctx context.Context, toEmail, code string) error {
	subject := "Your Security Verification OTP"
	body := fmt.Sprintf("Your one-time verification code is: %s. It will expire in 10 minutes.", code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, toEmail, subject, body)

	if m.password == "" || m.smtpHost == "" {
		slog.Info("MOCK_EMAIL_SENT", "to", toEmail, "otp_code", code)
		return nil
	}

	auth := smtp.PlainAuth("", m.from, m.password, m.smtpHost)
	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)
	return smtp.SendMail(addr, auth, m.from, []string{toEmail}, []byte(msg))
}

type AsyncOTPMailer struct {
	syncMailer OTPMailer
	queue      chan EmailTask
	workers    int
	retries    int
	wg         sync.WaitGroup

	// closed guards against enqueueing onto — or closing twice — a queue that
	// Shutdown has already closed. Both are runtime panics in Go.
	mu     sync.RWMutex
	closed bool
}

type EmailTask struct {
	ToEmail string
	Code    string
	Retries int
}

func NewAsyncOTPMailer(syncMailer OTPMailer, workers, queueSize, retries int) *AsyncOTPMailer {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}
	m := &AsyncOTPMailer{
		syncMailer: syncMailer,
		queue:      make(chan EmailTask, queueSize),
		workers:    workers,
		retries:    retries,
	}
	m.start()
	return m
}

func (m *AsyncOTPMailer) start() {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// worker drains the queue until it is closed. It deliberately has no second
// "quit" channel: selecting on one raced with the queue and let Shutdown
// discard already-accepted OTP mails.
func (m *AsyncOTPMailer) worker(id int) {
	defer m.wg.Done()
	for task := range m.queue {
		m.processTask(task)
	}
}

func (m *AsyncOTPMailer) processTask(task EmailTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.syncMailer.SendOTP(ctx, task.ToEmail, task.Code)
	if err != nil {
		if task.Retries < m.retries {
			task.Retries++
			slog.Warn("mailer task retry", "to", task.ToEmail, "attempt", task.Retries, "error", err)
			// Re-queueing after Shutdown closed the channel used to panic the
			// worker goroutine.
			if !m.enqueue(task) {
				slog.Error("mailer retry dropped (queue full or shut down)", "to", task.ToEmail)
			}
		} else {
			slog.Error("mailer task failed after retries", "to", task.ToEmail, "error", err)
		}
	}
}

// enqueue pushes a task unless the mailer has been shut down or the queue is
// saturated.
func (m *AsyncOTPMailer) enqueue(task EmailTask) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return false
	}
	select {
	case m.queue <- task:
		return true
	default:
		return false
	}
}

func (m *AsyncOTPMailer) EnqueueOTP(toEmail, code string) bool {
	if !m.enqueue(EmailTask{ToEmail: toEmail, Code: code, Retries: 0}) {
		slog.Error("async mailer queue is full or shut down, dropped message", "to", toEmail)
		return false
	}
	return true
}

// Shutdown stops accepting new mail, lets the workers drain what is already
// queued, and returns when they finish or ctx expires. Calling it twice is
// safe.
func (m *AsyncOTPMailer) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	close(m.queue)
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
