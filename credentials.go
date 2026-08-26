package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CredentialProfile struct {
	Name     string `json:"name"`
	UserID   string `json:"userID"`
	Password string `json:"password"`
}

type CredentialStore struct {
	Active   string              `json:"active"`
	Profiles []CredentialProfile `json:"profiles"`
}

func ResolveCredentials(envUser, envPass string, fileData []byte) (string, string) {
	if envUser != "" && envPass != "" {
		return strings.TrimSpace(envUser), strings.TrimSpace(envPass)
	}

	store, err := decodeStore(fileData)
	if err != nil {
		return "", ""
	}

	for _, p := range store.Profiles {
		if p.Name == store.Active {
			return strings.TrimSpace(p.UserID), strings.TrimSpace(p.Password)
		}
	}

	if len(store.Profiles) > 0 {
		return strings.TrimSpace(store.Profiles[0].UserID), strings.TrimSpace(store.Profiles[0].Password)
	}

	return "", ""
}

func decodeStore(data []byte) (*CredentialStore, error) {
	if len(data) == 0 {
		return &CredentialStore{}, nil
	}

	var store CredentialStore
	if err := json.Unmarshal(data, &store); err == nil {
		if len(store.Profiles) > 0 {
			return &store, nil
		}
	}

	var old struct {
		UserID   string `json:"userID"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return nil, err
	}

	return &CredentialStore{
		Active: "default",
		Profiles: []CredentialProfile{
			{Name: "default", UserID: strings.TrimSpace(old.UserID), Password: strings.TrimSpace(old.Password)},
		},
	}, nil
}

func CredentialsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".fiwi"), nil
}

func LoadCredentialStore(path string) (*CredentialStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeStore(data)
}

func SaveCredentialStore(path string, store *CredentialStore) error {
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ClearCredentialsFile(path string) error {
	return os.Remove(path)
}

func (s *CredentialStore) ActiveProfile() *CredentialProfile {
	for i := range s.Profiles {
		if s.Profiles[i].Name == s.Active {
			return &s.Profiles[i]
		}
	}
	return nil
}

func (s *CredentialStore) FindProfile(name string) *CredentialProfile {
	for i := range s.Profiles {
		if s.Profiles[i].Name == name {
			return &s.Profiles[i]
		}
	}
	return nil
}

func (s *CredentialStore) AddProfile(name, userID, password string) error {
	if s.FindProfile(name) != nil {
		return fmt.Errorf("profile %q already exists", name)
	}
	s.Profiles = append(s.Profiles, CredentialProfile{
		Name: name, UserID: userID, Password: password,
	})
	if s.Active == "" {
		s.Active = name
	}
	return nil
}

func (s *CredentialStore) RemoveProfile(name string) error {
	if len(s.Profiles) <= 1 {
		return fmt.Errorf("cannot remove the only profile")
	}
	idx := -1
	for i, p := range s.Profiles {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("profile %q not found", name)
	}
	s.Profiles = append(s.Profiles[:idx], s.Profiles[idx+1:]...)
	if s.Active == name {
		s.Active = s.Profiles[0].Name
	}
	return nil
}

func (s *CredentialStore) SetActive(name string) error {
	if s.FindProfile(name) == nil {
		return fmt.Errorf("profile %q not found", name)
	}
	s.Active = name
	return nil
}

func (s *CredentialStore) ProfileNames() []string {
	names := make([]string, len(s.Profiles))
	for i, p := range s.Profiles {
		names[i] = p.Name
	}
	return names
}
