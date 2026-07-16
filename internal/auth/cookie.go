package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	ErrValueTooLong = errors.New("cookie value too long")
	ErrInvalidValue = errors.New("invalid cookie value")
)

func (s *AuthService) IsSessionValid(ctx context.Context, r *http.Request) bool {
	secret, err := hex.DecodeString(s.cfg.CookieSecret)
	if err != nil {
		return false
	}

	gobEncodedValue, err := readEncrypted(r, sessionCookie, secret)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			slog.ErrorContext(ctx, "failed to read cookie", "error", err)
		}
		return false
	}

	var session Session
	reader := strings.NewReader(gobEncodedValue)

	if err := gob.NewDecoder(reader).Decode(&session); err != nil {
		slog.ErrorContext(ctx, "failed to decode cookie value", "error", err)
		return false
	}

	if session.Version != s.cfg.TokenVersion || time.Now().Unix() >= session.Exp {
		return false
	}

	return true
}

func writeEncrypted(w http.ResponseWriter, cookie http.Cookie, secretKey string) error {
	secret, err := hex.DecodeString(secretKey)
	if err != nil {
		return err
	}
	if len(secret) != 32 {
		return fmt.Errorf("cookie secret must decode to 32 bytes, got %d", len(secretKey))
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return err
	}

	plaintext := fmt.Sprintf("%s:%s", cookie.Name, cookie.Value)

	encryptedValue := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	cookie.Value = string(encryptedValue)

	return write(w, cookie)
}

func readEncrypted(r *http.Request, name string, secretKey []byte) (string, error) {
	encryptedValue, err := read(r, name)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()

	if len(encryptedValue) < nonceSize {
		return "", ErrInvalidValue
	}

	nonce := encryptedValue[:nonceSize]
	ciphertext := encryptedValue[nonceSize:]

	plaintext, err := aesGCM.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return "", ErrInvalidValue
	}

	expectedName, value, ok := strings.Cut(string(plaintext), ":")
	if !ok {
		return "", ErrInvalidValue
	}

	if expectedName != name {
		return "", ErrInvalidValue
	}

	return value, nil
}

func write(w http.ResponseWriter, cookie http.Cookie) error {
	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

	if len(cookie.String()) > 4096 {
		return ErrValueTooLong
	}

	http.SetCookie(w, &cookie)

	return nil
}

func read(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidValue
	}

	return string(value), nil
}
