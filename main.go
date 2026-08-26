package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

var (
	userID   = strings.TrimSpace(os.Getenv("WIFI_USERID"))
	password = strings.TrimSpace(os.Getenv("WIFI_PASSWORD"))
)

func setEnv() error {
	if userID != "" && password != "" {
		return nil
	}

	path, err := CredentialsFilePath()
	if err != nil {
		return err
	}

	store, err := LoadCredentialStore(path)
	if err == nil {
		active := store.ActiveProfile()
		if active != nil && active.UserID != "" && active.Password != "" {
			userID = active.UserID
			password = active.Password
			return nil
		}
	}

	fmt.Println(Info("Welcome to fiwi! Enter your Wi-Fi credentials:"))

	var inputUserID, inputPassword string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&inputUserID).
				Prompt("> "),
			huh.NewInput().
				Title("Password").
				Value(&inputPassword).
				EchoMode(huh.EchoModePassword).
				Prompt("> "),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	userID = strings.TrimSpace(inputUserID)
	password = strings.TrimSpace(inputPassword)

	if userID == "" || password == "" {
		return fmt.Errorf("credentials are required")
	}

	store = &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: userID, Password: password},
		},
	}
	return SaveCredentialStore(path, store)
}

func main() {
	tuiMode := flag.Bool("tui", false, "open credential manager")
	flag.Parse()

	if *tuiMode {
		ShowCredentialManager()
		return
	}

	if err := setEnv(); err != nil {
		fmt.Println(Error(err.Error()))
		os.Exit(1)
	}

	cfg := RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   1500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	attempts := 0
	res, err := Retry(ctx, cfg, func() (string, error) {
		if attempts > 0 {
			fmt.Print("re")
		}
		fmt.Print("connecting...\r")
		attempts++
		return LoginWithCtx(ctx, http.DefaultClient, URL, userID, password)
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println(Error("Timeout exceeded"))
		} else {
			PrintError(err)
		}
		return
	}

	PrintLoginResult(FilterHTML(res))
}
