package mailer_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/mailer"
)

type mockSyncMailer struct {
	mu    sync.Mutex
	calls []string
}

func (m *mockSyncMailer) SendOTP(ctx context.Context, toEmail, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, toEmail+":"+code)
	return nil
}

func TestAsyncOTPMailer(t *testing.T) {
	mockMailer := &mockSyncMailer{}
	asyncMailer := mailer.NewAsyncOTPMailer(mockMailer, 2, 10, 2)

	ok := asyncMailer.EnqueueOTP("user@example.com", "123456")
	if !ok {
		t.Fatal("expected enqueue to succeed")
	}

	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := asyncMailer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("unexpected error during shutdown: %v", err)
	}

	mockMailer.mu.Lock()
	defer mockMailer.mu.Unlock()
	if len(mockMailer.calls) != 1 {
		t.Fatalf("expected 1 mail call, got %d", len(mockMailer.calls))
	}
	if mockMailer.calls[0] != "user@example.com:123456" {
		t.Errorf("unexpected call format: %s", mockMailer.calls[0])
	}
}
