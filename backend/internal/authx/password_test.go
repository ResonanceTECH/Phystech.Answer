package authx

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_Argon2idPHC(t *testing.T) {
	h, err := HashPassword("super-secret-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=32768,t=3,p=2$") {
		t.Fatalf("unexpected hash format: %s", h)
	}
	// PHC-строка не должна содержать сам пароль.
	if strings.Contains(h, "super-secret") {
		t.Fatal("hash leaks password")
	}
}

func TestVerifyPassword_Argon2id(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !VerifyPassword(h, "correct horse battery") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(h, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPassword_BcryptCompat(t *testing.T) {
	b, err := bcrypt.GenerateFromPassword([]byte("legacy-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if !VerifyPassword(string(b), "legacy-pass") {
		t.Fatal("expected bcrypt hash to verify")
	}
	if VerifyPassword(string(b), "nope") {
		t.Fatal("expected wrong bcrypt password to fail")
	}
}

func TestVerifyPassword_UnknownFormat(t *testing.T) {
	if VerifyPassword("md5:whatever", "x") {
		t.Fatal("unknown format must not verify")
	}
	if VerifyPassword("", "x") {
		t.Fatal("empty hash must not verify")
	}
}

func TestNewToken_Format(t *testing.T) {
	tok, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	// 32 байта → 43 символа base64url без паддинга.
	if len(tok) != 43 {
		t.Fatalf("token len = %d, want 43", len(tok))
	}
	if len(hash) != 64 {
		t.Fatalf("hash len = %d, want 64 hex", len(hash))
	}
	if HashToken(tok) != hash {
		t.Fatal("HashToken(token) != returned hash")
	}
	// Токен не должен содержать небезопасных для URL/cookie символов.
	if strings.ContainsAny(tok, "+/=") {
		t.Fatalf("token contains unsafe chars: %q", tok)
	}
}

func TestNewToken_Unique(t *testing.T) {
	a, ha, _ := NewToken()
	b, hb, _ := NewToken()
	if a == b || ha == hb {
		t.Fatal("tokens must be unique")
	}
}
