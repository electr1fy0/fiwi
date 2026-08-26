package main

import "fmt"

func Success(s string) string {
	return "\033[32m" + s + "\033[0m"
}

func Error(s string) string {
	return "\033[31m" + s + "\033[0m"
}

func Warning(s string) string {
	return "\033[33m" + s + "\033[0m"
}

func Info(s string) string {
	return "\033[34m" + s + "\033[0m"
}

func Bold(s string) string {
	return "\033[1m" + s + "\033[0m"
}

func PrintLoginResult(s string) {
	switch s {
	case "Access Granted":
		fmt.Println(Success("\u2713 " + s))
	case "Already logged in":
		fmt.Println(Info("\u2139 " + s))
	case "Invalid credentials":
		fmt.Println(Error("\u2717 " + s))
	default:
		fmt.Println(Warning("\u003f ") + s)
	}
}

func PrintError(err error) {
	fmt.Println(Error("\u2717 Error: ") + err.Error())
}
