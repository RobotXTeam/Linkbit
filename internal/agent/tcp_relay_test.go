package agent

import "testing"

func TestParseRelayTarget(t *testing.T) {
	host, port, err := ParseRelayTarget("friendlywrt:22")
	if err != nil {
		t.Fatalf("ParseRelayTarget() error = %v", err)
	}
	if host != "friendlywrt" || port != 22 {
		t.Fatalf("target = %s:%d, want friendlywrt:22", host, port)
	}

	if _, _, err := ParseRelayTarget("friendlywrt"); err == nil {
		t.Fatal("ParseRelayTarget() without port succeeded")
	}
	if _, _, err := ParseRelayTarget("friendlywrt:70000"); err == nil {
		t.Fatal("ParseRelayTarget() with invalid port succeeded")
	}
}
