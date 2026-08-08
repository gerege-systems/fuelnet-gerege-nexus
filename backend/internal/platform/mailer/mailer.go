package mailer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strings"
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

// MessageMailer sends a message this package did not word.
//
// It is a second interface rather than a method added to OTPMailer because a
// one-time code is not the only mail the platform sends any more — email
// verification links go out through the same queue — and widening OTPMailer
// would oblige every implementation, including test doubles, to grow a method
// most of them have no use for.
type MessageMailer interface {
	SendMessage(ctx context.Context, msg EmailMessage) error
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
	if m.password == "" || m.smtpHost == "" {
		slog.Info("MOCK_EMAIL_SENT", "to", toEmail, "otp_code", code)
		return nil
	}
	return m.SendMessage(ctx, EmailMessage{
		To:      toEmail,
		Subject: "Your Security Verification OTP",
		Body:    fmt.Sprintf("Your one-time verification code is: %s. It will expire in 10 minutes.", code),
	})
}

// SendMessage delivers a composed message.
//
// A header is only ever written from fields this process built, and a recipient
// address with a newline in it would let the caller append headers of its own —
// a second Bcc, a different From. Callers validate the address, and this is the
// backstop that means a mistake there is a refused mail rather than a forged
// one.
func (m *SyncOTPMailer) SendMessage(ctx context.Context, msg EmailMessage) error {
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return errors.New("mailer: recipient and subject must not contain line breaks")
	}

	if m.password == "" || m.smtpHost == "" {
		slog.Info("MOCK_EMAIL_SENT", "to", msg.To, "subject", msg.Subject, "body", msg.Body)
		return nil
	}

	raw := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, msg.To, mime.QEncoding.Encode("UTF-8", msg.Subject), msg.Body)

	auth := smtp.PlainAuth("", m.from, m.password, m.smtpHost)
	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)
	return smtp.SendMail(addr, auth, m.from, []string{msg.To}, []byte(raw))
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

	// Message, when set, is a composed mail rather than a one-time code, and
	// the worker delivers it through MessageMailer. The queue is shared on
	// purpose: both kinds of mail leave through the same SMTP conversation, so
	// they should share one set of workers and one retry budget.
	Message *EmailMessage
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

	err := m.deliver(ctx, task)
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

// deliver picks the delivery path a task asked for. A composed message needs a
// sender that can send one; an OTP-only implementation is refused loudly rather
// than silently delivering nothing.
func (m *AsyncOTPMailer) deliver(ctx context.Context, task EmailTask) error {
	if task.Message == nil {
		return m.syncMailer.SendOTP(ctx, task.ToEmail, task.Code)
	}
	sender, ok := m.syncMailer.(MessageMailer)
	if !ok {
		return errors.New("mailer: the configured sender cannot deliver composed messages")
	}
	return sender.SendMessage(ctx, *task.Message)
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

// EnqueueMessage accepts a composed mail for delivery. The false return is not
// decoration: a caller that has just recorded something the mail is supposed to
// announce — an email verification link, say — has to be able to undo that
// record rather than leave a promise nobody kept.
func (m *AsyncOTPMailer) EnqueueMessage(msg EmailMessage) bool {
	if !m.enqueue(EmailTask{ToEmail: msg.To, Message: &msg, Retries: 0}) {
		slog.Error("async mailer queue is full or shut down, dropped message", "to", msg.To)
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
