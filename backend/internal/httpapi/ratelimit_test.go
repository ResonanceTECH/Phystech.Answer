package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_BurstThenRefuse(t *testing.T) {
	l := newRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.allow("ip1") {
			t.Fatalf("request %d should be allowed (burst)", i+1)
		}
	}
	if l.allow("ip1") {
		t.Fatal("request beyond burst should be refused")
	}
	// Другой ключ — своё ведро.
	if !l.allow("ip2") {
		t.Fatal("other key should have own bucket")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	l := newRateLimiter(2, 20*time.Millisecond) // 100 req/s
	l.allow("k")
	l.allow("k")
	if l.allow("k") {
		t.Fatal("expected refusal at empty bucket")
	}
	time.Sleep(15 * time.Millisecond) // +~1.5 токена
	if !l.allow("k") {
		t.Fatal("expected token refill")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.7:51515"
	if got := clientIP(r); got != "203.0.113.7" {
		t.Fatalf("clientIP = %q", got)
	}
	r.RemoteAddr = "[2001:db8::1]:443"
	if got := clientIP(r); got != "2001:db8::1" {
		t.Fatalf("clientIP ipv6 = %q", got)
	}
}
