package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type portalTestTransport struct {
	handler http.HandlerFunc
}

func (transport portalTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, req)
	response := recorder.Result()
	response.Request = req
	return response, nil
}

func portalTestClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{Transport: portalTestTransport{handler: handler}}
}

func TestLogin_Success(t *testing.T) {
	client := portalTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected post, got %s", r.Method)
		}

		if r.URL.Path != "/cgi-bin/authlogin" || r.URL.Query().Get("URI") != "http://example.com" {
			t.Errorf("unexpected login URL: %s", r.URL)
		}
		for name, want := range map[string]string{
			"Content-Type":              "application/x-www-form-urlencoded",
			"Accept":                    "*/*",
			"Accept-Language":           "en-GB,en;q=0.9",
			"Origin":                    "http://" + r.Host,
			"Referer":                   "http://" + r.Host + "/cgi-bin/authlogin?URI=http://115.248.50.60/",
			"Upgrade-Insecure-Requests": "1",
			"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36",
		} {
			if got := r.Header.Get(name); got != want {
				t.Errorf("header %s = %q, want %q", name, got, want)
			}
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		if r.Form.Get("userId") != "testuser" {
			t.Errorf("wrong userid, got %s", r.Form.Get("userId"))
		}
		if r.Form.Get("password") != "testpass" {
			t.Errorf("wrong pass")
		}
		if len(r.PostForm) != 3 || r.PostForm.Get("serviceName") != "ProntoAuthentication" {
			t.Errorf("expected only the three working curl form fields")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("login successful"))

	})

	portal, err := url.Parse(URL)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := LoginWithCtx(context.Background(), client, "http://portal.test"+portal.RequestURI(), "testuser", "testpass")
	if err != nil {
		t.Fatalf("unexpcted error: %v", err)
	}

	if resp != "login successful" {
		t.Errorf("unexpcted response: %s", resp)
	}
}

func TestLogin_DoesNotFollowRedirects(t *testing.T) {
	redirected := false
	client := portalTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/destination" {
			redirected = true
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Location", "/destination")
		w.WriteHeader(http.StatusFound)
		w.Write([]byte("portal response"))
	})

	result, err := LoginWithCtx(context.Background(), client, "http://portal.test/login", "user", "pass")
	if err != nil || result != "portal response" {
		t.Fatalf("expected original portal response, got %q, %v", result, err)
	}
	if redirected {
		t.Fatal("login followed the redirect")
	}
	if client.CheckRedirect != nil {
		t.Fatal("login changed the caller's redirect policy")
	}
}

func TestLogin_EncodesCredentials(t *testing.T) {
	const user = "user+name@example.com"
	const pass = "p&=+ %word"
	client := portalTestClient(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		if r.PostForm.Get("userId") != user || r.PostForm.Get("password") != pass {
			t.Error("form encoding did not preserve credentials")
		}
	})
	if _, err := LoginWithCtx(context.Background(), client, "http://portal.test/login", user, pass); err != nil {
		t.Fatal(err)
	}
}

func TestHasInternetAccess(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if !HasInternetAccess(context.Background(), server.Client(), server.URL) {
			t.Fatal("expected internet access")
		}
	})

	t.Run("captive portal", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("login form"))
		}))
		defer server.Close()

		if HasInternetAccess(context.Background(), server.Client(), server.URL) {
			t.Fatal("expected captive portal response")
		}
	})
}

func TestLoginAndVerify(t *testing.T) {
	online := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			r.ParseForm()
			if r.PostForm.Get("serviceName") == "ProntoAuthentication" && len(r.PostForm) == 3 {
				online = true
			}
			w.Write([]byte("Access Granted"))
		case "/probe":
			if online {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	result, err := LoginAndVerify(context.Background(), server.Client(), server.URL+"/login", server.URL+"/probe", "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Access Granted" {
		t.Fatalf("expected verified access, got %q", result)
	}
}

func TestLoginAndVerify_RequiresInternetAccess(t *testing.T) {
	client := portalTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Access Granted"))
	})

	_, err := LoginAndVerify(context.Background(), client, "http://portal.test/login", "http://portal.test/probe", "user", "pass")
	if !errors.Is(err, accessNotGranted) {
		t.Fatalf("expected failed connectivity verification, got %v", err)
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

func TestLogin_EmptyCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("userId") != "" || r.Form.Get("password") != "" {
			t.Errorf("expected empty credentials")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("login successful"))
	}))
	defer server.Close()

	client := server.Client()
	_, err := LoginWithCtx(context.Background(), client, server.URL, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	t.Run("invalid credentials html", func(t *testing.T) {
		htmlFailure := `<!doctype html> account does not exist`

		got := FilterHTML(htmlFailure)
		if got != "Invalid credentials" {
			t.Errorf("expected Invalid credentials, got %s", got)
		}
	})

	t.Run("unknown response", func(t *testing.T) {
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

	t.Run("case insensitive access granted", func(t *testing.T) {
		got := FilterHTML("ACCESS GRANTED")
		if got != "Access Granted" {
			t.Errorf("expected Access Granted, got %s", got)
		}
	})

	t.Run("already exists triggers access granted", func(t *testing.T) {
		got := FilterHTML("Account Already Exists")
		if got != "Access Granted" {
			t.Errorf("expected Access Granted, got %s", got)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		got := FilterHTML("")
		if got != "" {
			t.Errorf("expected empty, got %s", got)
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
	t.Run("success on first attempt", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Second,
		}
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
		cfg := RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Second,
		}
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
		cfg := RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Second,
		}
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
		cfg := RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Second,
		}
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

	t.Run("single attempt", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts: 1,
			BaseDelay:   time.Millisecond,
			MaxDelay:    time.Second,
		}
		fn := func() (string, error) {
			return "ok", nil
		}

		res, err := Retry(context.Background(), cfg, fn)
		if err != nil {
			t.Fatal(err)
		}
		if res != "ok" {
			t.Fatalf("expected ok, got %s", res)
		}
	})

	t.Run("backoff capped at max delay", func(t *testing.T) {
		cfg := RetryConfig{
			MaxAttempts: 4,
			BaseDelay:   50 * time.Millisecond,
			MaxDelay:    80 * time.Millisecond,
		}
		start := time.Now()
		calls := 0

		fn := func() (string, error) {
			calls++
			return "", errors.New("fail")
		}

		Retry(context.Background(), cfg, fn)

		elapsed := time.Since(start)
		expected := 50*time.Millisecond + 80*time.Millisecond + 80*time.Millisecond
		if elapsed >= expected+200*time.Millisecond {
			t.Fatalf("backoff not capped at max delay, took %v (expected ~%v)", elapsed, expected)
		}
	})

	t.Run("context timeout during backoff", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		cfg := RetryConfig{
			MaxAttempts: 10,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    time.Second,
		}

		fn := func() (string, error) {
			return "", errors.New("fail")
		}

		_, err := Retry(ctx, cfg, fn)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	})
}
