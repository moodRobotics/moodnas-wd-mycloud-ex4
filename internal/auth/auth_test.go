package auth

import (
	"encoding/base32"
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	h, err := hashWith("correct horse battery", 2000)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("correct horse battery", h) {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword("wrong", h) {
		t.Fatal("wrong password accepted")
	}
	if VerifyPassword("x", "garbage") {
		t.Fatal("garbage hash accepted")
	}
}

// RFC 6238 test vector (SHA1, T=59s -> 94287082, last 6 digits 287082).
func TestTOTPVector(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	if !VerifyTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("RFC 6238 vector failed")
	}
	if VerifyTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("bad code accepted")
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter()
	now := time.Unix(1000, 0)
	for i := 0; i < 4; i++ {
		l.Fail("ip", now)
	}
	if ok, _ := l.Allowed("ip", now); !ok {
		t.Fatal("locked too early")
	}
	l.Fail("ip", now)
	if ok, wait := l.Allowed("ip", now); ok || wait != time.Minute {
		t.Fatalf("expected 1 min lock, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allowed("ip", now.Add(61*time.Second)); !ok {
		t.Fatal("lock did not expire")
	}
}

func TestSetupCode(t *testing.T) {
	c := SetupCode()
	if len(c) != 9 || c[4] != '-' {
		t.Fatalf("bad setup code %q", c)
	}
}
