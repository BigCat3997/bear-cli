package prompt

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func TextInput(label string) (string, error) {
	fmt.Print(label)
	var input string
	_, err := fmt.Scanln(&input)
	return strings.TrimSpace(input), err
}

func PasswordInput(label string) (string, error) {
	fmt.Print(label)
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
