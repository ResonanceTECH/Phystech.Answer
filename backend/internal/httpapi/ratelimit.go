package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter — простой in-memory token bucket на ключ (обычно IP).
// Подходит для одного инстанса backend; при горизонтальном масштабировании
// заменяется на общий стор (Redis).
type rateLimiter struct {
	mu      sync.Mutex
	rate    float64 // токенов в секунду
	burst   float64 // ёмкость ведра
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(requests int, per time.Duration) *rateLimiter {
	return &rateLimiter{
		rate:    float64(requests) / per.Seconds(),
		burst:   float64(requests),
		buckets: make(map[string]*bucket),
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: time.Now()}
		l.buckets[key] = b
	}
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// clientIP — адрес клиента для rate limiting. X-Forwarded-For не доверяем:
// backend работает за своим прокси/напрямую.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
