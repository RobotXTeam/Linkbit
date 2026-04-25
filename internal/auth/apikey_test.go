package auth

import "testing"

func TestNewAPIKeyAndVerify(t *testing.T) {
	key, err := NewAPIKey()
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}
	if len(key) < 32 {
		t.Fatalf("NewAPIKey() returned unexpectedly short key: %q", key)
	}

	pepper := []byte("test-pepper")
	digest, err := HashAPIKey(key, pepper)
	if err != nil {
		t.Fatalf("HashAPIKey() error = %v", err)
	}

	if !VerifyAPIKey(key, digest, pepper) {
		t.Fatal("VerifyAPIKey() rejected the original key")
	}
	if VerifyAPIKey(key+"x", digest, pepper) {
		t.Fatal("VerifyAPIKey() accepted a modified key")
	}
}

func TestHashAPIKeyRequiresPepper(t *testing.T) {
	if _, err := HashAPIKey("lb_test", nil); err != ErrInvalidSecret {
		t.Fatalf("HashAPIKey() error = %v, want %v", err, ErrInvalidSecret)
	}
}
