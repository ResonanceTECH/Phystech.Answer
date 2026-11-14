// Package authx — примитивы авторизации: хеширование паролей Argon2id
// (с обратной совместимостью по bcrypt-хешам ранних пользователей) и
// генерация непрозрачных токенов сессий/сброса пароля.
package authx

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Параметры Argon2id: баланс стойкости и ресурсов сервера (OWASP cheat sheet).
const (
	argonTime    = 3
	argonMemory  = 32 * 1024 // 32 MiB
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword возвращает PHC-строку Argon2id: $argon2id$v=19$m=...,t=...,p=...$salt$key
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword проверяет пароль по сохранённому хешу. Поддерживаются
// Argon2id (PHC-строка) и устаревшие bcrypt-хеши сидированных пользователей.
func VerifyPassword(encodedHash, password string) bool {
	switch {
	case strings.HasPrefix(encodedHash, "$argon2id$"):
		return verifyArgon2id(encodedHash, password)
	case strings.HasPrefix(encodedHash, "$2"):
		return bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password)) == nil
	default:
		return false
	}
}

func verifyArgon2id(encoded, password string) bool {
	// PHC-строка: $argon2id$v=19$m=...,t=...,p=...$salt$key
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var (
		version               int
		memory, time, threads uint32
	)
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}
	saltB64, keyB64 := parts[4], parts[5]
	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(keyB64)
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, time, memory, uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

var errRand = errors.New("authx: rand.Read failed")

// NewToken генерирует 32 случайных байта и возвращает непрозрачный токен
// (base64url, безопасен для cookie и URL) и его SHA-256-хеш для хранения в БД.
func NewToken() (token, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", errRand
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashToken(token), nil
}

// HashToken — SHA-256 хранимого вида токена. Токены уже обладают высокой
// энтропией, поэтому достаточно быстрого криптографического хеша.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
