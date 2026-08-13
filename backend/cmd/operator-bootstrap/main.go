/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Creates the first operator account for the control plane. There is no web
 * registration for the console, on purpose: see internal/platform/controlplane.
 */

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/controlplane"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"
)

func main() {
	email := flag.String("email", "", "the operator's e-mail address")
	name := flag.String("name", "", "the operator's name")
	role := flag.String("role", string(controlplane.RoleSuperadmin),
		"superadmin, operator, support or auditor")
	flag.Usage = usage
	flag.Parse()

	if *email == "" || *name == "" {
		flag.Usage()
		os.Exit(2)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fail("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fail("could not connect to the database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		fail("the database is not reachable: %v", err)
	}

	password, err := readPassword()
	if err != nil {
		fail("%v", err)
	}

	operator, enrolment, err := controlplane.CreateOperator(ctx, db, controlplane.NewOperator{
		Email:    *email,
		Name:     *name,
		Role:     controlplane.Role(*role),
		Password: password,
	})
	if errors.Is(err, controlplane.ErrOperatorExists) {
		fail("an operator with that address already exists")
	}
	if err != nil {
		fail("%v", err)
	}

	fmt.Printf("\nOperator created: %s (%s)\n\n", operator.Email, operator.Role)
	fmt.Println("Add this to an authenticator application — 1Password, Aegis, Google Authenticator:")
	fmt.Printf("\n  secret: %s\n  uri:    %s\n\n", enrolment.Secret, enrolment.URI)
	fmt.Println("The account cannot sign in until a code from it is confirmed below.")

	// Looping rather than exiting on a wrong code: the alternative is an
	// account that exists, cannot sign in, and cannot be created again because
	// the address is taken — which is a support call on the day the console is
	// first set up.
	reader := bufio.NewReader(os.Stdin)
	for attempt := 0; attempt < 3; attempt++ {
		fmt.Print("Code from the authenticator: ")
		code, err := reader.ReadString('\n')
		if err != nil {
			fail("could not read the code: %v", err)
		}
		err = controlplane.ConfirmSecondFactor(ctx, db, operator.ID, strings.TrimSpace(code))
		if err == nil {
			fmt.Printf("\nDone. %s can sign in at the control plane.\n", operator.Email)
			return
		}
		fmt.Fprintf(os.Stderr, "  %v\n", err)
	}

	fail("the authenticator was not confirmed; the account exists but cannot sign in.\n"+
		"Confirm it later by running this command again against the same address, "+
		"or remove the row and start over:\n"+
		"  DELETE FROM operator_accounts WHERE lower(email) = lower('%s');", operator.Email)
}

// readPassword asks twice and never echoes.
//
// It reads from the terminal rather than from a flag or an environment
// variable, both of which leave the password in a shell history, a process
// list, or a container's inspect output — places it outlives the command in.
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("this command needs a terminal to read the password from " +
			"(use `docker exec -it`, not `docker exec`)")
	}

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
	return string(first), nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `operator-bootstrap — create a control plane operator.

The console has no sign-up screen. The first account is made here, by somebody
who already holds the database credentials, and every account after it is made
from the console by a superadmin.

Usage:
  DATABASE_URL=... operator-bootstrap -email you@example.mn -name "Your Name" [-role superadmin]

In production, with the compose stack running:
  docker exec -it gerege_nexus_api /app/operator-bootstrap -email ... -name "..."

Roles: superadmin (everything), operator (daily work), support (people),
auditor (read-only). See docs/CONTROL_PLANE.md.

`)
	flag.PrintDefaults()
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "operator-bootstrap: "+format+"\n", args...)
	os.Exit(1)
}
