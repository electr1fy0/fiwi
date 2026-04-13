package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := LoginWithCtx(ctx, http.DefaultClient, URL, userID, password)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("timeout exceeded")
		} else {
			log.Fatal(err)
		}
	}
	fmt.Print(resp)
}
