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
			t.Errorf("wrong userid, got %s", r.Form.Get("userId"))
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
	if err == nil {
		t.Errorf("expected error, got none")
	} else if !errors.Is(err, internalServerErr) {
		t.Errorf("expected internal server error, got %s", err)
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

	t.Run("already logged in html", func(t *testing.T) {
		alreadyHTML := `<html><head><meta http-equiv="refresh" content="0;url=http://detectportal.firefox.com/canonical.html"></head></html>%`

		got := FilterHTML(alreadyHTML)
		if got != "Already logged in" {
			t.Errorf("expected %s, got %s", "Already logged in", got)
		}
	})

}

func TestResolveCredentials(t *testing.T) {
	t.Run("env takes priority", func(t *testing.T) {
		u, p := ResolveCredentials("envUser", "envPass", []byte(`{"userID":"fileUser","password":"filePass"}`))

		if u != "envUser" || p != "envPass" {
			t.Fatalf("expected env creds, got %s %s", u, p)
		}
	})

	t.Run("fallback to file", func(t *testing.T) {
		u, p := ResolveCredentials("", "", []byte(`{"userID":"fileUser","password":"filePass"}`))

		if u != "fileUser" || p != "filePass" {
			t.Fatalf("expected file creds, got %s %s", u, p)
		}
	})

	t.Run("empty case", func(t *testing.T) {
		u, p := ResolveCredentials("", "", nil)

		if u != "" || p != "" {
			t.Fatalf("expected empty creds, got %s %s", u, p)
		}
	})
}

func TestRetry(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
	}

	t.Run("success on first attempt", func(t *testing.T) {
		calls := 0

		fn := func() (string, error) {
			calls++
			return "ok", nil
		}

		res, err := Retry(context.Background(), cfg, fn)

		if err != nil {
			t.Fatal(err)
		}
		if res != "ok" {
			t.Fatalf("expected ok, got %s", res)
		}
		if calls != 1 {
			t.Fatalf("expected 1 call, got %d", calls)
		}
	})

	t.Run("eventual success", func(t *testing.T) {
		calls := 0

		fn := func() (string, error) {
			calls++
			if calls < 3 {
				return "", errors.New("fail")
			}
			return "ok", nil
		}

		res, err := Retry(context.Background(), cfg, fn)

		if err != nil {
			t.Fatal(err)
		}
		if calls != 3 {
			t.Fatalf("expected 3 calls, got %d", calls)
		}
		if res != "ok" {
			t.Fatalf("unexpected result: %s", res)
		}
	})

	t.Run("all attempts fail", func(t *testing.T) {
		cfg.MaxAttempts = 3
		calls := 0

		fn := func() (string, error) {
			calls++
			return "", errors.New("fail")
		}

		_, err := Retry(context.Background(), cfg, fn)

		if err == nil {
			t.Fatal("expected error")
		}
		if calls != 3 {
			t.Fatalf("expected 3 attempts, got %d", calls)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0

		fn := func() (string, error) {
			calls++
			cancel()
			return "", errors.New("fail")
		}

		_, err := Retry(ctx, cfg, fn)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 call, got %d", calls)
		}
	})
}
