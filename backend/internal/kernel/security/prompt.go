/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package security

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// Asking somebody at a terminal to choose a password.
//
// Here for the same reason HashPassword is: both planes bootstrap an account
// from a command — the console's first operator and a deployment's first
// organisation administrator — and a second copy of this would be a second
// answer to how a password is taken, in the one place where taking it wrongly
// leaves it in a shell history for ever.

// ReadNewPassword asks for a password twice on the terminal and never echoes
// it. minLength is stated before the first prompt rather than after the second:
// learning a requirement by being refused wastes somebody's time at exactly the
// wrong moment — the first minute of standing a deployment up.
//
// It reads from the terminal rather than from a flag or an environment
// variable, both of which leave the password in a shell history, a process
// list, or a container's inspect output — places it outlives the command in.
func ReadNewPassword(minLength int) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("this command needs a terminal to read the password from " +
			"(use `docker exec -it`, not `docker exec`)")
	}

	fmt.Printf("Choose a password of at least %d characters.\n", minLength)
	fmt.Print("Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("could not read the password: %w", err)
	}
	fmt.Print("Again: ")
	second, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("could not read the password: %w", err)
	}
	if string(first) != string(second) {
		return "", errors.New("the two passwords were not the same")
	}
	if len([]rune(string(first))) < minLength {
		return "", fmt.Errorf("the password must be at least %d characters", minLength)
	}
	return string(first), nil
}
