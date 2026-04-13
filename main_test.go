package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
	resp, err := Login(client, server.URL, "testuser", "testpass")
	if err != nil {
		t.Fatalf("unexpcted error: %v", err)
	}

	if resp != "login successful" {
		t.Errorf("unexpcted response: %s", resp)
	}
}

func TestLogin_NetworkError(t *testing.T) {
	client := &http.Client{}

	_, err := Login(client, "http://invalid-url", "u", "p")
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

	_, err := Login(client, server.URL, "u", "p")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
