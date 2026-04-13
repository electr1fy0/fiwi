package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var (
	URL      = "http://phc.prontonetworks.com/cgi-bin/authlogin?URI=http://detectportal.firefox.com/canonical.html"
	userID   = strings.TrimSpace(os.Getenv("WIFI_USERID"))
	password = strings.TrimSpace(os.Getenv("WIFI_PASSWORD"))
)

var internalServerErr = errors.New("internal server error")

func LoginWithCtx(ctx context.Context, client *http.Client, portalURL, userID, password string) (string, error) {
	credentials := &url.Values{}
	credentials.Add("userId", userID)
	credentials.Add("password", password)
	credentials.Add("serviceName", "ProntoAuthentication")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, portalURL, strings.NewReader(credentials.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 500 {
		return "", internalServerErr
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ResolveCredentials(envUser, envPass string, fileData []byte) (string, string) {
	if envUser != "" && envPass != "" {
		return strings.TrimSpace(envUser), strings.TrimSpace(envPass)
	}

	var creds struct {
		UserID   string `json:"userID"`
		Password string `json:"password"`
	}
	_ = json.Unmarshal(fileData, &creds)

	return strings.TrimSpace(creds.UserID), strings.TrimSpace(creds.Password)
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

func setEnv() error {
	var err error
	if userID != "" && password != "" {
		return nil
	}
	home, _ := os.UserHomeDir()
	fiwiPath := filepath.Join(home, ".fiwi")
	data, err := os.ReadFile(fiwiPath)
	var creds struct {
		UserID   string `json:"userID"`
		Password string `json:"password"`
	}
	json.Unmarshal(data, &creds)
	userID = strings.TrimSpace(creds.UserID)
	password = strings.TrimSpace(creds.Password)

	if userID != "" && password != "" {
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("welcome to fiwi. enter your credentials once to save them locally")
	fmt.Print("Enter Wi-Fi username: ")
	userID, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read userID ")
		os.Exit(1)
	}
	fmt.Print("Enter Wi-Fi password: ")
	password, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read userID ")
		os.Exit(1)
	}
	credentials, err := json.Marshal(map[string]string{
		"userID":   userID,
		"password": password,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(fiwiPath, credentials, 0644)
	return err
}

func main() {
	setEnv()

	cfg := RetryConfig{
		5,
		100 * time.Millisecond,
		10 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
