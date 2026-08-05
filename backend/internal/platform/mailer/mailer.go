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
	quit       chan struct{}
}

type EmailTask struct {
	ToEmail string
	Code    string
	Retries int
}

func NewAsyncOTPMailer(syncMailer OTPMailer, workers, queueSize, retries int) *AsyncOTPMailer {
	m := &AsyncOTPMailer{
		syncMailer: syncMailer,
		queue:      make(chan EmailTask, queueSize),
		workers:    workers,
		retries:    retries,
		quit:       make(chan struct{}),
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

func (m *AsyncOTPMailer) worker(id int) {
	defer m.wg.Done()
	for {
		select {
		case task, ok := <-m.queue:
			if !ok {
				return
			}
			m.processTask(task)
		case <-m.quit:
			return
		}
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
			select {
			case m.queue <- task:
			default:
				slog.Error("mailer queue full during retry", "to", task.ToEmail)
			}
		} else {
			slog.Error("mailer task failed after retries", "to", task.ToEmail, "error", err)
		}
	}
}

func (m *AsyncOTPMailer) EnqueueOTP(toEmail, code string) bool {
	select {
	case m.queue <- EmailTask{ToEmail: toEmail, Code: code, Retries: 0}:
		return true
	default:
		slog.Error("async mailer queue is full, dropped message", "to", toEmail)
		return false
	}
}

func (m *AsyncOTPMailer) Shutdown(ctx context.Context) error {
	close(m.quit)
	close(m.queue)
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
