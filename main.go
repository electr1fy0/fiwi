package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	URL      = "http://phc.prontonetworks.com/cgi-bin/authlogin?URI=http://detectportal.firefox.com/canonical.html"
	userID   = os.Getenv("WIFI_USERID")
	password = os.Getenv("WIFI_PASSWORD")
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func LoginWithCtx(ctx context.Context, client *http.Client, portalURL, userID, password string) (string, error) {
	credentials := &url.Values{}
	credentials.Add("userId", userID)
	credentials.Add("password", password)
	credentials.Add("serviceName", "ProntoAuthentication")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, portalURL, strings.NewReader(credentials.Encode()))
	if err != nil {
		return "", err
	}
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("internal server error")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func FilterHTML(s string) string {
	lowered := strings.ToLower(s)
	if trimmed, ok := strings.CutPrefix(lowered, "<!doctype html>"); ok {
		switch {
		case strings.Contains(trimmed, "access granted"), strings.Contains(trimmed, "already exists"):
			return "Access Granted"
		case strings.Contains(trimmed, "account does not exist"):
			return "Invalid credentials"
		}
	}

	return s
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() (string, error)) (string, error) {
	backoff := cfg.BaseDelay

	var lastErr error
	for range cfg.MaxAttempts {
		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		time.Sleep(backoff)

		backoff = min(cfg.MaxDelay, backoff*2)
	}

	return "", lastErr
}

func main() {
	cfg := RetryConfig{
		5,
		100 * time.Millisecond,
		10 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := Retry(ctx, cfg, func() (string, error) {
		return LoginWithCtx(ctx, http.DefaultClient, URL, userID, password)
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Timeout exceeded")
		} else {
			fmt.Println(err)
		}
		return
	}

	fmt.Print(FilterHTML(res))
}
