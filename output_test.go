package main

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	result := Success("ok")
	if !strings.HasPrefix(result, "\033[32m") {
		t.Errorf("expected green prefix, got %q", result)
	}
	if !strings.HasSuffix(result, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", result)
	}
	if !strings.Contains(result, "ok") {
		t.Errorf("expected content, got %q", result)
	}
}

func TestError(t *testing.T) {
	result := Error("fail")
	if !strings.HasPrefix(result, "\033[31m") {
		t.Errorf("expected red prefix, got %q", result)
	}
	if !strings.HasSuffix(result, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", result)
	}
}

func TestWarning(t *testing.T) {
	result := Warning("warn")
	if !strings.HasPrefix(result, "\033[33m") {
		t.Errorf("expected yellow prefix, got %q", result)
	}
	if !strings.HasSuffix(result, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", result)
	}
}

func TestInfo(t *testing.T) {
	result := Info("info")
	if !strings.HasPrefix(result, "\033[34m") {
		t.Errorf("expected blue prefix, got %q", result)
	}
	if !strings.HasSuffix(result, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", result)
	}
}

func TestBold(t *testing.T) {
	result := Bold("bold")
	if !strings.HasPrefix(result, "\033[1m") {
		t.Errorf("expected bold prefix, got %q", result)
	}
	if !strings.HasSuffix(result, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", result)
	}
}

func TestSuccessContainsContent(t *testing.T) {
	result := Success("hello world")
	if result != "\033[32mhello world\033[0m" {
		t.Errorf("unexpected format: %q", result)
	}
}

func TestLoginResultAccessGranted(t *testing.T) {
	PrintLoginResult("Access Granted")
}

func TestLoginResultAlreadyLoggedIn(t *testing.T) {
	PrintLoginResult("Already logged in")
}

func TestLoginResultInvalidCredentials(t *testing.T) {
	PrintLoginResult("Invalid credentials")
}

func TestLoginResultUnknown(t *testing.T) {
	PrintLoginResult("some random response")
}
