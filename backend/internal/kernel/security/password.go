/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package security

import "golang.org/x/crypto/bcrypt"

// Hashing a password, for whoever is doing the hashing.
//
// Both planes do. A person signing into an organisation and an operator signing
// into the console are different accounts in different tables under different
// database roles, and the one thing they are not allowed to differ on is how a
// password is stored — two answers to that question is one deployment where
// half the passwords are weaker and nothing says which half.
//
// So it is here rather than in either plane: the cost is a single decision, and
// changing it changes it for everybody.

// HashPassword returns a bcrypt hash at the library's default cost.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash reports whether the password is the one behind the hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
