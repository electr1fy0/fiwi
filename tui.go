package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

func ShowCredentialManager() {
	for {
		var action string

		selectForm := huh.NewSelect[string]().
			Title("fiwi Credential Manager").
			Options(
				huh.NewOption("View Profiles", "view"),
				huh.NewOption("Add Profile", "add"),
				huh.NewOption("Edit Profile", "edit"),
				huh.NewOption("Set Active Profile", "active"),
				huh.NewOption("Delete Profile", "delete"),
				huh.NewOption("Test Login", "test"),
				huh.NewOption("Exit", "exit"),
			).
			Value(&action)

		err := selectForm.Run()
		if err != nil {
			return
		}

		switch action {
		case "view":
			viewProfiles()
		case "add":
			addProfile()
		case "edit":
			editProfile()
		case "active":
			setActiveProfile()
		case "delete":
			deleteProfile()
		case "test":
			testLoginTUI()
		case "exit":
			return
		}
	}
}

func getCredentialPath() string {
	path, err := CredentialsFilePath()
	if err != nil {
		return ""
	}
	return path
}

func loadStoreOrMessage() *CredentialStore {
	path := getCredentialPath()
	if path == "" {
		fmt.Println(Error("Cannot determine home directory"))
		return nil
	}

	store, err := LoadCredentialStore(path)
	if err != nil {
		fmt.Println(Error("No stored credentials found"))
		return nil
	}
	return store
}

func saveStore(store *CredentialStore) bool {
	path := getCredentialPath()
	if path == "" {
		fmt.Println(Error("Cannot determine home directory"))
		return false
	}
	if err := SaveCredentialStore(path, store); err != nil {
		fmt.Println(Error("Failed to save: " + err.Error()))
		return false
	}
	return true
}

func viewProfiles() {
	store := loadStoreOrMessage()
	if store == nil {
		pause()
		return
	}

	if len(store.Profiles) == 0 {
		fmt.Println(Warning("No profiles. Use Add Profile to create one."))
		pause()
		return
	}

	fmt.Println()
	fmt.Println(Bold("  Profiles:"))
	for _, p := range store.Profiles {
		marker := " "
		if p.Name == store.Active {
			marker = ">"
		}
		masked := strings.Repeat("*", len(p.Password))
		fmt.Printf("  %s %s  %s / %s\n", marker, Info(p.Name), p.UserID, masked)
	}
	fmt.Println()
	pause()
}

func addProfile() {
	var name, username, password string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile Name").
				Value(&name).
				Prompt("> "),
			huh.NewInput().
				Title("Username").
				Value(&username).
				Prompt("> "),
			huh.NewInput().
				Title("Password").
				Value(&password).
				EchoMode(huh.EchoModePassword).
				Prompt("> "),
		),
	)

	err := form.Run()
	if err != nil {
		return
	}

	name = strings.TrimSpace(name)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if name == "" || username == "" || password == "" {
		fmt.Println(Warning("All fields are required"))
		pause()
		return
	}

	path := getCredentialPath()
	store, err := LoadCredentialStore(path)
	if err != nil {
		store = &CredentialStore{}
	}

	if err := store.AddProfile(name, username, password); err != nil {
		fmt.Println(Error(err.Error()))
		pause()
		return
	}

	if !saveStore(store) {
		return
	}

	fmt.Println(Success(fmt.Sprintf("Profile %q added", name)))
	pause()
}

func editProfile() {
	store := loadStoreOrMessage()
	if store == nil {
		return
	}

	if len(store.Profiles) == 0 {
		fmt.Println(Warning("No profiles to edit"))
		pause()
		return
	}

	names := store.ProfileNames()
	var selected string

	selectForm := huh.NewSelect[string]().
		Title("Select profile to edit").
		Options(huh.NewOptions(names...)...).
		Value(&selected)

	if err := selectForm.Run(); err != nil {
		return
	}

	profile := store.FindProfile(selected)
	if profile == nil {
		return
	}

	username := profile.UserID
	password := profile.Password

	editForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&username).
				Prompt("> "),
			huh.NewInput().
				Title("Password").
				Value(&password).
				EchoMode(huh.EchoModePassword).
				Prompt("> "),
		),
	)

	if err := editForm.Run(); err != nil {
		fmt.Println(Warning("Edit cancelled"))
		return
	}

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		fmt.Println(Warning("Both fields are required"))
		return
	}

	profile.UserID = username
	profile.Password = password

	if !saveStore(store) {
		return
	}

	fmt.Println(Success(fmt.Sprintf("Profile %q updated", selected)))
	pause()
}

func setActiveProfile() {
	store := loadStoreOrMessage()
	if store == nil {
		return
	}

	if len(store.Profiles) == 0 {
		fmt.Println(Warning("No profiles"))
		pause()
		return
	}

	names := store.ProfileNames()
	var selected string

	selectForm := huh.NewSelect[string]().
		Title("Select active profile").
		Options(huh.NewOptions(names...)...).
		Value(&selected)

	if err := selectForm.Run(); err != nil {
		return
	}

	if err := store.SetActive(selected); err != nil {
		fmt.Println(Error(err.Error()))
		pause()
		return
	}

	if !saveStore(store) {
		return
	}

	fmt.Println(Success(fmt.Sprintf("Active profile set to %q", selected)))
	pause()
}

func deleteProfile() {
	store := loadStoreOrMessage()
	if store == nil {
		return
	}

	if len(store.Profiles) <= 1 {
		fmt.Println(Warning("Cannot delete the only profile. Use Clear Credentials to start fresh."))
		pause()
		return
	}

	names := store.ProfileNames()
	var selected string

	selectForm := huh.NewSelect[string]().
		Title("Select profile to delete").
		Options(huh.NewOptions(names...)...).
		Value(&selected)

	if err := selectForm.Run(); err != nil {
		return
	}

	var confirmed bool
	confirmForm := huh.NewConfirm().
		Title(fmt.Sprintf("Delete profile %q?", selected)).
		Description("This cannot be undone").
		Value(&confirmed)

	if err := confirmForm.Run(); err != nil || !confirmed {
		return
	}

	if err := store.RemoveProfile(selected); err != nil {
		fmt.Println(Error(err.Error()))
		pause()
		return
	}

	if !saveStore(store) {
		return
	}

	fmt.Println(Success(fmt.Sprintf("Profile %q deleted", selected)))
	pause()
}

func testLoginTUI() {
	store := loadStoreOrMessage()
	if store == nil {
		return
	}

	active := store.ActiveProfile()
	if active == nil {
		fmt.Println(Error("No active profile. Add one first."))
		pause()
		return
	}

	fmt.Println(Info(fmt.Sprintf("Testing profile %q... (this may take up to 30s)", active.Name)))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   1500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}

	result, loginErr := Retry(ctx, cfg, func() (string, error) {
		return LoginWithCtx(ctx, http.DefaultClient, URL, active.UserID, active.Password)
	})

	fmt.Println()
	if loginErr != nil {
		PrintError(loginErr)
	} else {
		PrintLoginResult(FilterHTML(result))
	}

	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func pause() {
	fmt.Print("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
