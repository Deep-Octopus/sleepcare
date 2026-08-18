package logger

import (
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer secret")
	h.Set("X-Token", "tok")
	h.Set("Accept", "application/json")
	got := SanitizeHeaders(h)
	if got["Authorization"] != maskValue || got["X-Token"] != maskValue {
		t.Fatalf("sensitive headers not masked: %+v", got)
	}
	if got["Accept"] != "application/json" {
		t.Fatalf("normal header changed: %+v", got)
	}
}

func TestTruncate(t *testing.T) {
	if Truncate("short", 100) != "short" {
		t.Fatal("short should be unchanged")
	}
	long := strings.Repeat("a", 2000)
	if Truncate(long, 1024) != truncatedMark {
		t.Fatal("long should be replaced with mark")
	}
}

func TestSanitizeBodyMasksClientGrantAndAnswers(t *testing.T) {
	body := `{"grant":"one-time-value","answers":{"DEMO_CHOICE":"A"},"nested":{"token":"session-value"}}`
	masked := SanitizeBody("application/json", body)
	for _, secret := range []string{"one-time-value", "DEMO_CHOICE", "session-value"} {
		if strings.Contains(masked, secret) {
			t.Fatalf("sensitive client value remained in log body: %s", masked)
		}
	}
	if strings.Count(masked, maskValue) != 3 {
		t.Fatalf("unexpected masked body: %s", masked)
	}
}
