package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected post, got %s", r.Method)
		}

		r.ParseForm()
		if r.Form.Get("userId") != "testuser" {
			t.Errorf("wrong userid")
		}
		if r.Form.Get("password") != "testpass" {
			t.Errorf("wrong pass")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("login successful"))

	}))

	defer server.Close()
	client := server.Client()
	resp, err := LoginWithCtx(context.Background(), client, server.URL, "testuser", "testpass")
	if err != nil {
		t.Fatalf("unexpcted error: %v", err)
	}

	if resp != "login successful" {
		t.Errorf("unexpcted response: %s", resp)
	}
}

func TestLogin_NetworkError(t *testing.T) {
	client := &http.Client{}

	_, err := LoginWithCtx(context.Background(), client, "http://invalid-url", "u", "p")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestLogin_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := server.Client()

	_, err := LoginWithCtx(context.Background(), client, server.URL, "u", "p")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogin_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
	}))
	defer server.Close()

	client := server.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	_, err := LoginWithCtx(ctx, client, server.URL, "u", "p")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected timeout, got %s", err)
	}
}

func Test_FilterHTML(t *testing.T) {

	t.Run("success html", func(t *testing.T) {
		htmlSuccess := `<!doctype html> toAccess Grantedto`

		got := FilterHTML(htmlSuccess)
		if got != "Access Granted" {
			t.Errorf("expected Access Granted, got %s", got)
		}
	})

	t.Run("success html", func(t *testing.T) {
		htmlFailure := `<!doctype html> account does not exist`

		got := FilterHTML(htmlFailure)
		if got != "Invalid credentials" {
			t.Errorf("expected Access Granted, got %s", got)
		}
	})

	t.Run("success html", func(t *testing.T) {
		notHTML := `<doctype> something not html idk`

		got := FilterHTML(notHTML)
		if got != notHTML {
			t.Errorf("expected %s, got %s", notHTML, got)
		}
	})

}
