package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCredentials_EnvPriority(t *testing.T) {
	store := &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "fileUser", Password: "filePass"},
		},
	}
	u, p := ResolveCredentials("envUser", "envPass", marshalStore(t, store))
	if u != "envUser" || p != "envPass" {
		t.Fatalf("expected env creds, got %q %q", u, p)
	}
}

func TestResolveCredentials_StoreFallback(t *testing.T) {
	store := &CredentialStore{
		Active: "work",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "defaultUser", Password: "defaultPass"},
			{Name: "work", UserID: "workUser", Password: "workPass"},
		},
	}
	u, p := ResolveCredentials("", "", marshalStore(t, store))
	if u != "workUser" || p != "workPass" {
		t.Fatalf("expected active profile creds, got %q %q", u, p)
	}
}

func TestResolveCredentials_EmptyStore(t *testing.T) {
	u, p := ResolveCredentials("", "", nil)
	if u != "" || p != "" {
		t.Fatalf("expected empty, got %q %q", u, p)
	}
}

func TestResolveCredentials_OldFormat(t *testing.T) {
	u, p := ResolveCredentials("", "", []byte(`{"userID":"oldUser","password":"oldPass"}`))
	if u != "oldUser" || p != "oldPass" {
		t.Fatalf("expected old format creds, got %q %q", u, p)
	}
}

func TestResolveCredentials_MalformedJSON(t *testing.T) {
	u, p := ResolveCredentials("", "", []byte(`not json`))
	if u != "" || p != "" {
		t.Fatalf("expected empty for malformed JSON, got %q %q", u, p)
	}
}

func TestResolveCredentials_EnvWhitespace(t *testing.T) {
	u, p := ResolveCredentials("  envUser  ", "  envPass  ", nil)
	if u != "envUser" || p != "envPass" {
		t.Fatalf("expected trimmed env creds, got %q %q", u, p)
	}
}

func TestResolveCredentials_PartialEnvFallsbackToStore(t *testing.T) {
	store := &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "fileUser", Password: "filePass"},
		},
	}
	u, p := ResolveCredentials("", "nonempty", marshalStore(t, store))
	if u != "fileUser" || p != "filePass" {
		t.Fatalf("expected store creds (partial env ignored), got %q %q", u, p)
	}
}

func TestResolveCredentials_ActiveNotFound(t *testing.T) {
	store := &CredentialStore{
		Active: "nonexistent",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "user", Password: "pass"},
		},
	}
	u, p := ResolveCredentials("", "", marshalStore(t, store))
	if u != "user" || p != "pass" {
		t.Fatalf("expected first profile when active not found, got %q %q", u, p)
	}
}

func TestSaveAndLoadCredentialStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fiwi")

	store := &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "testuser", Password: "testpass"},
		},
	}
	if err := SaveCredentialStore(path, store); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCredentialStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Active != "default" {
		t.Fatalf("expected active 'default', got %q", loaded.Active)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].UserID != "testuser" || loaded.Profiles[0].Password != "testpass" {
		t.Fatalf("got %+v", loaded.Profiles[0])
	}
}

func TestLoadCredentialStore_OldFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fiwi")

	os.WriteFile(path, []byte(`{"userID":"oldUser","password":"oldPass"}`), 0644)

	store, err := LoadCredentialStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if store.Active != "default" {
		t.Fatalf("expected active 'default' after migration, got %q", store.Active)
	}
	if len(store.Profiles) != 1 {
		t.Fatalf("expected 1 profile after migration, got %d", len(store.Profiles))
	}
	if store.Profiles[0].Name != "default" {
		t.Fatalf("expected migrated profile name 'default', got %q", store.Profiles[0].Name)
	}
	if store.Profiles[0].UserID != "oldUser" || store.Profiles[0].Password != "oldPass" {
		t.Fatalf("got %+v", store.Profiles[0])
	}
}

func TestLoadCredentialStore_FileNotFound(t *testing.T) {
	_, err := LoadCredentialStore("/nonexistent/path/for/fiwi/test")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestClearCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fiwi")

	SaveCredentialStore(path, &CredentialStore{Active: "d", Profiles: []CredentialProfile{{Name: "d", UserID: "u", Password: "p"}}})

	if err := ClearCredentialsFile(path); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file still exists after clear")
	}
}

func TestClearCredentialsFile_NotExist(t *testing.T) {
	err := ClearCredentialsFile("/nonexistent/path/for/fiwi/test")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestCredentialsFilePath(t *testing.T) {
	path, err := CredentialsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %s", path)
	}
	if filepath.Base(path) != ".fiwi" {
		t.Fatalf("expected .fiwi file, got %s", filepath.Base(path))
	}
}

func TestActiveProfile(t *testing.T) {
	store := &CredentialStore{
		Active: "work",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "u1", Password: "p1"},
			{Name: "work", UserID: "u2", Password: "p2"},
		},
	}

	active := store.ActiveProfile()
	if active == nil {
		t.Fatal("expected active profile")
	}
	if active.Name != "work" || active.UserID != "u2" {
		t.Fatalf("expected work profile, got %+v", active)
	}
}

func TestActiveProfile_NotFound(t *testing.T) {
	store := &CredentialStore{
		Active:   "nonexistent",
		Profiles: []CredentialProfile{{Name: "default", UserID: "u", Password: "p"}},
	}

	active := store.ActiveProfile()
	if active != nil {
		t.Fatal("expected nil when active not found")
	}
}

func TestFindProfile(t *testing.T) {
	store := &CredentialStore{
		Profiles: []CredentialProfile{
			{Name: "a", UserID: "u1", Password: "p1"},
			{Name: "b", UserID: "u2", Password: "p2"},
		},
	}

	if p := store.FindProfile("a"); p == nil || p.UserID != "u1" {
		t.Fatalf("expected profile a, got %v", p)
	}
	if p := store.FindProfile("b"); p == nil || p.UserID != "u2" {
		t.Fatalf("expected profile b, got %v", p)
	}
	if p := store.FindProfile("c"); p != nil {
		t.Fatalf("expected nil for nonexistent, got %v", p)
	}
}

func TestAddProfile(t *testing.T) {
	store := &CredentialStore{}
	if err := store.AddProfile("work", "u", "p"); err != nil {
		t.Fatal(err)
	}
	if len(store.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(store.Profiles))
	}
	if store.Active != "work" {
		t.Fatalf("expected active to be 'work', got %q", store.Active)
	}
}

func TestAddProfile_Duplicate(t *testing.T) {
	store := &CredentialStore{}
	store.AddProfile("work", "u", "p")
	err := store.AddProfile("work", "u2", "p2")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestAddProfile_ActiveNotOverridden(t *testing.T) {
	store := &CredentialStore{Active: "existing"}
	store.AddProfile("existing", "u", "p")
	store.AddProfile("new", "u2", "p2")
	if store.Active != "existing" {
		t.Fatalf("expected active to stay 'existing', got %q", store.Active)
	}
}

func TestRemoveProfile(t *testing.T) {
	store := &CredentialStore{
		Active: "a",
		Profiles: []CredentialProfile{
			{Name: "a", UserID: "u1", Password: "p1"},
			{Name: "b", UserID: "u2", Password: "p2"},
		},
	}

	if err := store.RemoveProfile("a"); err != nil {
		t.Fatal(err)
	}
	if len(store.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(store.Profiles))
	}
	if store.Active != "b" {
		t.Fatalf("expected active to switch to 'b', got %q", store.Active)
	}
}

func TestRemoveProfile_LastProfile(t *testing.T) {
	store := &CredentialStore{
		Active:   "a",
		Profiles: []CredentialProfile{{Name: "a", UserID: "u", Password: "p"}},
	}

	err := store.RemoveProfile("a")
	if err == nil {
		t.Fatal("expected error when removing the only profile")
	}
}

func TestRemoveProfile_NotFound(t *testing.T) {
	store := &CredentialStore{
		Profiles: []CredentialProfile{{Name: "a", UserID: "u", Password: "p"}},
	}

	err := store.RemoveProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestSetActive(t *testing.T) {
	store := &CredentialStore{
		Active: "a",
		Profiles: []CredentialProfile{
			{Name: "a", UserID: "u1", Password: "p1"},
			{Name: "b", UserID: "u2", Password: "p2"},
		},
	}

	if err := store.SetActive("b"); err != nil {
		t.Fatal(err)
	}
	if store.Active != "b" {
		t.Fatalf("expected active 'b', got %q", store.Active)
	}
}

func TestSetActive_NotFound(t *testing.T) {
	store := &CredentialStore{
		Profiles: []CredentialProfile{{Name: "a", UserID: "u", Password: "p"}},
	}

	err := store.SetActive("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProfileNames(t *testing.T) {
	store := &CredentialStore{
		Profiles: []CredentialProfile{
			{Name: "c", UserID: "u1", Password: "p1"},
			{Name: "a", UserID: "u2", Password: "p2"},
			{Name: "b", UserID: "u3", Password: "p3"},
		},
	}

	names := store.ProfileNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "c" || names[1] != "a" || names[2] != "b" {
		t.Fatalf("unexpected order: %v", names)
	}
}

func TestMultiProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fiwi")

	store := &CredentialStore{
		Active: "work",
		Profiles: []CredentialProfile{
			{Name: "home", UserID: "homeUser", Password: "homePass"},
			{Name: "work", UserID: "workUser", Password: "workPass"},
			{Name: "cafe", UserID: "cafeUser", Password: "cafePass"},
		},
	}

	if err := SaveCredentialStore(path, store); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCredentialStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Active != "work" {
		t.Fatalf("expected active 'work', got %q", loaded.Active)
	}
	if len(loaded.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(loaded.Profiles))
	}

	work := loaded.FindProfile("work")
	if work == nil || work.UserID != "workUser" || work.Password != "workPass" {
		t.Fatalf("work profile wrong: %+v", work)
	}
}

func TestMultiProfileResolve(t *testing.T) {
	store := &CredentialStore{
		Active: "cafe",
		Profiles: []CredentialProfile{
			{Name: "home", UserID: "homeUser", Password: "homePass"},
			{Name: "work", UserID: "workUser", Password: "workPass"},
			{Name: "cafe", UserID: "cafeUser", Password: "cafePass"},
		},
	}

	data := marshalStore(t, store)

	u, p := ResolveCredentials("", "", data)
	if u != "cafeUser" || p != "cafePass" {
		t.Fatalf("expected cafe creds, got %q %q", u, p)
	}

	store.Active = "home"
	data = marshalStore(t, store)

	u, p = ResolveCredentials("", "", data)
	if u != "homeUser" || p != "homePass" {
		t.Fatalf("expected home creds, got %q %q", u, p)
	}
}

func TestSaveWithSpecialCharacters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fiwi")

	store := &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: "user@domain.com", Password: "p@ssw0rd!#$%"},
		},
	}

	if err := SaveCredentialStore(path, store); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCredentialStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Profiles[0].UserID != "user@domain.com" || loaded.Profiles[0].Password != "p@ssw0rd!#$%" {
		t.Fatalf("got %+v", loaded.Profiles[0])
	}
}

func marshalStore(t *testing.T, store *CredentialStore) []byte {
	t.Helper()
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
