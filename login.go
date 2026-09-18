package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	URL               = "http://phc.prontonetworks.com/cgi-bin/authlogin?URI=http://detectportal.firefox.com/canonical.html"
	ConnectivityURL   = "http://connectivitycheck.gstatic.com/generate_204"
	internalServerErr = errors.New("internal server error")
	accessNotGranted  = errors.New("portal did not grant internet access")
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
	credentials.Add("Submit22", "Login")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, portalURL, strings.NewReader(credentials.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", internalServerErr
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func HasInternetAccess(ctx context.Context, client *http.Client, probeURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusNoContent
}

func LoginAndVerify(ctx context.Context, client *http.Client, portalURL, probeURL, userID, password string) (string, error) {
	result, err := LoginWithCtx(ctx, client, portalURL, userID, password)
	if err != nil {
		return "", err
	}

	filtered := FilterHTML(result)
	if filtered == "Invalid credentials" {
		return filtered, nil
	}
	if HasInternetAccess(ctx, client, probeURL) {
		return "Access Granted", nil
	}

	return "", accessNotGranted
}

func FilterHTML(s string) string {
	lowered := strings.ToLower(s)
	switch {
	case strings.Contains(lowered, "access granted"), strings.Contains(lowered, "already exists"):
		return "Access Granted"
	case strings.Contains(lowered, "http://detectportal.firefox.com/canonical.html"):
		return "Already logged in"
	case strings.Contains(lowered, "account does not exist"):
		return "Invalid credentials"
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

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
		backoff = min(cfg.MaxDelay, backoff*2)
	}

	return "", lastErr
}
