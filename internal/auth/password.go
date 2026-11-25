// Package auth implements password hashing, TOTP, sessions and login
// throttling using only the Go standard library.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PBKDF2-HMAC-SHA512 with 210k iterations (OWASP 2023+). On the EX4's
// 2 GHz Kirkwood this costs well under a second per login.
const defaultIterations = 210_000

var ErrWeakPassword = errors.New("password must be at least 10 characters")

func ValidatePassword(pw string) error {
	if utf8.RuneCountInString(pw) < 10 {
		return ErrWeakPassword
	}
	return nil
}

func HashPassword(pw string) (string, error) {
	return hashWith(pw, defaultIterations)
}

func hashWith(pw string, iter int) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha512.New, pw, salt, iter, 32)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha512$%d$%s$%s", iter, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func VerifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha512" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1000 {
		return false
	}
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(parts[2])
	want, err2 := enc.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha512.New, pw, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash lets login spend the same time whether or not the user exists.
var dummyHash, _ = HashPassword("timing-equaliser-not-a-password")

func VerifyOrDummy(pw string, encoded string) bool {
	if encoded == "" {
		VerifyPassword(pw, dummyHash)
		return false
	}
	return VerifyPassword(pw, encoded)
}

// RandomToken returns n random bytes, URL-safe base64 encoded.
func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// SetupCode is a short human-typable one-time code shown on the OLED and in
// the log, so that only someone with physical/console access can claim a
// freshly installed NAS.
func SetupCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	out := make([]byte, 0, 9)
	for i, v := range b {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, alphabet[int(v)%len(alphabet)])
	}
	return string(out)
}
