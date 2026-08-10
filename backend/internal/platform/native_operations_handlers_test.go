package platform

import (
	"encoding/base64"
	"testing"
)

func TestStaffPINFormat(t *testing.T) {
	for _, pin := range []string{"0000", "123456", "123456789012"} {
		if !validStaffPIN.MatchString(pin) {
			t.Fatalf("valid PIN rejected: %s", pin)
		}
	}
	for _, pin := range []string{"123", "1234567890123", "12ab"} {
		if validStaffPIN.MatchString(pin) {
			t.Fatalf("invalid PIN accepted: %s", pin)
		}
	}
}
func TestPushTokenEncryptionDoesNotPersistPlaintext(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	t.Setenv("PUSH_TOKEN_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
	got, err := encryptPushToken("fcm-secret-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if got == "fcm-secret-token-value" {
		t.Fatal("token was not encrypted")
	}
	if _, err = base64.StdEncoding.DecodeString(got); err != nil {
		t.Fatal("ciphertext is not transport-safe base64")
	}
}
func TestPushTokenEncryptionRequiresKey(t *testing.T) {
	t.Setenv("PUSH_TOKEN_ENCRYPTION_KEY", "")
	if _, err := encryptPushToken("token"); err == nil {
		t.Fatal("missing encryption key accepted")
	}
}
